package drive

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "webdav",
		DisplayName: i18n.T("drive.webdav.name"),
		README:      i18n.T("drive.webdav.readme"),
		ConfigForm: []types.FormItem{
			{Field: "url", Label: i18n.T("drive.webdav.form.url.label"), Type: "text", Required: true, Description: i18n.T("drive.webdav.form.url.description")},
			{Field: "username", Label: i18n.T("drive.webdav.form.username.label"), Type: "text", Description: i18n.T("drive.webdav.form.username.description")},
			{Field: "password", Label: i18n.T("drive.webdav.form.password.label"), Type: "password"},
			{Field: "cache_ttl", Label: i18n.T("drive.webdav.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.webdav.form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewWebDAVDrive},
	})
}

// NewWebDAVDrive creates a webdav drive
func NewWebDAVDrive(ctx context.Context, config drive_util.DriveConfig,
	utils drive_util.DriveUtils) (types.IDrive, error) {
	u := config["url"]
	username := config["username"]
	password := config["password"]

	cacheTtl, e := time.ParseDuration(config["cache_ttl"])
	if e != nil {
		cacheTtl = -1
	}

	uu, e := url.Parse(u)
	if e != nil {
		return nil, e
	}
	pathPrefix := uu.Path

	w := &WebDAVDrive{
		url: u, username: username, password: password,
		cacheTTL: cacheTtl, pathPrefix: pathPrefix,
	}

	if cacheTtl <= 0 {
		w.cache = drive_util.DummyCache()
	} else {
		w.cache = utils.CreateCache(w.deserializeEntry, nil)
	}

	client, e := req.NewClient(u, w.beforeRequest, w.afterRequest, nil)
	if e != nil {
		return nil, e
	}
	w.c = client

	// check
	_, e = w.Get(ctx, "/")
	if e != nil {
		return nil, e
	}
	return w, nil
}

type WebDAVDrive struct {
	url        string
	pathPrefix string
	username   string
	password   string

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	c *req.Client
}

func (w *WebDAVDrive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (w *WebDAVDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if cached, _ := w.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	resp, e := w.c.Request(ctx, "PROPFIND", utils.BuildURL(path), types.SM{"Depth": "0"}, nil)
	if e != nil {
		return nil, e
	}
	res := multiStatus{}
	if e := resp.XML(&res); e != nil {
		return nil, e
	}
	entry := w.newEntry(res.Response[0])
	_ = w.cache.PutEntry(entry, w.cacheTTL)
	return entry, nil
}

func (w *WebDAVDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, w, path); e != nil {
			return nil, e
		}
	}
	resp, e := w.c.Request(ctx, "PUT", path, nil,
		req.NewReaderBody(drive_util.ProgressReader(reader, ctx), size))
	if e != nil {
		return nil, e
	}
	_ = resp.Dispose()
	_ = w.cache.Evict(utils.PathParent(path), false)
	_ = w.cache.Evict(path, false)
	return w.Get(ctx, path)
}

func (w *WebDAVDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	if dir, e := w.Get(ctx, path); e == nil {
		if !dir.Type().IsDir() {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		return dir, nil
	}
	resp, e := w.c.Request(ctx, "MKCOL", path, nil, nil)
	if e != nil {
		return nil, e
	}
	_ = resp.Dispose()
	_ = w.cache.Evict(utils.PathParent(path), false)
	return w.Get(ctx, path)
}

func (w *WebDAVDrive) isSelf(e types.IEntry) bool {
	if we, ok := e.(*webDavEntry); ok {
		return we.d == w
	}
	return false
}

func (w *WebDAVDrive) copyOrMove(method string, from types.IEntry, to string,
	override bool, ctx types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, w.isSelf)
	if from == nil || from.Type().IsDir() {
		return nil, err.NewUnsupportedError()
	}
	wEntry := from.(*webDavEntry)
	dest, e := w.c.BuildURL(to)
	if e != nil {
		return nil, e
	}
	header := types.SM{"Destination": dest}
	if !override {
		header["Overwrite"] = "F"
	}
	resp, e := w.c.Request(ctx, method, wEntry.path, header, nil)
	if e != nil && !(!override && e == errorPreconditionFailed) {
		return nil, e
	}
	if e == nil {
		_ = resp.Dispose()
	}
	_ = w.cache.Evict(to, true)
	_ = w.cache.Evict(utils.PathParent(to), false)
	if method == "MOVE" {
		_ = w.cache.Evict(from.Path(), true)
		_ = w.cache.Evict(utils.PathParent(from.Path()), false)
	}
	return w.Get(ctx, to)
}

func (w *WebDAVDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	return w.copyOrMove("COPY", from, to, override, ctx)
}

func (w *WebDAVDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	return w.copyOrMove("MOVE", from, to, override, ctx)
}

func (w *WebDAVDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := w.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	resp, e := w.c.Request(ctx, "PROPFIND", utils.BuildURL(path), types.SM{"Depth": "1"}, nil)
	if e != nil {
		return nil, e
	}
	res := multiStatus{}
	if e := resp.XML(&res); e != nil {
		return nil, e
	}

	depth := utils.PathDepth(path)
	entries := make([]types.IEntry, 0)
	for _, e := range res.Response {
		if utils.PathDepth(e.Href)-utils.PathDepth(w.pathPrefix) > depth {
			entries = append(entries, w.newEntry(e))
		}
	}
	_ = w.cache.PutChildren(path, entries, w.cacheTTL)
	return entries, nil
}

func (w *WebDAVDrive) Delete(ctx types.TaskCtx, path string) error {
	resp, e := w.c.Request(ctx, "DELETE", path, nil, nil)
	if e != nil {
		return e
	}
	_ = resp.Dispose()
	_ = w.cache.Evict(path, true)
	_ = w.cache.Evict(utils.PathParent(path), false)
	return nil
}

func (w *WebDAVDrive) Upload(ctx context.Context, path string, size int64,
	override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, w, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (w *WebDAVDrive) beforeRequest(req *http.Request) error {
	if w.username != "" {
		req.SetBasicAuth(w.username, w.password)
	}
	return nil
}

var errorPreconditionFailed = errors.New("precondition failed")

func (w *WebDAVDrive) afterRequest(resp req.Response) error {
	if resp.Status() < 200 || resp.Status() >= 300 {
		if resp.Status() == http.StatusNotFound {
			return err.NewNotFoundError()
		}
		if resp.Status() == http.StatusPreconditionFailed {
			return errorPreconditionFailed
		}
		if resp.Status() == http.StatusUnauthorized {
			return err.NewUnauthorizedError(i18n.T("drive.webdav.wrong_user_or_password"))
		}
		return err.NewRemoteApiError(500, i18n.T("drive.webdav.remote_error", strconv.Itoa(resp.Status())))
	}
	return nil
}

func (w *WebDAVDrive) deserializeEntry(dat string) (types.IEntry, error) {
	ec, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	return &webDavEntry{
		path: ec.Path, modTime: ec.ModTime,
		size: ec.Size, isDir: ec.Type.IsDir(), d: w,
	}, nil
}

func (w *WebDAVDrive) newEntry(res propfindResponse) *webDavEntry {
	modTime, _ := time.Parse(time.RFC1123, res.LastModified)
	href, _ := url.PathUnescape(res.Href)
	href = href[len(w.pathPrefix):]
	return &webDavEntry{
		path:    utils.CleanPath(href),
		modTime: utils.Millisecond(modTime),
		size:    res.Size,
		isDir:   res.CollectionMark != nil,
		d:       w,
	}
}

type webDavEntry struct {
	path    string
	modTime int64
	size    int64
	isDir   bool

	d *WebDAVDrive
}

func (w *webDavEntry) Path() string {
	return w.path
}

func (w *webDavEntry) Type() types.EntryType {
	if w.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (w *webDavEntry) Size() int64 {
	if w.Type().IsDir() {
		return -1
	}
	return w.size
}

func (w *webDavEntry) Meta() types.EntryMeta {
	return types.EntryMeta{CanRead: true, CanWrite: true}
}

func (w *webDavEntry) ModTime() int64 {
	return w.modTime
}

func (w *webDavEntry) Drive() types.IDrive {
	return w.d
}

func (w *webDavEntry) Name() string {
	return utils.PathBase(w.path)
}

func (w *webDavEntry) GetReader(ctx context.Context) (io.ReadCloser, error) {
	resp, e := w.d.c.Get(ctx, w.path, nil)
	if e != nil {
		return nil, e
	}
	return resp.Response().Body, nil
}

func (w *webDavEntry) GetURL(context.Context) (*types.ContentURL, error) {
	if !w.Type().IsFile() {
		return nil, err.NewNotAllowedError()
	}
	u, e := w.d.c.BuildURL(w.path)
	if e != nil {
		return nil, e
	}
	var header types.SM = nil
	if w.d.username != "" {
		header = types.SM{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(w.d.username+":"+w.d.password)),
		}
	}
	return &types.ContentURL{URL: u, Proxy: true, Header: header}, nil
}

type multiStatus struct {
	Response []propfindResponse `xml:"response"`
}

type propfindResponse struct {
	Href           string    `xml:"href"`
	LastModified   string    `xml:"propstat>prop>getlastmodified"`
	Size           int64     `xml:"propstat>prop>getcontentlength"`
	ETag           string    `xml:"propstat>prop>getetag"`
	CollectionMark *xml.Name `xml:"propstat>prop>resourcetype>collection"`
}
