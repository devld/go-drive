package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
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
		e := deleteEntry(c)
		if e != nil {
			_ = c.Error(e)
		}
	})

	r.POST("/upload/*path", func(c *gin.Context) {
		config, e := upload(c)
		writeResponse(c, e, config)
	})

	// write file
	r.PUT("/content/*path", func(c *gin.Context) {
		entry, e := writeContent(c)
		writeResponse(c, e, entry)
	})

	// get task
	r.GET("/task/:id", func(c *gin.Context) {
		t, e := GetTaskRunner(c).GetTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = common.NewNotFoundError(e.Error())
		}
		writeResponse(c, e, t)
	})

	r.DELETE("/task/:id", func(c *gin.Context) {
		e := GetTaskRunner(c).RemoveTask(c.Param("id"))
		if e != nil && e == task.ErrorNotFound {
			e = common.NewNotFoundError(e.Error())
		}
		writeResponse(c, e, nil)
	})
}

func writeResponse(c *gin.Context, e error, result interface{}) {
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, result)
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
	t, e = GetTaskRunner(c).GetTask(t.Id)
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
	t, e = GetTaskRunner(c).GetTask(t.Id)
	return &t, e
}

func checkCopyOrMove(from, to string) error {
	if strings.HasPrefix(to, from) {
		return common.NewNotAllowedMessageError("not allowed")
	}
	return nil
}

func deleteEntry(c *gin.Context) error {
	path := common.CleanPath(c.Param("path"))
	return getDrive(c).Delete(path)
}

func upload(c *gin.Context) (*uploadConfig, error) {
	path := c.Param("path")
	override := c.Query("override")
	size, e := strconv.ParseInt(c.Query("size"), 10, 64)
	if e != nil || size < 0 {
		return nil, common.NewBadRequestError("invalid file size")
	}
	config, err := getDrive(c).Upload(path, size, override != "")
	if err != nil {
		return nil, err
	}
	return newUploadConfig(config), nil
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

type entryJson struct {
	Path      string                 `json:"path"`
	Name      string                 `json:"name"`
	Type      types.EntryType        `json:"type"`
	Size      int64                  `json:"size"`
	Meta      map[string]interface{} `json:"meta"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
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
		Path:      e.Path(),
		Name:      e.Name(),
		Type:      e.Type(),
		Size:      e.Size(),
		Meta:      meta,
		CreatedAt: e.CreatedAt(),
		UpdatedAt: e.UpdatedAt(),
	}
}

type uploadConfig struct {
	Provider string      `json:"provider"`
	Config   interface{} `json:"config"`
}

func newUploadConfig(c *types.DriveUploadConfig) *uploadConfig {
	return &uploadConfig{c.Provider, c.Config}
}
