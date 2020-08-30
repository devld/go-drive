package common

import (
	"go-drive/common/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	fsPath "path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func FileExists(path string) (bool, error) {
	_, e := os.Stat(path)
	if os.IsNotExist(e) {
		return false, nil
	}
	return e == nil, e
}

func IsDir(path string) (bool, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return false, e
	}
	return stat.IsDir(), nil
}

func IsRootPath(path string) bool {
	return path == ""
}

func CleanPath(path string) string {
	path = fsPath.Clean(path)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	for strings.HasPrefix(path, "../") {
		path = path[3:]
	}
	return path
}

func PathParent(path string) string {
	path = CleanPath(path)
	parent := fsPath.Dir(path)
	if parent == "/" || parent == "." {
		parent = ""
	}
	return parent
}

func PathParentTree(path string) []string {
	if path == "" {
		return nil
	}
	path = CleanPath(path)
	r := make([]string, 0, PathDepth(path))
	for path != "" {
		r = append(r, path)
		path = PathParent(path)
	}
	return r
}

var slashPattern = regexp.MustCompile("/")

func PathDepth(path string) int {
	path = CleanPath(path)
	if path == "" {
		return 0
	}
	return len(slashPattern.FindAll([]byte(path), -1)) + 1
}

func PanicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

func RequireNotNil(v interface{}, msg string) {
	if v == nil {
		panic(msg)
	}
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

func CopyWithProgress(dst io.Writer, src io.Reader, progress types.OnProgress) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		w, err := io.CopyBuffer(dst, src, buf)
		if err != nil {
			break
		}
		if w == 0 {
			break
		}
		written += w
		progress(written)
	}
	return
}

func CopyIContent(content types.IContent, w io.Writer, progress types.OnProgress) (int64, error) {
	// copy file from url
	url, _, e := content.GetURL()
	if e == nil {
		resp, e := http.Get(url)
		if e != nil {
			return -1, e
		}
		if resp.StatusCode != 200 {
			return -1, NewRemoteApiError(resp.StatusCode, "failed to copy file")
		}
		defer func() { _ = resp.Body.Close() }()
		return CopyWithProgress(w, resp.Body, progress)
	}
	// copy file from reader
	reader, e := content.GetReader()
	if e != nil {
		return -1, e
	}
	defer func() { _ = reader.Close() }()
	return CopyWithProgress(w, reader, progress)
}

func CopyIContentToTempFile(content types.IContent, progress types.OnProgress) (*os.File, error) {
	file, e := ioutil.TempFile("", "drive-copy")
	if e != nil {
		return nil, e
	}
	_, e = CopyIContent(content, file, progress)
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
				r.Header.Set("Host", dest.Host)
				r.Header.Del("Referer")
			}}

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
			time.Unix(0, content.UpdatedAt()*int64(time.Millisecond)),
			readSeeker)
		return nil
	}

	w.Header().Set("Content-Length", strconv.FormatInt(content.Size(), 10))
	_, e = io.Copy(w, reader)
	return e
}
