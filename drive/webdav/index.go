package webdav

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var davT = i18n.TPrefix("drive.webdav.")

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "webdav",
		DisplayName: davT("name"),
		README:      davT("readme"),
		ConfigForm: []types.FormItem{
			{Field: "url", Label: davT("form.url.label"), Type: "text", Required: true, Description: davT("form.url.description")},
			{Field: "username", Label: davT("form.username.label"), Type: "text", Description: davT("form.username.description")},
			{Field: "password", Label: davT("form.password.label"), Type: "password", Description: davT("form.password.description")},
			{Field: "cache_ttl", Label: davT("form.cache_ttl.label"), Type: "text", Description: davT("form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewDrive},
	})
}

// NewDrive creates a webdav drive
func NewDrive(ctx context.Context, config types.SM,
	utils drive_util.DriveUtils) (types.IDrive, error) {
	u := config["url"]
	username := config["username"]
	password := config["password"]

	cacheTtl := config.GetDuration("cache_ttl", -1)

	u = strings.TrimRight(u, "/")

	uu, e := url.Parse(u)
	if e != nil {
		return nil, e
	}
	pathPrefix := uu.Path

	w := &Drive{
		username: username, password: password,
		cacheTTL: cacheTtl, pathPrefix: pathPrefix,
	}

	if cacheTtl <= 0 {
		w.cache = drive_util.DummyCache()
	} else {
		w.cache = utils.CreateCache(w.deserializeEntry)
	}

	client, e := req.NewClient(u, w.beforeRequest, w.afterRequest, &http.Client{})
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

type Drive struct {
	pathPrefix string
	username   string
	password   string

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	c *req.Client
}

func (w *Drive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{Writable: true}
}

func (w *Drive) Get(ctx context.Context, path string) (types.IEntry, error) {
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

func (w *Drive) Save(ctx types.TaskCtx, path string, size int64,
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

func (w *Drive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
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

func (w *Drive) copyOrMove(method string, from types.IEntry, to string,
	override bool, ctx types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(w, from)
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

func (w *Drive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	return w.copyOrMove("COPY", from, to, override, ctx)
}

func (w *Drive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	return w.copyOrMove("MOVE", from, to, override, ctx)
}

func (w *Drive) List(ctx context.Context, path string) ([]types.IEntry, error) {
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

func (w *Drive) Delete(ctx types.TaskCtx, path string) error {
	resp, e := w.c.Request(ctx, "DELETE", path, nil, nil)
	if e != nil {
		return e
	}
	_ = resp.Dispose()
	_ = w.cache.Evict(path, true)
	_ = w.cache.Evict(utils.PathParent(path), false)
	return nil
}

func (w *Drive) Upload(ctx context.Context, path string, size int64,
	override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, w, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (w *Drive) beforeRequest(req *http.Request) error {
	if w.username != "" {
		req.SetBasicAuth(w.username, w.password)
	}
	return nil
}

var errorPreconditionFailed = errors.New("precondition failed")

func (w *Drive) afterRequest(resp req.Response) error {
	if resp.Status() < 200 || resp.Status() >= 300 {
		if resp.Status() == http.StatusNotFound {
			return err.NewNotFoundError()
		}
		if resp.Status() == http.StatusPreconditionFailed {
			return errorPreconditionFailed
		}
		if resp.Status() == http.StatusUnauthorized {
			return err.NewUnauthorizedError(davT("wrong_user_or_password"))
		}
		return err.NewRemoteApiError(500, davT("remote_error", strconv.Itoa(resp.Status())))
	}
	return nil
}

func (w *Drive) deserializeEntry(ec drive_util.EntryCacheItem) (types.IEntry, error) {
	return &webDavEntry{
		path: ec.Path, modTime: ec.ModTime,
		size: ec.Size, isDir: ec.Type.IsDir(), d: w,
	}, nil
}

func (w *Drive) newEntry(res propfindResponse) *webDavEntry {
	modTime, _ := time.Parse(time.RFC1123, res.LastModified)
	href, _ := url.PathUnescape(res.Href)
	href = strings.TrimPrefix(href, w.pathPrefix)
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

	d *Drive
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
	return types.EntryMeta{Readable: true, Writable: true}
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

func (w *webDavEntry) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	headers := types.SM{}
	rangeStr := drive_util.BuildRangeHeader(start, size)
	if rangeStr != "" {
		headers["Range"] = rangeStr
	}
	resp, e := w.d.c.Get(ctx, w.path, headers)
	if e != nil {
		return nil, e
	}
	if rangeStr != "" && resp.Status() != http.StatusPartialContent {
		return nil, err.NewUnsupportedError()
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
