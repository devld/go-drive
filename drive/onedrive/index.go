package onedrive

import (
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	scope                = "Files.ReadWrite offline_access User.Read"
	redirectUri          = "https://go-drive.top/oauth_callback"
	tokenRefreshInterval = 30 * time.Minute
)

type OneDrive struct {
	accessToken string
	driveId     string

	ds               drive_util.DriveDataStore
	refreshTokenStop func()
	c                *common.HttpClient

	clientId     string
	clientSecret string

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	uploadProxy   bool
	downloadProxy bool
}

func NewOneDrive(config drive_util.DriveConfig, utils drive_util.DriveUtils) (types.IDrive, error) {
	params, e := utils.Data.Load("token", "expires_at", "drive_id")
	if e != nil {
		return nil, e
	}
	expiresAt := time.Unix(common.ToInt64(params["expires_at"], -1), 0)
	if expiresAt.Before(time.Now()) {
		return nil, common.NewNotAllowedMessageError("drive not configured")
	}

	proxyUpload := config["proxy_upload"]
	proxyDownload := config["proxy_download"]
	cacheTtl, e := time.ParseDuration(config["cache_ttl"])
	if e != nil {
		cacheTtl = -1
	}

	od := &OneDrive{
		accessToken:   params["token"],
		ds:            utils.Data,
		clientId:      config["client_id"],
		clientSecret:  config["client_secret"],
		driveId:       params["drive_id"],
		cacheTTL:      cacheTtl,
		uploadProxy:   proxyUpload != "",
		downloadProxy: proxyDownload != "",
	}
	od.cache = utils.CreateCache(od.deserializeEntry, nil)

	if od.driveId == "" {
		return nil, common.NewNotAllowedMessageError("drive not configured")
	}
	od.refreshToken()
	od.refreshTokenStop = common.TimeTick(od.refreshToken, tokenRefreshInterval)

	od.c, e = common.NewHttpClient(
		common.BuildURL("https://graph.microsoft.com/v1.0/drives/{}", od.driveId),
		od.addToken, ifApiCallError, nil)
	if e != nil {
		_ = od.Dispose()
		return nil, e
	}
	return od, nil
}

func (o *OneDrive) Meta() types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (o *OneDrive) Get(path string) (types.IEntry, error) {
	if common.IsRootPath(path) {
		return &oneDriveEntry{path: path, isDir: true}, nil
	}
	if cached, _ := o.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	resp, e := o.c.Get(common.BuildURL("/root:/{}?expand=thumbnails", path), nil)
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
	if e != nil && !common.IsNotFoundError(e) {
		return nil, e
	}
	if e == nil {
		entry = get.(*oneDriveEntry)
	}
	if !override && entry != nil {
		return nil, common.NewNotAllowedMessageError("file exists")
	}
	parent, e := o.Get(common.PathParent(path))
	filename := common.PathBase(path)
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
	parent := common.PathParent(path)
	name := common.PathBase(path)
	resp, e := o.c.Post(pathURL(parent)+"/children", nil, common.NewJsonBody(types.M{
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
	if from == nil || from.Type().IsDir() {
		return nil, common.NewUnsupportedError()
	}
	ctx.Total(from.Size(), false)
	toParentPath := common.PathParent(to)
	toName := common.PathBase(to)
	resp, e := o.c.Post(
		idURL(from.(*oneDriveEntry).id)+"/copy", nil,
		common.NewJsonBody(types.M{
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
	_ = o.cache.Evict(common.PathParent(to), false)
	return o.Get(to)
}

func (o *OneDrive) Move(from types.IEntry, to string, _ bool, ctx types.TaskCtx) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, o.isSelf)
	if from == nil || from.Type().IsDir() {
		return nil, common.NewUnsupportedError()
	}
	ctx.Total(from.Size(), false)
	toParentPath := common.PathParent(to)
	toName := common.PathBase(to)
	resp, e := o.c.Request("PATCH",
		idURL(from.(*oneDriveEntry).id), nil,
		common.NewJsonBody(types.M{
			"parentReference": types.M{"path": itemPath(toParentPath)},
			"name":            toName,
		}),
	)
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(common.PathParent(to), false)
	_ = o.cache.Evict(from.Path(), true)
	_ = o.cache.Evict(common.PathParent(from.Path()), false)
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
	_ = o.cache.Evict(common.PathParent(path), false)
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
		_ = o.cache.Evict(common.PathParent(path), false)
		return nil, nil
	default:
		parent, e := o.Get(common.PathParent(path))
		if e != nil {
			return nil, e
		}
		filename := common.PathBase(path)
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

func (o *OneDrive) Dispose() error {
	o.refreshTokenStop()
	return nil
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
		modTime:              common.Millisecond(modTime),
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
	return common.PathBase(o.path)
}

func (o *oneDriveEntry) GetReader() (io.ReadCloser, error) {
	u, _, e := o.GetURL()
	if e != nil {
		return nil, e
	}
	return common.GetURL(u)
}

func (o *oneDriveEntry) GetURL() (string, bool, error) {
	if o.isDir {
		return "", false, common.NewNotAllowedError()
	}
	u := o.downloadUrl
	if o.downloadUrlExpiresAt <= time.Now().Unix() {
		resp, e := o.d.c.Get(pathURL(o.path)+"/content", nil)
		if e != nil {
			return "", false, e
		}
		if resp.Status() != http.StatusFound {
			return "", false, errors.New(fmt.Sprintf("%d", resp.Status()))
		}
		u = resp.Response().Header.Get("Location")
		o.downloadUrl = u
		o.downloadUrlExpiresAt = time.Now().Add(downloadUrlTTL).Unix()
		_ = o.d.cache.PutEntry(o, o.d.cacheTTL)
	}
	return o.downloadUrl, o.d.downloadProxy, nil
}

func (o *oneDriveEntry) EntryData() types.SM {
	return types.SM{
		"id": o.id,
		"du": o.downloadUrl,
		"de": strconv.FormatInt(o.downloadUrlExpiresAt, 10),
		"th": o.thumbnail,
	}
}
