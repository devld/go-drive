package onedrive

import (
	"errors"
	"fmt"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"strconv"
	"time"
)

type OneDrive struct {
	driveId string

	c *req.Client

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	uploadProxy   bool
	downloadProxy bool
}

func NewOneDrive(config drive_util.DriveConfig, driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	resp, e := drive_util.OAuthGet(*oauthReq(driveUtils.Config), config, driveUtils.Data)
	if e != nil {
		return nil, e
	}

	proxyUpload := config["proxy_upload"]
	proxyDownload := config["proxy_download"]
	cacheTtl, e := time.ParseDuration(config["cache_ttl"])
	if e != nil {
		cacheTtl = -1
	}

	params, e := driveUtils.Data.Load("drive_id")
	od := &OneDrive{
		driveId:       params["drive_id"],
		cacheTTL:      cacheTtl,
		uploadProxy:   proxyUpload != "",
		downloadProxy: proxyDownload != "",
	}
	if cacheTtl <= 0 {
		od.cache = drive_util.DummyCache()
	} else {
		od.cache = driveUtils.CreateCache(od.deserializeEntry, nil)
	}

	if od.driveId == "" {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.onedrive.drive_not_selected"))
	}

	od.c, e = req.NewClient(
		utils.BuildURL("https://graph.microsoft.com/v1.0/drives/{}", od.driveId),
		nil, ifApiCallError, resp.Client(nil))

	return od, e
}

func (o *OneDrive) Meta() types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (o *OneDrive) Get(path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return &oneDriveEntry{id: "root", path: path, isDir: true}, nil
	}
	if cached, _ := o.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	resp, e := o.c.Get(utils.BuildURL("/root:/{}?expand=thumbnails", path), nil)
	if e != nil {
		return nil, e
	}
	entry, e := o.toEntry(resp)
	if e == nil {
		_ = o.cache.PutEntry(entry, o.cacheTTL)
	}
	return entry, nil
}

func (o *OneDrive) Save(path string, size int64, override bool, reader io.Reader, ctx types.TaskCtx) (types.IEntry, error) {
	var entry *oneDriveEntry = nil
	get, e := o.Get(path)
	if e != nil && !err.IsNotFoundError(e) {
		return nil, e
	}
	if e == nil {
		entry = get.(*oneDriveEntry)
	}
	if !override && entry != nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	filename := utils.PathBase(path)
	if filename == "" {
		return nil, err.NewBadRequestError(i18n.T("drive.invalid_path"))
	}
	parent, e := o.Get(utils.PathParent(path))
	if e != nil {
		return nil, e
	}
	if size <= uploadChunkSize {
		if entry != nil {
			entry, e = o.uploadSmallFileOverride(entry.id, size, reader, ctx)
		} else {
			entry, e = o.uploadSmallFile(parent.(*oneDriveEntry).id, filename, size, reader, ctx)
		}
	} else {
		entry, e = o.uploadLargeFile(parent.(*oneDriveEntry).id, filename, size, override, reader, ctx)
	}
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(parent.Path(), false)
	_ = o.cache.Evict(path, false)
	return entry, nil
}

func (o *OneDrive) MakeDir(path string) (types.IEntry, error) {
	parent := utils.PathParent(path)
	name := utils.PathBase(path)
	resp, e := o.c.Post(pathURL(parent)+"/children", nil, req.NewJsonBody(types.M{
		"name":                              name,
		"folder":                            types.M{},
		"@microsoft.graph.conflictBehavior": "fail",
	}))
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(parent, false)
	return o.toEntry(resp)
}

func (o *OneDrive) isSelf(e types.IEntry) bool {
	if fe, ok := e.(*oneDriveEntry); ok {
		return fe.d == o
	}
	return false
}

func (o *OneDrive) Copy(from types.IEntry, to string, _ bool, ctx types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, o.isSelf)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	ctx.Total(from.Size(), false)
	toParentPath := utils.PathParent(to)
	toName := utils.PathBase(to)
	resp, e := o.c.Post(
		idURL(from.(*oneDriveEntry).id)+"/copy", nil,
		req.NewJsonBody(types.M{
			"parentReference": types.M{"path": itemPath(toParentPath)},
			"name":            toName,
		}),
	)
	if e != nil {
		return nil, e
	}
	_ = resp.Dispose()
	if resp.Status() == 202 {
		// we should wait for it to finish
		waitUrl := resp.Response().Header.Get("Location")
		if e := waitLongRunningAction(waitUrl); e != nil {
			return nil, e
		}
	}
	ctx.Progress(from.Size(), false)
	_ = o.cache.Evict(to, true)
	_ = o.cache.Evict(utils.PathParent(to), false)
	return o.Get(to)
}

func (o *OneDrive) Move(from types.IEntry, to string, _ bool, ctx types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, o.isSelf)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	ctx.Total(from.Size(), false)
	toParentPath := utils.PathParent(to)
	toName := utils.PathBase(to)
	resp, e := o.c.Request("PATCH",
		idURL(from.(*oneDriveEntry).id), nil,
		req.NewJsonBody(types.M{
			"parentReference": types.M{"path": itemPath(toParentPath)},
			"name":            toName,
		}),
	)
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(utils.PathParent(to), false)
	_ = o.cache.Evict(from.Path(), true)
	_ = o.cache.Evict(utils.PathParent(from.Path()), false)
	ctx.Progress(from.Size(), false)
	return o.toEntry(resp)
}

func (o *OneDrive) List(path string) ([]types.IEntry, error) {
	if cached, _ := o.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	reqPath := pathURL(path) + "/children?$expand=thumbnails"
	res := driveItems{}
	resp, e := o.c.Get(reqPath, nil)
	if e != nil {
		return nil, e
	}
	if e := resp.Json(&res); e != nil {
		return nil, e
	}
	entries := make([]types.IEntry, 0)
	for _, v := range res.Items {
		if v.Deleted != nil {
			continue
		}
		entries = append(entries, o.newEntry(v))
	}
	_ = o.cache.PutChildren(path, entries, o.cacheTTL)
	return entries, nil
}

func (o *OneDrive) Delete(path string, _ types.TaskCtx) error {
	entry, e := o.Get(path)
	if e != nil {
		return e
	}
	resp, e := o.c.Request("DELETE", idURL(entry.(*oneDriveEntry).id), nil, nil)
	if e != nil {
		return e
	}
	_ = resp.Dispose()
	_ = o.cache.Evict(path, true)
	_ = o.cache.Evict(utils.PathParent(path), false)
	return nil
}

func (o *OneDrive) Upload(path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	if o.uploadProxy {
		return types.UseLocalProvider(size), nil
	}
	action := config["action"]
	switch action {
	case "CompleteUpload":
		_ = o.cache.Evict(path, false)
		_ = o.cache.Evict(utils.PathParent(path), false)
		return nil, nil
	default:
		parent, e := o.Get(utils.PathParent(path))
		if e != nil {
			return nil, e
		}
		filename := utils.PathBase(path)
		sessionUrl, e := o.createUploadSession(parent.(*oneDriveEntry).id, filename, override)
		if e != nil {
			return nil, e
		}
		return &types.DriveUploadConfig{
			Provider: types.OneDriveProvider,
			Config:   types.SM{"url": sessionUrl},
		}, nil
	}
}

func (o *OneDrive) newEntry(item driveItem) *oneDriveEntry {
	modTime, _ := time.Parse(time.RFC3339, item.ModTime)
	thumbnailUrl := ""
	if supportThumbnail(item) &&
		item.Thumbnails != nil && len(item.Thumbnails) > 0 &&
		item.Thumbnails[0].Large != nil {
		thumbnailUrl = item.Thumbnails[0].Large.URL
	}
	return &oneDriveEntry{
		id:                   item.Id,
		path:                 item.Path(),
		isDir:                item.Folder != nil,
		size:                 item.Size,
		modTime:              utils.Millisecond(modTime),
		d:                    o,
		thumbnail:            thumbnailUrl,
		downloadUrl:          item.DownloadURL,
		downloadUrlExpiresAt: time.Now().Add(downloadUrlTTL).Unix(),
	}
}

type oneDriveEntry struct {
	id      string
	path    string
	isDir   bool
	size    int64
	modTime int64
	d       *OneDrive

	thumbnail string

	downloadUrl          string
	downloadUrlExpiresAt int64
}

func (o *oneDriveEntry) Path() string {
	return o.path
}

func (o *oneDriveEntry) Type() types.EntryType {
	if o.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (o *oneDriveEntry) Size() int64 {
	return o.size
}

func (o *oneDriveEntry) Meta() types.EntryMeta {
	return types.EntryMeta{
		CanRead: true, CanWrite: true,
		Thumbnail: o.thumbnail,
	}
}

func (o *oneDriveEntry) ModTime() int64 {
	return o.modTime
}

func (o *oneDriveEntry) Drive() types.IDrive {
	return o.d
}

func (o *oneDriveEntry) Name() string {
	return utils.PathBase(o.path)
}

func (o *oneDriveEntry) GetReader() (io.ReadCloser, error) {
	u, e := o.GetURL()
	if e != nil {
		return nil, e
	}
	return drive_util.GetURL(u.URL, nil)
}

func (o *oneDriveEntry) GetURL() (*types.ContentURL, error) {
	if o.isDir {
		return nil, err.NewNotAllowedError()
	}
	u := o.downloadUrl
	if o.downloadUrlExpiresAt <= time.Now().Unix() {
		resp, e := o.d.c.Get(pathURL(o.path)+"/content", nil)
		if e != nil {
			return nil, e
		}
		if resp.Status() != http.StatusFound {
			return nil, errors.New(fmt.Sprintf("%d", resp.Status()))
		}
		u = resp.Response().Header.Get("Location")
		o.downloadUrl = u
		o.downloadUrlExpiresAt = time.Now().Add(downloadUrlTTL).Unix()
		_ = o.d.cache.PutEntry(o, o.d.cacheTTL)
	}
	return &types.ContentURL{URL: o.downloadUrl, Proxy: o.d.downloadProxy}, nil
}

func (o *oneDriveEntry) EntryData() types.SM {
	return types.SM{
		"id": o.id,
		"du": o.downloadUrl,
		"de": strconv.FormatInt(o.downloadUrlExpiresAt, 10),
		"th": o.thumbnail,
	}
}
