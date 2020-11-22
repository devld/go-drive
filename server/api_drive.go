package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/task"
	"go-drive/common/types"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func InitDriveRoutes(router gin.IRouter) {

	// get file content
	router.HEAD("/content/*path", func(c *gin.Context) {
		if e := getContent(c); e != nil {
			_ = c.Error(e)
		}
	})
	router.GET("/content/*path", func(c *gin.Context) {
		if e := getContent(c); e != nil {
			_ = c.Error(e)
		}
	})
	router.GET("/thumbnail/*path", func(c *gin.Context) {
		if e := getThumbnail(c); e != nil {
			_ = c.Error(e)
		}
	})

	r := router.Group("/", Auth())

	// list entries/drives
	r.GET("/entries/*path", func(c *gin.Context) {
		list, e := list(c)
		writeResponse(c, e, list)
	})

	// get entry info
	r.GET("/entry/*path", func(c *gin.Context) {
		entry, e := get(c)
		writeResponse(c, e, entry)
	})

	// mkdir
	r.POST("/mkdir/*path", func(c *gin.Context) {
		entry, e := makeDir(c)
		writeResponse(c, e, entry)
	})

	// copy file
	r.POST("/copy", func(c *gin.Context) {
		t, e := copyEntry(c)
		writeResponse(c, e, t)
	})

	// move file
	r.POST("/move", func(c *gin.Context) {
		t, e := move(c)
		writeResponse(c, e, t)
	})

	// deleteEntry entry
	r.DELETE("/entry/*path", func(c *gin.Context) {
		t, e := deleteEntry(c)
		writeResponse(c, e, t)
	})

	// get upload config
	r.POST("/upload/*path", func(c *gin.Context) {
		config, e := upload(c)
		writeResponse(c, e, config)
	})

	// write file
	r.PUT("/content/*path", func(c *gin.Context) {
		entry, e := writeContent(c)
		writeResponse(c, e, entry)
	})

	// chunk upload request
	r.POST("/chunk", func(c *gin.Context) {
		upload, e := chunkUploadRequest(c)
		writeResponse(c, e, upload)
	})

	// chunk upload
	r.PUT("/chunk/:id/:seq", func(c *gin.Context) {
		e := chunkUpload(c)
		if e != nil {
			_ = c.Error(e)
		}
	})

	// chunk upload complete
	r.POST("/chunk-content/*path", func(c *gin.Context) {
		entry, e := chunkUploadComplete(c)
		writeResponse(c, e, entry)
	})

	// delete chunk upload
	r.DELETE("/chunk/:id", func(c *gin.Context) {
		id := c.Param("id")
		e := GetChunkUploader().DeleteUpload(id)
		if e != nil {
			_ = c.Error(e)
		}
	})

	// get task
	r.GET("/task/:id", func(c *gin.Context) {
		t, e := TaskRunner().GetTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = common.NewNotFoundMessageError(e.Error())
		}
		writeResponse(c, e, t)
	})

	// cancel and delete task
	r.DELETE("/task/:id", func(c *gin.Context) {
		_, e := TaskRunner().StopTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = common.NewNotFoundMessageError(e.Error())
		}
		if e != nil {
			_ = c.Error(e)
		}
	})
}

func writeResponse(c *gin.Context, e error, result interface{}) {
	if e != nil {
		_ = c.Error(e)
		return
	}
	if result != nil {
		SetResult(c, result)
	}
}

func getDrive(c *gin.Context) types.IDrive {
	session := GetSession(c)
	return NewPermissionWrapperDrive(
		c.Request, session,
		RootDrive().Get(),
		PermissionDAO(),
	)
}

func list(c *gin.Context) ([]entryJson, error) {
	path := common.CleanPath(c.Param("path"))
	entries, e := getDrive(c).List(path)
	if e != nil {
		return nil, e
	}
	res := make([]entryJson, 0, len(entries))
	for _, v := range entries {
		res = append(res, *newEntryJson(v))
	}
	return res, nil
}

func get(c *gin.Context) (*entryJson, error) {
	path := common.CleanPath(c.Param("path"))
	entry, e := getDrive(c).Get(path)
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func makeDir(c *gin.Context) (*entryJson, error) {
	path := common.CleanPath(c.Param("path"))
	entry, e := getDrive(c).MakeDir(path)
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func copyEntry(c *gin.Context) (*task.Task, error) {
	drive_ := getDrive(c)
	from := common.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(from)
	if e != nil {
		return nil, e
	}
	to := common.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		return nil, e
	}
	override := c.Query("override")
	t, e := TaskRunner().ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Copy(fromEntry, to, override != "", ctx)
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second)

	if e != nil {
		return nil, nil
	}
	return &t, e
}

func move(c *gin.Context) (*task.Task, error) {
	drive_ := getDrive(c)
	from := common.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(from)
	if e != nil {
		return nil, e
	}
	to := common.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		return nil, e
	}
	override := c.Query("override")
	t, e := TaskRunner().ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Move(fromEntry, to, override != "", ctx)
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second)

	if e != nil {
		return nil, nil
	}
	return &t, e
}

func checkCopyOrMove(from, to string) error {
	if from == to {
		return common.NewNotAllowedMessageError("Copy or move to same path is not allowed")
	}
	if strings.HasPrefix(to, from) && common.PathDepth(from) != common.PathDepth(to) {
		return common.NewNotAllowedMessageError("Copy or move to child path is not allowed")
	}
	return nil
}

func deleteEntry(c *gin.Context) (*task.Task, error) {
	path := common.CleanPath(c.Param("path"))
	t, e := TaskRunner().ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		return nil, getDrive(c).Delete(path, ctx)
	}, 2*time.Second)
	if e != nil {
		return nil, e
	}
	return &t, e
}

func upload(c *gin.Context) (*uploadConfig, error) {
	path := common.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := common.ToInt64(c.Query("size"), -1)
	request := make(types.SM, 0)
	if e := c.Bind(&request); e != nil {
		return nil, e
	}
	config, err := getDrive(c).Upload(path, size, override != "", request)
	if err != nil {
		return nil, err
	}
	return newUploadConfigJson(config), nil
}

func getContent(c *gin.Context) error {
	path := common.CleanPath(c.Param("path"))
	file, e := getDrive(c).Get(path)
	if e != nil {
		return e
	}
	if content, ok := file.(types.IContent); ok {

		useProxy := c.Query("proxy")
		maxProxySize := common.Conf().GetMaxProxySize()
		if maxProxySize > 0 && file.Size() > maxProxySize {
			useProxy = ""
		}

		return drive_util.DownloadIContent(content, c.Writer, c.Request, useProxy != "")
	}
	return common.NewNotAllowedError()
}

func getThumbnail(c *gin.Context) error {
	path := common.CleanPath(c.Param("path"))
	if !checkSignature(c.Request, path) {
		return common.NewNotFoundError()
	}
	entry, e := getDrive(c).Get(path)
	if e != nil {
		return e
	}
	if entry.Meta().Props != nil && entry.Meta().Thumbnail != "" {
		c.Redirect(http.StatusFound, entry.Meta().Thumbnail)
		return nil
	}
	if entry.Size() > common.Conf().GetMaxThumbnailSize() {
		return common.NewNotFoundError()
	}
	file, e := GetThumbnail().Create(entry)
	if e != nil {
		return e
	}
	defer func() { _ = file.Close() }()
	stat, e := file.Stat()
	if e != nil {
		return e
	}
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(common.Conf().ThumbnailCacheTTl.Seconds())))
	http.ServeContent(c.Writer, c.Request, "thumbnail.jpg", stat.ModTime(), file)
	return nil
}

func writeContent(c *gin.Context) (*task.Task, error) {
	path := common.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := common.ToInt64(c.GetHeader("Content-Length"), -1)
	defer func() { _ = c.Request.Body.Close() }()
	file, e := drive_util.CopyReaderToTempFile(c.Request.Body, task.DummyContext())
	if e != nil {
		return nil, e
	}
	stat, e := file.Stat()
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, e
	}
	if size != stat.Size() {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, common.NewBadRequestError("invalid file size")
	}
	t, e := TaskRunner().ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		defer func() {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}()
		return getDrive(c).Save(path, size, override != "", file, ctx)
	}, 2*time.Second)
	if e != nil {
		return nil, e
	}
	return &t, e
}

func chunkUploadRequest(c *gin.Context) (*ChunkUpload, error) {
	size := common.ToInt64(c.Query("size"), -1)
	chunkSize := common.ToInt64(c.Query("chunk_size"), -1)
	if size <= 0 || chunkSize <= 0 {
		return nil, common.NewBadRequestError("invalid size or chunk_size")
	}
	upload, e := GetChunkUploader().CreateUpload(size, chunkSize)
	if e != nil {
		return nil, e
	}
	return &upload, nil
}

func chunkUpload(c *gin.Context) error {
	id := c.Param("id")
	seq, e := strconv.Atoi(c.Param("seq"))
	if e != nil {
		return e
	}
	return GetChunkUploader().ChunkUpload(id, seq, c.Request.Body)
}

func chunkUploadComplete(c *gin.Context) (*task.Task, error) {
	path := common.CleanPath(c.Param("path"))
	id := c.Query("id")
	uploader := GetChunkUploader()
	t, e := TaskRunner().ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		file, e := uploader.CompleteUpload(id, ctx)
		if e != nil {
			return nil, e
		}
		stat, e := file.Stat()
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		ctx.Progress(0, true)
		entry, e := getDrive(c).Save(path, stat.Size(), true, file, ctx)
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		_ = file.Close()
		e = uploader.DeleteUpload(id)
		return newEntryJson(entry), nil
	}, 2*time.Second)
	if e != nil {
		return nil, e
	}
	return &t, nil
}

type entryJson struct {
	Path    string          `json:"path"`
	Name    string          `json:"name"`
	Type    types.EntryType `json:"type"`
	Size    int64           `json:"size"`
	Meta    types.M         `json:"meta"`
	ModTime int64           `json:"mod_time"`
}

func newEntryJson(e types.IEntry) *entryJson {
	entryMeta := e.Meta()
	meta := common.CopyMap(entryMeta.Props)
	meta["can_write"] = entryMeta.CanWrite
	return &entryJson{
		Path:    e.Path(),
		Name:    common.PathBase(e.Path()),
		Type:    e.Type(),
		Size:    e.Size(),
		Meta:    meta,
		ModTime: e.ModTime(),
	}
}

type uploadConfig struct {
	Provider string      `json:"provider"`
	Config   interface{} `json:"config"`
}

func newUploadConfigJson(c *types.DriveUploadConfig) *uploadConfig {
	if c == nil {
		return nil
	}
	return &uploadConfig{c.Provider, c.Config}
}
