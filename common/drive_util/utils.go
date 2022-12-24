package drive_util

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	"path"
	"strconv"

	"github.com/bmatcuk/doublestar/v4"
)

func GetIEntry(entry types.IEntry, test func(iEntry types.IEntry) bool) types.IEntry {
	if entry == nil {
		return nil
	}
	for {
		if test != nil && test(entry) {
			return entry
		}
		if wrapper, ok := entry.(types.IEntryWrapper); ok {
			entry = wrapper.GetIEntry()
		} else {
			break
		}
	}
	if test != nil {
		return nil
	}
	return entry
}

func UnwrapIEntry(entry types.IEntry) types.IEntry {
	for {
		ew, ok := entry.(types.IEntryWrapper)
		if !ok {
			return entry
		} else {
			entry = ew.GetIEntry()
		}
	}
}

func GetSelfEntry(d types.IDrive, entry types.IEntry) types.IEntry {
	return GetIEntry(entry, func(ee types.IEntry) bool { return ee.Drive() == d })
}

func Copy(ctx types.TaskCtx, dst io.Writer, src io.Reader) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		if e := ctx.Err(); e != nil {
			return written, e
		}
		w, ee := io.CopyBuffer(dst, src, buf)
		if ee != nil {
			break
		}
		if w == 0 {
			break
		}
		written += w
		ctx.Progress(w, false)
	}
	return
}

func CopyReaderToTempFile(ctx types.TaskCtx, reader io.Reader, tempDir string) (*os.File, error) {
	file, e := os.CreateTemp(tempDir, "drive-copy")
	if e != nil {
		return nil, e
	}
	_, e = Copy(ctx, file, reader)
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, e
	}
	_, e = file.Seek(0, 0)
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, e
	}
	return file, nil
}

func GetIContentReader(ctx context.Context, content types.IContentReader, start, size int64) (io.ReadCloser, error) {
	u, e := content.GetURL(ctx)
	if e == nil {
		return GetURL(ctx, u.URL, u.Header, start, size)
	}
	return content.GetReader(ctx, start, size)
}

func CopyIContent(ctx types.TaskCtx, content types.IContentReader, dst io.Writer) error {
	reader, e := GetIContentReader(ctx, content, -1, -1)
	if e != nil {
		return e
	}
	defer func() {
		_ = reader.Close()
	}()
	_, e = Copy(ctx, dst, reader)
	return e
}

func CopyIContentToTempFile(ctx types.TaskCtx, content types.IContentReader, tempDir string) (*os.File, error) {
	reader, e := GetIContentReader(ctx, content, -1, -1)
	if e != nil {
		return nil, e
	}
	defer func() {
		_ = reader.Close()
	}()
	return CopyReaderToTempFile(ctx, reader, tempDir)
}

func DownloadIContent(ctx context.Context, content types.IContent,
	w http.ResponseWriter, req *http.Request, forceProxy bool) error {
	u, e := content.GetURL(ctx)
	if e == nil {
		if u.Proxy || forceProxy || u.Header != nil {
			dest, e := url2.Parse(u.URL)
			if e != nil {
				return e
			}
			proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
				r.URL = dest
				r.Host = dest.Host
				r.Header.Del("Origin")
				r.Header.Del("Referer")
				r.Header.Del("Authorization")
				r.Header.Del(common.HeaderAuth)
				r.Header.Del("Cookie")
				if u.Header != nil {
					for k, v := range u.Header {
						r.Header.Set(k, v)
					}
				}
			}}

			defer func() {
				if i := recover(); i != nil && i != http.ErrAbortHandler {
					panic(i)
				}
			}()

			proxy.ServeHTTP(w, req)
			return nil
		} else {
			w.WriteHeader(http.StatusFound)
			w.Header().Set("Location", u.URL)
		}
		return nil
	}
	if !err.IsUnsupportedError(e) {
		return e
	}
	reader, e := content.GetReader(ctx, -1, -1)
	if e != nil {
		return e
	}
	defer func() { _ = reader.Close() }()
	readSeeker, ok := reader.(io.ReadSeeker)
	if ok {
		http.ServeContent(
			w, req, content.Name(),
			utils.Time(content.ModTime()),
			readSeeker)
		return nil
	}
	size := content.Size()
	if size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	if req.Method != http.MethodHead {
		_, e = io.Copy(w, reader)
	}
	return e
}

// region copy all

type EntryTreeNode struct {
	Entry    types.IEntry
	Children []EntryTreeNode
	Excluded bool
}

type DoCopy = func(from types.IEntry, driveTo types.IDrive, to string, ctx types.TaskCtx) error
type CopyCallback = func(entry types.IEntry, allProcessed bool, ctx types.TaskCtx) error

var ErrSkipDir = errors.New("skip")

func buildEntriesTree(ctx types.TaskCtx, entry types.IEntry, filter func(types.IEntry) (bool, error), bytesProgress bool) (*EntryTreeNode, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	dirSkipped := false
	filterMatched := true
	if filter != nil {
		ok, e := filter(entry)
		filterMatched = ok
		if e == ErrSkipDir {
			dirSkipped = true
			e = nil
		}
		if e != nil {
			return nil, e
		}
		if !ok && entry.Type().IsFile() {
			return nil, nil
		}
	}
	if bytesProgress {
		if entry.Type().IsFile() {
			ctx.Total(entry.Size(), false)
		}
	} else {
		ctx.Total(1, false)
	}
	r := &EntryTreeNode{entry, nil, !filterMatched}
	if entry.Type().IsFile() {
		return r, nil
	}
	if dirSkipped && !filterMatched {
		return nil, nil
	}
	if dirSkipped {
		return r, nil
	}
	entries, e := entry.Drive().List(ctx, entry.Path())
	if e != nil {
		return r, e
	}
	children := make([]EntryTreeNode, 0, len(entries))
	for _, e := range entries {
		node, ee := buildEntriesTree(ctx, e, filter, bytesProgress)
		if ee != nil {
			return r, ee
		}
		if node != nil {
			children = append(children, *node)
		}
	}
	r.Children = children
	return r, nil
}

func BuildEntriesTree(ctx types.TaskCtx, root types.IEntry, bytesProgress bool) (EntryTreeNode, error) {
	if ctx == nil {
		ctx = task.DummyContext()
	}
	r, e := buildEntriesTree(ctx, root, nil, bytesProgress)
	if e != nil {
		return EntryTreeNode{}, e
	}
	if r == nil {
		return EntryTreeNode{}, err.NewNotFoundMessageError("no matched entries")
	}
	return *r, nil
}

func FindEntries(ctx types.TaskCtx, root types.IDrive, pattern string, bytesProgress bool) ([]types.IEntry, error) {
	if pattern == "" {
		return nil, err.NewNotFoundMessageError("empty pattern")
	}
	if ctx == nil {
		ctx = task.DummyContext()
	}
	result := make([]types.IEntry, 0)
	dfs, e := NewDriveFS(root, "", nil)
	if e != nil {
		return nil, e
	}
	e = doublestar.GlobWalk(dfs, pattern, func(path string, d fs.DirEntry) error {
		entry, e := root.Get(ctx, path)
		if e != nil {
			return e
		}
		if bytesProgress {
			if entry.Type().IsFile() {
				ctx.Total(entry.Size(), false)
			}
		} else {
			ctx.Total(1, false)
		}
		result = append(result, entry)
		return nil
	}, doublestar.WithFailOnIOErrors())
	return result, e
}

func flattenEntriesTree(root EntryTreeNode, deepFirst bool, result []EntryTreeNode) []EntryTreeNode {
	if !deepFirst && !root.Excluded {
		result = append(result, root)
	}
	for _, e := range root.Children {
		result = flattenEntriesTree(e, deepFirst, result)
	}
	if deepFirst && !root.Excluded {
		result = append(result, root)
	}
	return result
}

func FlattenEntriesTree(root EntryTreeNode, deepFirst bool) []EntryTreeNode {
	result := make([]EntryTreeNode, 0)
	return flattenEntriesTree(root, deepFirst, result)
}

func VisitEntriesTree(root EntryTreeNode, visit func(e types.IEntry) error) error {
	e := visit(root.Entry)
	if e != nil {
		return e
	}
	for _, node := range root.Children {
		if e := VisitEntriesTree(node, visit); e != nil {
			return e
		}
	}
	return nil
}

func copyAll(ctx types.TaskCtx, entry EntryTreeNode, driveTo types.IDrive, to string,
	newParent bool, doCopy DoCopy, after CopyCallback) (bool, error) {
	if e := ctx.Err(); e != nil {
		return false, e
	}
	var dstType types.EntryType
	dstExists := false
	if newParent {
		dstExists = false
	} else {
		dst, e := driveTo.Get(ctx, to)
		if e != nil && !err.IsNotFoundError(e) {
			return false, e
		}
		dstExists = e == nil
		if dstExists {
			dstType = dst.Type()
		}
	}

	allProcessed := true
	if entry.Entry.Type().IsDir() {
		dirCreate := false
		if dstExists {
			if dstType.IsFile() {
				return false, err.NewNotAllowedMessageError(
					i18n.T("drive.copy_type_mismatch1", entry.Entry.Path(), to))
			}
		} else {
			_, e := driveTo.MakeDir(ctx, to)
			if e != nil {
				return false, e
			}
			dirCreate = true
		}
		if entry.Children != nil {
			for _, e := range entry.Children {
				r, ee := copyAll(ctx, e, driveTo, utils.CleanPath(path.Join(to, utils.PathBase(e.Entry.Path()))),
					dirCreate, doCopy, after)
				if ee != nil {
					return false, ee
				}
				if !r {
					allProcessed = false
				}
			}
		}
	}

	if entry.Entry.Type().IsFile() {
		if dstExists {
			if dstType.IsDir() {
				return false, err.NewNotAllowedMessageError(
					i18n.T("drive.copy_type_mismatch2", entry.Entry.Path(), to))
			}
		}

		if e := doCopy(entry.Entry, driveTo, to, ctx); e != nil {
			return false, e
		}
	}
	if e := after(entry.Entry, allProcessed, ctx); e != nil {
		return false, e
	}
	return allProcessed, nil
}

func CopyAll(ctx types.TaskCtx, entry types.IEntry, driveTo types.IDrive, to string,
	doCopy DoCopy, after CopyCallback) error {
	tree, e := BuildEntriesTree(ctx, entry, true)
	if e != nil {
		return e
	}
	if after == nil {
		after = func(entry types.IEntry, fullProcessed bool, ctx types.TaskCtx) error { return nil }
	}
	_, e = copyAll(ctx, tree, driveTo, to, false, doCopy, after)
	return e
}

func CopyEntry(ctx types.TaskCtx, from types.IEntry, driveTo types.IDrive, to string,
	override bool, tempDir string) error {
	file, e := CopyIContentToTempFile(task.DummyContext(), from, tempDir)
	if e != nil {
		return e
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()
	_, e = driveTo.Save(ctx, to, from.Size(), override, file)
	return e
}

// endregion

type progressReader struct {
	r   io.Reader
	ctx types.TaskCtx
}

func (p *progressReader) Read(b []byte) (n int, err error) {
	read, e := p.r.Read(b)
	if e == nil || e == io.EOF {
		p.ctx.Progress(int64(read), false)
	}
	return read, e
}

func ProgressReader(reader io.Reader, ctx types.TaskCtx) io.Reader {
	return &progressReader{r: reader, ctx: ctx}
}

func getURL(ctx context.Context, u string, header types.SM) (int, io.ReadCloser, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return 0, nil, e
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return 0, nil, e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return resp.StatusCode, nil, err.NewRemoteApiError(resp.StatusCode,
			i18n.T("util.request_failed", strconv.Itoa(resp.StatusCode)))
	}
	return resp.StatusCode, resp.Body, nil
}

func BuildRangeHeader(start, size int64) string {
	rangeStr := ""
	if start >= 0 {
		rangeStr = fmt.Sprintf("bytes=%d-", start)
		if size > 0 {
			rangeStr += fmt.Sprintf("%d", start+size-1)
		}
	}
	return rangeStr
}

func GetURL(ctx context.Context, u string, header types.SM, start, size int64) (io.ReadCloser, error) {
	rangeStr := BuildRangeHeader(start, size)
	if rangeStr != "" {
		header = utils.MapCopy(header, nil)
		header["Range"] = rangeStr
	}
	s, b, e := getURL(ctx, u, header)
	if e != nil {
		return nil, e
	}
	if rangeStr != "" && s != http.StatusPartialContent {
		return nil, err.NewUnsupportedMessageError("Range request not supported")
	}
	return b, nil
}

func NewURLContentReader(url string, headers types.SM, proxy bool) types.IContentReader {
	return &contentReaderImpl{url, headers, proxy}
}

type contentReaderImpl struct {
	url     string
	headers types.SM
	proxy   bool
}

func (t *contentReaderImpl) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	return GetURL(ctx, t.url, t.headers, start, size)
}

func (t *contentReaderImpl) GetURL(context.Context) (*types.ContentURL, error) {
	return &types.ContentURL{
		URL:    t.url,
		Header: t.headers,
		Proxy:  t.proxy,
	}, nil
}

func RequireFileNotExists(ctx context.Context, d types.IDrive, p string) (types.IEntry, error) {
	get, e := d.Get(ctx, p)
	if e == nil {
		return get, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	if !err.IsNotFoundError(e) {
		return nil, e
	}
	return nil, nil
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}

func LimitReadCloser(rc io.ReadCloser, n int64) io.ReadCloser {
	return limitedReadCloser{io.LimitReader(rc, n), rc}
}
