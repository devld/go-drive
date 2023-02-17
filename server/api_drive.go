package server

import (
	"archive/zip"
	"context"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxProxySizeKey = "proxy.maxSize"
	maxZipSizeKey   = "zip.maxSize"
)

func InitDriveRoutes(
	router gin.IRouter,
	access *drive.Access,
	searcher *search.Service,
	config common.Config,
	thumbnail *thumbnail.Maker,
	signer *utils.Signer,
	chunkUploader *ChunkUploader,
	runner task.Runner,
	tokenStore types.TokenStore,
	userDAO *storage.UserDAO,
	optionsDAO *storage.OptionsDAO,
	pathMetaDAO *storage.PathMetaDAO) error {

	dr := driveRoute{
		config:        config,
		access:        access,
		searcher:      searcher,
		tokenStore:    tokenStore,
		chunkUploader: chunkUploader,
		thumbnail:     thumbnail,
		runner:        runner,
		signer:        signer,
		options:       optionsDAO,
		pathMeta:      pathMetaDAO,
	}

	scriptsDir, _ := config.GetDir(config.DriveUploadersDir, false)
	router.Group("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Next()
	}).Static("/drive-uploader", scriptsDir)

	signatureAuthRoute := router.Group("/", SignatureAuth(signer, userDAO, false))

	// get file content
	signatureAuthRoute.HEAD("/content/*path", dr.getContent)
	signatureAuthRoute.GET("/content/*path", dr.getContent)
	signatureAuthRoute.GET("/thumbnail/*path", dr.getThumbnail)

	tokenAuth := TokenAuth(tokenStore)
	r := router.Group("/", tokenAuth)

	router.POST("/zip", TokenAuthWithPostParams(tokenStore), dr.zipDownload)

	// list entries/drives
	router.GET("/entries/*path", SignatureAuth(signer, userDAO, true), tokenAuth, dr.list)

	// set path password
	r.POST("/password/*path", dr.setPathPassword)

	// get entry info
	r.GET("/entry/*path", dr.get)
	// mkdir
	r.POST("/mkdir/*path", dr.makeDir)
	// copy file
	r.POST("/copy", dr.copyEntry)
	// move file
	r.POST("/move", dr.move)
	// deleteEntry entry
	r.DELETE("/entry/*path", dr.deleteEntry)
	// get upload config
	r.POST("/upload/*path", dr.upload)
	// write file
	r.PUT("/content/*path", dr.writeContent)
	// chunk upload request
	r.POST("/chunk", dr.chunkUploadRequest)
	// chunk upload
	r.PUT("/chunk/:id/:seq", dr.chunkUpload)
	// chunk upload complete
	r.POST("/chunk-content/*path", dr.chunkUploadComplete)
	// delete chunk upload
	r.DELETE("/chunk/:id", dr.deleteChunkUpload)
	// search
	r.GET("/search/*path", dr.search)

	return nil
}

type driveRoute struct {
	config common.Config

	access   *drive.Access
	searcher *search.Service

	tokenStore    types.TokenStore
	chunkUploader *ChunkUploader
	thumbnail     *thumbnail.Maker
	runner        task.Runner
	signer        *utils.Signer

	options  *storage.OptionsDAO
	pathMeta *storage.PathMetaDAO
}

func (dr *driveRoute) getDrive(c *gin.Context) (types.IDrive, error) {
	session := GetSession(c)
	return dr.access.GetDrive(session)
}

func (dr *driveRoute) setPathPassword(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	password := c.Query("password")
	if e := dr._setPassword(c, path, password); e != nil {
		_ = c.Error(e)
		return
	}
}

func (dr *driveRoute) _setPassword(c *gin.Context, path, password string) error {
	if password == "" {
		return nil
	}
	if len(password) > 32 {
		password = password[:32]
	}
	meta, e := dr.pathMeta.GetMerged(path)
	if e != nil {
		return e
	}
	if meta == nil || meta.Password.V == "" {
		return nil
	}
	return UpdateSession(c, dr.tokenStore, func(session *types.Session) {
		session.Props["password:"+meta.Password.Path] = password
	})
}

func (dr *driveRoute) list(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}

	if e := dr._setPassword(c, path, c.Query("password")); e != nil {
		_ = c.Error(e)
		return
	}

	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	entries, e := d.List(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	session := GetSession(c)
	res := make([]entryJson, 0, len(entries)+1)
	res = append(res, *dr.newEntryJson(entry, session))
	for _, v := range entries {
		res = append(res, *dr.newEntryJson(v, session))
	}
	SetResult(c, res)
}

func (dr *driveRoute) get(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, dr.newEntryJson(entry, GetSession(c)))
}

func (dr *driveRoute) makeDir(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	entry, e := d.MakeDir(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, dr.newEntryJson(entry, GetSession(c)))
}

func (dr *driveRoute) copyEntry(c *gin.Context) {
	drive_, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(c.Request.Context(), from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	session := GetSession(c)
	override := utils.ToBool(c.Query("override"))
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Copy(ctx, fromEntry, to, override)
		if e != nil {
			return nil, e
		}
		return dr.newEntryJson(r, session), nil
	}, 2*time.Second, task.WithNameGroup(from+" -> "+to, "drive/copy"))

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) move(c *gin.Context) {
	drive_, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(c.Request.Context(), from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	session := GetSession(c)
	override := utils.ToBool(c.Query("override"))
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Move(ctx, fromEntry, to, override)
		if e != nil {
			return nil, e
		}
		return dr.newEntryJson(r, session), nil
	}, 2*time.Second, task.WithNameGroup(from+" -> "+to, "drive/move"))

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func checkCopyOrMove(from, to string) error {
	if from == to {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_same_path_not_allowed"))
	}
	if strings.HasPrefix(to, from) && utils.PathDepth(from) != utils.PathDepth(to) {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	}
	return nil
}

func (dr *driveRoute) deleteEntry(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		return nil, d.Delete(ctx, path)
	}, 2*time.Second, task.WithNameGroup(path, "drive/delete"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) upload(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	override := utils.ToBool(c.Query("override"))
	size := utils.ToInt64(c.Query("size"), -1)
	request := make(types.SM, 0)
	if e := c.Bind(&request); e != nil {
		_ = c.Error(e)
		return
	}
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	config, e := d.Upload(c.Request.Context(), path, size, override, request)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if config != nil {
		SetResult(c, uploadConfig{config.Provider, config.Path, config.Config})
	}
}

func (dr *driveRoute) getContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	file, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	useProxy := utils.ToBool(c.Query("proxy"))
	proxyMaxSize := dr.options.GetValue(maxProxySizeKey).DataSize(-1)

	if proxyMaxSize > 0 && file.Size() > proxyMaxSize {
		useProxy = false
	}
	if e := drive_util.DownloadIContent(c.Request.Context(), file, c.Writer, c.Request, useProxy); e != nil {
		_ = c.Error(e)
		return
	}
}

func (dr *driveRoute) zipDownload(c *gin.Context) {
	files := utils.SplitLines(c.PostForm("files"))
	if len(files) == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	prefix := c.PostForm("prefix")

	drive, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}

	entries := make([]types.IEntry, 0, len(files))
	for _, f := range files {
		if f == "" {
			continue
		}
		file := utils.CleanPath(f)
		entry, e := drive.Get(c.Request.Context(), file)
		if e != nil {
			_ = c.Error(e)
			return
		}
		entries = append(entries, entry)
	}

	ctx := task.NewTaskContext(c.Request.Context())

	entriesTrees := make([]drive_util.EntryTreeNode, 0, len(entries))
	for _, entry := range entries {
		rootNode, e := drive_util.BuildEntriesTree(ctx, entry, true)
		if e != nil {
			return
		}
		entriesTrees = append(entriesTrees, rootNode)
	}

	totalSize := ctx.GetTotal()
	maxAllowedSizeOpt := dr.options.GetValue(maxZipSizeKey)
	maxAllowSize := maxAllowedSizeOpt.DataSize(-1)
	if maxAllowSize > 0 && totalSize > maxAllowSize {
		_ = c.Error(err.NewNotAllowedMessageError(i18n.T("api.zip.size_exceed", string(maxAllowedSizeOpt))))
		return
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition",
		"attachment; filename=\""+
			url.QueryEscape(fmt.Sprintf("packaged_%d.zip", len(files)))+"\"")

	zipFile := zip.NewWriter(c.Writer)
	defer func() {
		_ = zipFile.Close()
	}()

	for _, node := range entriesTrees {
		if e := drive_util.VisitEntriesTree(node, func(entry types.IEntry) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			name := entry.Path()
			if entry.Type().IsDir() {
				name += "/"
			}

			if prefix != "" && strings.HasPrefix(name, prefix+"/") {
				name = strings.TrimPrefix(name, prefix+"/")
			}

			file, e := zipFile.Create(name)
			if e != nil {
				return e
			}
			if entry.Type().IsFile() {
				if e := drive_util.CopyIContent(task.NewContextWrapper(c.Request.Context()), entry, file); e != nil {
					return e
				}
			}
			return nil
		}); e != nil {
			return
		}
	}
}

func (dr *driveRoute) getThumbnail(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if entry.Meta().Props != nil && entry.Meta().Thumbnail != "" {
		c.Redirect(http.StatusFound, entry.Meta().Thumbnail)
		return
	}
	makeCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	file, e := dr.thumbnail.Make(
		makeCtx, dr.wrapEntryWithAccessKey(entry, c.Query(common.SignatureQueryKey)),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}
	defer func() { _ = file.Close() }()
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dr.config.Thumbnail.TTL.Seconds())))
	c.Header("Content-Type", file.MimeType())
	http.ServeContent(c.Writer, c.Request, "", file.ModTime(), file)
}

func (dr *driveRoute) writeContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	session := GetSession(c)
	override := utils.ToBool(c.Query("override"))
	size := utils.ToInt64(c.GetHeader("Content-Length"), -1)
	defer func() { _ = c.Request.Body.Close() }()
	file, e := drive_util.CopyReaderToTempFile(task.DummyContext(), c.Request.Body, dr.config.TempDir)
	if e != nil {
		_ = c.Error(e)
		return
	}
	stat, e := file.Stat()
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(e)
		return
	}
	if size != stat.Size() {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_file_size")))
		return
	}
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		tempFile := utils.NewTempFile(file)
		defer func() {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
		}()
		r, e := d.Save(ctx, path, size, override, tempFile)
		if e != nil {
			return nil, e
		}
		return dr.newEntryJson(r, session), nil
	}, 2*time.Second, task.WithNameGroup(path, "drive/write"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) chunkUploadRequest(c *gin.Context) {
	size := utils.ToInt64(c.Query("size"), -1)
	chunkSize := utils.ToInt64(c.Query("chunkSize"), -1)
	if size <= 0 || chunkSize <= 0 {
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_size_or_chunk_size")))
		return
	}
	upload, e := dr.chunkUploader.CreateUpload(size, chunkSize)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, upload)
}

func (dr *driveRoute) chunkUpload(c *gin.Context) {
	id := c.Param("id")
	seq, e := strconv.Atoi(c.Param("seq"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	if e := dr.chunkUploader.ChunkUpload(id, seq, c.Request.Body); e != nil {
		_ = c.Error(e)
	}
}

func (dr *driveRoute) chunkUploadComplete(c *gin.Context) {
	d, e := dr.getDrive(c)
	if e != nil {
		_ = c.Error(e)
		return
	}
	session := GetSession(c)
	override := utils.ToBool(c.Query("override"))
	path := utils.CleanPath(c.Param("path"))
	id := c.Query("id")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		file, e := dr.chunkUploader.CompleteUpload(id, ctx)
		if e != nil {
			return nil, e
		}
		stat, e := file.Stat()
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		ctx.Progress(0, true)
		tempFile := utils.NewTempFile(file)
		entry, e := d.Save(ctx, path, stat.Size(), override, tempFile)
		if e != nil {
			_ = tempFile.Close()
			return nil, e
		}
		_ = tempFile.Close()
		_ = dr.chunkUploader.DeleteUpload(id)
		return dr.newEntryJson(entry, session), nil
	}, 2*time.Second, task.WithNameGroup(path, "drive/chunk-merge"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) deleteChunkUpload(c *gin.Context) {
	id := c.Param("id")
	if e := dr.chunkUploader.DeleteUpload(id); e != nil {
		_ = c.Error(e)
	}
}

func (dr *driveRoute) search(c *gin.Context) {
	root := utils.CleanPath(c.Param("path"))
	query := c.Query("q")
	next := utils.ToInt(c.Query("next"), 0)

	chroot, e := dr.access.GetChroot(GetSession(c))
	if e != nil {
		_ = c.Error(e)
		return
	}
	if chroot != nil {
		var e error
		root, e = chroot.WrapPath(root)
		if e != nil {
			if err.IsNotFoundError(e) {
				SetResult(c, search.EmptySearchResult)
				return
			}
			_ = c.Error(e)
			return
		}
	}

	r, e := dr.searcher.Search(
		c.Request.Context(), root, query, next,
		dr.access.GetPerms().Filter(GetSession(c)),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}

	if chroot != nil {
		for i := range r.Items {
			item := &r.Items[i]
			item.Entry.Path = chroot.UnwrapPath(item.Entry.Path)
		}
	}

	SetResult(c, r)
}

func (dr *driveRoute) newEntryJson(e types.IEntry, s types.Session) *entryJson {
	entryMeta := e.Meta()
	meta := utils.MapCopy(entryMeta.Props, nil)
	meta["writable"] = entryMeta.Writable
	if entryMeta.Thumbnail != "" {
		meta["thumbnail"] = entryMeta.Thumbnail
	}
	if entryMeta.Thumbnail == "" {
		// thumbnail is true
		// so the thumbnail is generated by the entry self
		if te := thumbnail.GetWrappedThumbnailEntry(e); te != nil {
			meta["thumbnail"] = true
		}
	}
	meta["accessKey"] = MakeSignature(dr.signer, e.Path(), s.User.Username, dr.config.SignatureTTL)

	if !s.HasUserGroup(types.AdminUserGroup) {
		delete(meta, "mountAt")
	}
	return &entryJson{
		Path:    e.Path(),
		Name:    utils.PathBase(e.Path()),
		Type:    e.Type(),
		Size:    e.Size(),
		Meta:    meta,
		ModTime: e.ModTime(),
	}
}

func (dr *driveRoute) wrapEntryWithAccessKey(entry types.IEntry, accessKey string) types.IEntry {
	return drive_util.WrapEntryWithMeta(entry, types.M{"accessKey": accessKey})
}

type entryJson struct {
	Path    string          `json:"path"`
	Name    string          `json:"name"`
	Type    types.EntryType `json:"type"`
	Size    int64           `json:"size"`
	Meta    types.M         `json:"meta"`
	ModTime int64           `json:"modTime"`
}

type uploadConfig struct {
	Provider string      `json:"provider"`
	Path     string      `json:"path,omitempty"`
	Config   interface{} `json:"config"`
}
