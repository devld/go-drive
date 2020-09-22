package common

import (
	"fmt"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	"path"
	"sort"
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

func pathPermissionLess(a, b types.PathPermission) bool {
	if a.Depth != b.Depth {
		return a.Depth > b.Depth
	}
	if a.IsForAnonymous() {
		if b.IsForAnonymous() {
			return a.Policy < b.Policy
		} else {
			return false
		}
	} else {
		if b.IsForAnonymous() {
			return true
		} else {
			if a.IsForUser() {
				if b.IsForUser() {
					return a.Policy < b.Policy
				} else {
					return true
				}
			} else {
				if b.IsForUser() {
					return false
				} else {
					return a.Policy < b.Policy
				}
			}
		}
	}
}

func ResolveAcceptedPermissions(items []types.PathPermission) types.Permission {
	sort.Slice(items, func(i, j int) bool { return pathPermissionLess(items[i], items[j]) })
	acceptedPermission := types.PermissionEmpty
	rejectedPermission := types.PermissionEmpty
	for _, item := range items {
		if item.IsAccept() {
			acceptedPermission |= item.Permission & ^rejectedPermission
		}
		if item.IsReject() {
			// acceptedPermission - ( item.Permission(reject) - acceptedPermission )
			acceptedPermission &= ^(item.Permission & (^acceptedPermission))
			rejectedPermission |= item.Permission
		}
	}
	return acceptedPermission
}

func Copy(dst io.Writer, src io.Reader, ctx task.Context) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		if ctx.Canceled() {
			return written, task.ErrorCanceled
		}
		w, err := io.CopyBuffer(dst, src, buf)
		if err != nil {
			break
		}
		if w == 0 {
			break
		}
		written += w
		ctx.Progress(written)
	}
	return
}

func CopyReaderToTempFile(reader io.Reader, ctx task.Context) (*os.File, error) {
	file, e := ioutil.TempFile("", "drive-copy")
	if e != nil {
		return nil, e
	}
	_, e = Copy(file, reader, ctx)
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

func GetIContentReader(content types.IContent) (io.ReadCloser, error) {
	url, _, e := content.GetURL()
	if e == nil {
		resp, e := http.Get(url)
		if e != nil {
			return nil, e
		}
		if resp.StatusCode != 200 {
			return nil, NewRemoteApiError(resp.StatusCode, "failed to copy file")
		}
		return resp.Body, nil
	}
	return content.GetReader()
}

func CopyIContentToTempFile(content types.IContent, ctx task.Context) (*os.File, error) {
	reader, e := GetIContentReader(content)
	if e != nil {
		return nil, e
	}
	return CopyReaderToTempFile(reader, ctx)
}

func DownloadIContent(content types.IContent, w http.ResponseWriter, req *http.Request) error {
	url, proxy, e := content.GetURL()
	if e == nil {
		if proxy {
			dest, e := url2.Parse(url)
			if e != nil {
				return e
			}
			proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
				r.URL = dest
				r.Host = dest.Host
				r.Header.Del("Referer")
				r.Header.Del("Authorization")
			}}

			defer func() {
				if i := recover(); i != http.ErrAbortHandler {
					panic(i)
				}
			}()

			proxy.ServeHTTP(w, req)
			return nil
		} else {
			w.WriteHeader(302)
			w.Header().Set("Location", url)
		}
		return e
	}
	reader, e := content.GetReader()
	if e != nil {
		return e
	}
	defer func() { _ = reader.Close() }()
	readSeeker, ok := reader.(io.ReadSeeker)
	if ok {
		http.ServeContent(
			w, req, content.Name(),
			Time(content.ModTime()),
			readSeeker)
		return nil
	}

	w.Header().Set("Content-Length", strconv.FormatInt(content.Size(), 10))
	_, e = io.Copy(w, reader)
	return e
}

// region copy all

type ctxWrapper struct {
	ctx task.Context
}

func (c ctxWrapper) Progress(int64) {
}

func (c ctxWrapper) Total(int64) {
}

func (c ctxWrapper) Canceled() bool {
	return c.ctx.Canceled()
}

type EntryNode struct {
	types.IEntry
	children []EntryNode
}

type DoCopy = func(from types.IEntry, driveTo types.IDrive, to string, ctx task.Context) error
type CopyCallback = func(entry types.IEntry, allProcessed bool, ctx task.Context) error

func buildEntriesTree(entry types.IEntry, total int, ctx task.Context) (EntryNode, error) {
	if ctx.Canceled() {
		return EntryNode{}, task.ErrorCanceled
	}
	r := EntryNode{entry, nil}
	if entry.Type().IsFile() {
		return r, nil
	}
	entries, e := entry.Drive().List(entry.Path())
	if e != nil {
		return r, e
	}
	children := make([]EntryNode, len(entries))
	total += len(entries)
	ctx.Total(int64(total))
	for i, e := range entries {
		node, err := buildEntriesTree(e, total, ctx)
		if err != nil {
			return r, err
		}
		children[i] = node
	}
	r.children = children
	return r, nil
}

func BuildEntriesTree(root types.IEntry, ctx task.Context) (EntryNode, error) {
	if ctx == nil {
		ctx = task.DummyContext()
	}
	ctx.Total(1)
	return buildEntriesTree(root, 1, ctx)
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

func copyAll(entry EntryNode, driveTo types.IDrive, to string, override bool,
	ctx task.Context, newParent bool, doCopy DoCopy, after CopyCallback) (int, bool, error) {
	processed := 0
	if ctx.Canceled() {
		return processed, false, task.ErrorCanceled
	}
	var dstType types.EntryType
	dstExists := false
	if newParent {
		dstExists = false
	} else {
		dst, e := driveTo.Get(to)
		if e != nil && !IsNotFoundError(e) {
			return processed, false, e
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
				return processed, false, NewNotAllowedMessageError(fmt.Sprintf(
					"dest '%s' is a file, but src '%s' is a dir", to, entry.Path()))
			}
		} else {
			_, e := driveTo.MakeDir(to)
			if e != nil {
				return processed, false, e
			}
			dirCreate = true
		}
		if entry.children != nil {
			for _, e := range entry.children {
				p, r, err := copyAll(e, driveTo, CleanPath(path.Join(to, e.Name())), override, ctx, dirCreate, doCopy, after)
				if err != nil {
					return processed, false, err
				}
				processed += p
				ctx.Progress(int64(processed))
				if !r {
					allProcessed = false
				}
			}
		}
	}

	if entry.Type().IsFile() {
		if dstExists {
			if dstType.IsDir() {
				return processed, false, NewNotAllowedMessageError(fmt.Sprintf(
					"dest '%s' is a dir, but src '%s' is a file", to, entry.Path()))
			}
			if !override {
				// skip
				return processed + 1, false, nil
			}
		}

		if e := doCopy(entry.IEntry, driveTo, to, ctxWrapper{ctx}); e != nil {
			return processed, false, e
		}
	}
	if e := after(entry, allProcessed, ctxWrapper{ctx}); e != nil {
		return processed, false, e
	}
	processed += 1
	ctx.Progress(int64(processed))
	return processed, allProcessed, nil
}

func CopyAll(entry types.IEntry, driveTo types.IDrive, to string, override bool,
	ctx task.Context, doCopy DoCopy, after CopyCallback) error {
	tree, err := BuildEntriesTree(entry, ctx)
	if err != nil {
		return err
	}
	if after == nil {
		after = func(entry types.IEntry, fullProcessed bool, ctx task.Context) error { return nil }
	}
	_, _, err = copyAll(tree, driveTo, to, override, ctx, false, doCopy, after)
	return err
}

func CopyEntry(from types.IEntry, driveTo types.IDrive, to string, ctx task.Context) error {
	content, ok := from.(types.IContent)
	if !ok {
		return NewNotAllowedMessageError(fmt.Sprintf("file '%s' is not readable", from.Path()))
	}
	file, e := CopyIContentToTempFile(content, ctx)
	if e != nil {
		return e
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, e = driveTo.Save(to, file, ctx)
	return e
}

// endregion
