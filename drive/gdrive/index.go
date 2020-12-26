package gdrive

import (
	"context"
	"fmt"
	"go-drive/common/drive_util"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"io"
	url2 "net/url"
	path2 "path"
	"strings"
	"time"
)

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "gdrive",
		DisplayName: i18n.T("drive.gdrive.name"),
		README:      i18n.T("drive.gdrive.readme"),
		ConfigForm: []types.FormItem{
			{Field: "client_id", Label: i18n.T("drive.gdrive.form.client_id.label"), Type: "text", Required: true},
			{Field: "client_secret", Label: i18n.T("drive.gdrive.form.client_secret.label"), Type: "password", Required: true},
			{Field: "cache_ttl", Label: i18n.T("drive.gdrive.form.cache_ttl.label"), Type: "text", Description: i18n.T("drive.gdrive.form.cache_ttl.description"), DefaultValue: "4h"},
		},
		Factory: drive_util.DriveFactory{Create: NewGDrive, InitConfig: InitConfig, Init: Init},
	})
}

func NewGDrive(ctx context.Context, config types.SM, utils drive_util.DriveUtils) (types.IDrive, error) {
	resp, e := drive_util.OAuthGet(*oauthReq(utils.Config), config, utils.Data)
	if e != nil {
		return nil, e
	}
	service, e := drive.NewService(ctx, option.WithHTTPClient(resp.Client(nil)))
	if e != nil {
		return nil, e
	}

	cacheTtl := config.GetDuration("cache_ttl", -1)

	g := &GDrive{
		s:        service,
		cacheTTL: cacheTtl,
		ts:       resp.TokenSource(nil),
	}
	if cacheTtl <= 0 {
		g.cache = drive_util.DummyCache()
	} else {
		g.cache = utils.CreateCache(g.deserializeEntry, nil)
	}
	return g, nil
}

type GDrive struct {
	s *drive.Service

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	ts oauth2.TokenSource
}

func (g *GDrive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (g *GDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	return g.getByPath(path, ctx)
}

func (g *GDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, g, path); e != nil {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
	}
	parent, filename, e := g.getParentTarget(path, ctx)
	if e != nil {
		return nil, e
	}

	ctx.Total(size, true)

	var lastCurrent int64 = 0
	resp, e := g.s.Files.Create(&drive.File{Name: filename, Parents: []string{parent.fileId()}}).
		Media(reader).Context(ctx).ProgressUpdater(
		func(current, total int64) {
			ctx.Progress(current-lastCurrent, false)
			lastCurrent = current
		},
	).Do()
	if e != nil {
		return nil, e
	}

	_ = g.cache.Evict(utils.PathParent(path), false)
	_ = g.cache.Evict(path, false)

	return g.newEntry(parent.path, resp), nil
}

func (g *GDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	if dir, e := g.Get(ctx, path); e == nil {
		if !dir.Type().IsDir() {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		return dir, nil
	}
	parent, dirName, e := g.getParentTarget(path, ctx)
	if e != nil {
		return nil, e
	}
	resp, e := g.s.Files.Create(&drive.File{
		Name: dirName, Parents: []string{parent.fileId()},
		MimeType: typeFolder,
	}).Context(ctx).Do()
	if e != nil {
		return nil, e
	}
	_ = g.cache.Evict(utils.PathParent(path), false)
	return g.newEntry(parent.path, resp), nil
}

func (g *GDrive) isSelf(e types.IEntry) bool {
	if fe, ok := e.(*gdriveEntry); ok {
		return fe.d == g
	}
	return false
}

func (g *GDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, g.isSelf)
	if from == nil || from.Type().IsDir() {
		// google drive api does not support to copy folder
		return nil, err.NewUnsupportedError()
	}
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, g, to); e != nil {
			return nil, e
		}
	}
	ctx.Total(from.Size(), false)
	parent, filename, e := g.getParentTarget(to, ctx)
	if e != nil {
		return nil, e
	}
	resp, e := g.s.Files.Copy(from.(*gdriveEntry).id,
		&drive.File{Name: filename, Parents: []string{parent.fileId()}}).Do()
	if e != nil {
		return nil, e
	}
	ctx.Progress(from.Size(), false)

	_ = g.cache.Evict(to, true)
	_ = g.cache.Evict(utils.PathParent(to), false)

	return g.newEntry(parent.path, resp), nil
}

func (g *GDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetIEntry(from, g.isSelf)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, g, to); e != nil {
			return nil, e
		}
	}
	ctx.Total(from.Size(), false)
	parent, filename, e := g.getParentTarget(to, ctx)
	if e != nil {
		return nil, e
	}
	fromParent, e := g.getByPath(utils.PathParent(from.Path()), ctx)
	if e != nil {
		return nil, e
	}
	resp, e := g.s.Files.Update(from.(*gdriveEntry).id, &drive.File{Name: filename}).Context(ctx).
		AddParents(parent.fileId()).RemoveParents(fromParent.fileId()).Do()
	if e != nil {
		return nil, e
	}
	ctx.Progress(from.Size(), false)

	_ = g.cache.Evict(utils.PathParent(to), false)
	_ = g.cache.Evict(from.Path(), true)
	_ = g.cache.Evict(utils.PathParent(from.Path()), false)

	return g.newEntry(parent.path, resp), nil
}

func (g *GDrive) getParentTarget(path string, ctx context.Context) (*gdriveEntry, string, error) {
	name := utils.PathBase(path)
	if name == "" {
		return nil, "", err.NewBadRequestError(i18n.T("drive.invalid_path"))
	}
	parent, e := g.getByPath(utils.PathParent(path), ctx)
	if e != nil {
		return nil, "", e
	}
	return parent, name, nil
}

func (g *GDrive) getByPath(path string, ctx context.Context) (*gdriveEntry, error) {
	if utils.IsRootPath(path) {
		return &gdriveEntry{id: "root", isDir: true, modTime: -1, d: g}, nil
	}
	if cached, _ := g.cache.GetEntry(path); cached != nil {
		return cached.(*gdriveEntry), nil
	}
	siblings, e := g.List(ctx, utils.PathParent(path))
	if e != nil {
		return nil, e
	}
	var found *gdriveEntry = nil
	for _, e := range siblings {
		ge := e.(*gdriveEntry)
		if ge.path == path {
			found = ge
			break
		}
	}
	if found == nil {
		return nil, err.NewNotFoundError()
	}
	_ = g.cache.PutEntry(found, g.cacheTTL)
	return found, nil
}

func (g *GDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := g.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	id := "root"
	if !utils.IsRootPath(path) {
		ge, e := g.getByPath(path, ctx)
		if e != nil {
			return nil, e
		}
		id = ge.fileId()
	}
	resp, e := g.s.Files.List().Context(ctx).
		Q(fmt.Sprintf("'%s' in parents and trashed = false", id)).
		Fields("files(id,name,mimeType,parents,hasThumbnail,thumbnailLink,modifiedTime,driveId,size," +
			"shortcutDetails,capabilities(canDownload,canEdit,canDelete,canCopy))").
		Do()
	if e != nil {
		return nil, e
	}
	entries := g.processEntries(path, resp.Files)
	_ = g.cache.PutChildren(path, entries, g.cacheTTL)
	return entries, nil
}

func (g *GDrive) Delete(ctx types.TaskCtx, path string) error {
	ge, e := g.getByPath(path, ctx)
	if e != nil {
		return e
	}
	e = g.s.Files.Delete(ge.id).Context(ctx).Do()
	if e == nil {
		_ = g.cache.Evict(path, true)
		_ = g.cache.Evict(utils.PathParent(path), false)
	}
	return e
}

func (g *GDrive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, g, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

// processEntries appends file id to duplicate filenames
func (g *GDrive) processEntries(parentPath string, files []*drive.File) []types.IEntry {
	entries := make([]types.IEntry, 0, len(files))
	nameMap := make(map[string][]*gdriveEntry)
	for _, f := range files {
		f.Name = strings.ReplaceAll(f.Name, "/", "_")
		entry := g.newEntry(parentPath, f)
		nameMap[f.Name] = append(nameMap[f.Name], entry)
		entries = append(entries, entry)
	}
	for name, es := range nameMap {
		if len(es) <= 1 {
			continue
		}
		for _, e := range es {
			id := e.id
			if len(id) > 6 {
				id = id[:6]
			}
			e.path = path2.Join(parentPath, appendFilenameSuffix(name, "-"+id))
		}
	}
	return entries
}

func appendFilenameSuffix(name, suffix string) string {
	dotIdx := strings.LastIndexByte(name, '.')
	if dotIdx == -1 {
		return name + suffix
	}
	return name[:dotIdx] + suffix + name[dotIdx:]
}

func (g *GDrive) newEntry(parentPath string, file *drive.File) *gdriveEntry {
	modTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)
	size := file.Size
	if strings.HasPrefix(file.MimeType, typeGoogleAppPrefix) {
		size = -1
	}
	targetId := ""
	targetMime := ""
	if file.ShortcutDetails != nil {
		targetId = file.ShortcutDetails.TargetId
		targetMime = file.ShortcutDetails.TargetMimeType
	}
	thumbnail := file.ThumbnailLink
	if !strings.Contains(thumbnail, "googleusercontent.com") {
		thumbnail = ""
	}
	return &gdriveEntry{
		d: g, id: file.Id, mime: file.MimeType,
		path:  path2.Join(parentPath, file.Name),
		isDir: file.MimeType == typeFolder || targetMime == typeFolder,
		size:  size, modTime: utils.Millisecond(modTime),
		targetId: targetId, targetMime: targetMime, thumbnail: thumbnail,
	}
}

type gdriveEntry struct {
	id   string
	mime string
	// targetId is the target fileId, if it's a shortcut
	targetId string
	// targetMime is the target mimeType, if it's a shortcut
	targetMime string
	thumbnail  string

	path    string
	isDir   bool
	size    int64
	modTime int64

	d *GDrive
}

func (g *gdriveEntry) Path() string {
	return g.path
}

func (g *gdriveEntry) Type() types.EntryType {
	if g.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (g *gdriveEntry) Size() int64 {
	if g.isDir {
		return -1
	}
	return g.size
}

func (g *gdriveEntry) fileId() string {
	if g.targetId != "" {
		return g.targetId
	}
	return g.id
}

func (g *gdriveEntry) mimeType() string {
	if g.targetMime != "" {
		return g.targetMime
	}
	return g.mime
}

func (g *gdriveEntry) Meta() types.EntryMeta {
	return types.EntryMeta{
		CanRead: true, CanWrite: true, Thumbnail: g.thumbnail,
		Props: types.M{
			"ext": mimeTypeExtensionsMap[g.mimeType()],
		},
	}
}

func (g *gdriveEntry) ModTime() int64 {
	return g.modTime
}

func (g *gdriveEntry) Drive() types.IDrive {
	return g.d
}

func (g *gdriveEntry) Name() string {
	return utils.PathBase(g.path)
}

func (g *gdriveEntry) GetReader(ctx context.Context) (io.ReadCloser, error) {
	u, e := g.GetURL(ctx)
	if e != nil {
		return nil, e
	}
	return drive_util.GetURL(ctx, u.URL, u.Header)
}

func (g *gdriveEntry) GetURL(context.Context) (*types.ContentURL, error) {
	downloadUrl := ""

	fileId := g.fileId()
	exportMime := exportMimeTypeMap[g.mimeType()]
	if exportMime != "" {
		downloadUrl = utils.BuildURL(g.d.s.BasePath+"files/{}/export", fileId) +
			"?alt=media&mimeType=" + url2.QueryEscape(exportMime)
	} else {
		if strings.HasPrefix(g.mimeType(), typeGoogleAppPrefix) {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_not_downloadable"))
		}
	}
	if downloadUrl == "" {
		downloadUrl = utils.BuildURL(g.d.s.BasePath+"files/{}", fileId) + "?alt=media"
	}

	t, e := g.d.ts.Token()
	if e != nil {
		return nil, e
	}
	return &types.ContentURL{
		Proxy: true, URL: downloadUrl,
		Header: types.SM{"Authorization": t.TokenType + " " + t.AccessToken},
	}, nil
}

func (g *gdriveEntry) EntryData() types.SM {
	return types.SM{
		"i": g.id, "m": g.mime,
		"ti": g.targetId, "tm": g.targetMime,
		"th": g.thumbnail,
	}
}
