package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func InitDriveRoutes(router gin.IRouter) {

	// get file content
	router.GET("/content/*path", func(c *gin.Context) {
		if e := getContent(c); e != nil {
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
		e := GetChunkUploader(c).DeleteUpload(id)
		if e != nil {
			_ = c.Error(e)
		}
	})

	// get task
	r.GET("/task/:id", func(c *gin.Context) {
		t, e := GetTaskRunner(c).GetTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = common.NewNotFoundMessageError(e.Error())
		}
		writeResponse(c, e, t)
	})

	// cancel and delete task
	r.DELETE("/task/:id", func(c *gin.Context) {
		e := GetTaskRunner(c).RemoveTask(c.Param("id"))
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
	if result != nil && !reflect.ValueOf(result).IsNil() {
		SetResult(c, result)
	}
}

func getDrive(c *gin.Context) types.IDrive {
	session := GetSession(c)
	return NewPermissionWrapperDrive(
		c.Request, session,
		GetRootDrive(c).Get(),
		GetPermissionStorage(c),
		GetRequestSigner(c),
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
	t, e := GetTaskRunner(c).ExecuteAndWait(func(ctx task.Context) (interface{}, error) {
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
	t, e := GetTaskRunner(c).ExecuteAndWait(func(ctx task.Context) (interface{}, error) {
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
	if strings.HasPrefix(to, from) {
		return common.NewNotAllowedMessageError("Copy to child path is not allowed")
	}
	return nil
}

func deleteEntry(c *gin.Context) (*task.Task, error) {
	path := common.CleanPath(c.Param("path"))
	t, e := GetTaskRunner(c).ExecuteAndWait(func(ctx task.Context) (interface{}, error) {
		return nil, getDrive(c).Delete(path, ctx)
	}, 2*time.Second)
	if e != nil {
		return nil, e
	}
	return &t, e
}

func upload(c *gin.Context) (*uploadConfig, error) {
	path := c.Param("path")
	override := c.Query("override")
	sizeStr := c.Query("size")
	var size int64 = -1
	var e error
	if sizeStr != "" {
		size, e = strconv.ParseInt(c.Query("size"), 10, 64)
		if e != nil || size < 0 {
			return nil, common.NewBadRequestError("invalid file size")
		}
	}
	request := make(map[string]string, 0)
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
		return common.DownloadIContent(content, c.Writer, c.Request)
	}
	return common.NewNotAllowedError()
}

func writeContent(c *gin.Context) (*entryJson, error) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, err
	}
	path := common.CleanPath(c.Param("path"))
	entry, e := getDrive(c).Save(path, file, task.DummyContext())
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func chunkUploadRequest(c *gin.Context) (*ChunkUpload, error) {
	size, e1 := strconv.ParseInt(c.Query("size"), 10, 64)
	chunkSize, e2 := strconv.ParseInt(c.Query("chunk_size"), 10, 64)
	if e1 != nil || e2 != nil {
		return nil, common.NewBadRequestError("invalid size or chunk_size")
	}
	upload, e := GetChunkUploader(c).CreateUpload(size, chunkSize)
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
	return GetChunkUploader(c).ChunkUpload(id, seq, c.Request.Body)
}

func chunkUploadComplete(c *gin.Context) (*task.Task, error) {
	path := common.CleanPath(c.Param("path"))
	id := c.Query("id")
	uploader := GetChunkUploader(c)
	t, e := GetTaskRunner(c).ExecuteAndWait(func(ctx task.Context) (interface{}, error) {
		file, e := uploader.CompleteUpload(id, ctx)
		if e != nil {
			return nil, e
		}
		entry, e := getDrive(c).Save(path, file, task.DummyContext())
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
	Path    string                 `json:"path"`
	Name    string                 `json:"name"`
	Type    types.EntryType        `json:"type"`
	Size    int64                  `json:"size"`
	Meta    map[string]interface{} `json:"meta"`
	ModTime int64                  `json:"mod_time"`
}

func newEntryJson(e types.IEntry) *entryJson {
	entryMeta := e.Meta()
	meta := make(map[string]interface{})
	meta["can_write"] = entryMeta.CanWrite
	if entryMeta.Props != nil {
		for k, v := range entryMeta.Props {
			meta[k] = v
		}
	}
	return &entryJson{
		Path:    e.Path(),
		Name:    e.Name(),
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
