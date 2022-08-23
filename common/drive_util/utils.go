package drive_util

import (
	"context"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	"path"
	"strconv"
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
	file, e := ioutil.TempFile(tempDir, "drive-copy")
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

func GetIContentReader(ctx context.Context, content types.IContentReader) (io.ReadCloser, error) {
	u, e := content.GetURL(ctx)
	if e == nil {
		return GetURL(ctx, u.URL, u.Header)
	}
	return content.GetReader(ctx)
}

func CopyIContent(ctx types.TaskCtx, content types.IContentReader, dst io.Writer) error {
	reader, e := GetIContentReader(ctx, content)
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
	reader, e := GetIContentReader(ctx, content)
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
	reader, e := content.GetReader(ctx)
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

type EntryNode struct {
	types.IEntry
	children []EntryNode
}

type DoCopy = func(from types.IEntry, driveTo types.IDrive, to string, ctx types.TaskCtx) error
type CopyCallback = func(entry types.IEntry, allProcessed bool, ctx types.TaskCtx) error

func buildEntriesTree(ctx types.TaskCtx, entry types.IEntry, bytesProgress bool) (EntryNode, error) {
	if e := ctx.Err(); e != nil {
		return EntryNode{}, e
	}
	if bytesProgress {
		if entry.Type().IsFile() {
			ctx.Total(entry.Size(), false)
		}
	} else {
		ctx.Total(1, false)
	}
	r := EntryNode{entry, nil}
	if entry.Type().IsFile() {
		return r, nil
	}
	entries, e := entry.Drive().List(ctx, entry.Path())
	if e != nil {
		return r, e
	}
	children := make([]EntryNode, len(entries))
	for i, e := range entries {
		node, ee := buildEntriesTree(ctx, e, bytesProgress)
		if ee != nil {
			return r, ee
		}
		children[i] = node
	}
	r.children = children
	return r, nil
}

func BuildEntriesTree(ctx types.TaskCtx, root types.IEntry, bytesProgress bool) (EntryNode, error) {
	if ctx == nil {
		ctx = task.DummyContext()
	}
	return buildEntriesTree(ctx, root, bytesProgress)
}

func flattenEntriesTree(root EntryNode, result []EntryNode) []EntryNode {
	result = append(result, root)
	if root.children != nil {
		for _, e := range root.children {
			result = flattenEntriesTree(e, result)
		}
	}
	return result
}

func FlattenEntriesTree(root EntryNode) []EntryNode {
	result := make([]EntryNode, 0)
	return flattenEntriesTree(root, result)
}

func copyAll(ctx types.TaskCtx, entry EntryNode, driveTo types.IDrive, to string,
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
	if entry.Type().IsDir() {
		dirCreate := false
		if dstExists {
			if dstType.IsFile() {
				return false, err.NewNotAllowedMessageError(
					i18n.T("drive.copy_type_mismatch1", entry.Path(), to))
			}
		} else {
			_, e := driveTo.MakeDir(ctx, to)
			if e != nil {
				return false, e
			}
			dirCreate = true
		}
		if entry.children != nil {
			for _, e := range entry.children {
				r, ee := copyAll(ctx, e, driveTo, utils.CleanPath(path.Join(to, utils.PathBase(e.Path()))),
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

	if entry.Type().IsFile() {
		if dstExists {
			if dstType.IsDir() {
				return false, err.NewNotAllowedMessageError(
					i18n.T("drive.copy_type_mismatch2", entry.Path(), to))
			}
		}

		if e := doCopy(entry.IEntry, driveTo, to, ctx); e != nil {
			return false, e
		}
	}
	if e := after(entry, allProcessed, ctx); e != nil {
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

func GetURL(ctx context.Context, u string, header types.SM) (io.ReadCloser, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return nil, e
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return nil, e
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, err.NewRemoteApiError(resp.StatusCode,
			i18n.T("util.request_failed", strconv.Itoa(resp.StatusCode)))
	}
	return resp.Body, nil
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
