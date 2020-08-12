package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"log"
	"net/http"
	fsPath "path"
	"strconv"
)

type driveRoute struct {
	d common.IDrive
}

func newDriveRoute(d common.IDrive) driveRoute {
	return driveRoute{d}
}

func (dr driveRoute) init(r *gin.Engine) {
	// list entries/drives
	r.GET("/entries/*path", func(c *gin.Context) {
		path := c.Param("path")
		list, e := dr.list(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, list)
	})

	// get entry info
	r.GET("/entry/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.get(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	// mkdir
	r.POST("/mkdir/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.makeDir(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	r.POST("/copy", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")
		entry, e := dr.copy(from, to)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	// move file
	r.POST("/move", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")
		entry, e := dr.move(from, to)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	// delete entry
	r.DELETE("/entry/*path", func(c *gin.Context) {
		path := c.Param("path")
		e := dr.delete(path)
		if e != nil {
			dr.handleError(e, c)
		}
	})

	r.POST("/upload/*path", func(c *gin.Context) {
		path := c.Param("path")
		overwrite := c.Query("overwrite")
		size, e := strconv.ParseInt(c.Query("size"), 10, 64)
		if e != nil || size < 0 {
			_ = c.AbortWithError(400, e)
			return
		}
		config, e := dr.upload(path, size, overwrite != "")
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, config)
	})

	// get file content
	r.GET("/content/*path", func(c *gin.Context) {
		path := c.Param("path")
		if e := dr.getContent(path, c.Writer, c.Request); e != nil {
			dr.handleError(e, c)
		}
	})

	// write file
	r.PUT("/content/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.writeContent(path, c.Request)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})
}

func (dr driveRoute) handleError(e error, c *gin.Context) {
	if _, ok := e.(common.NotFoundError); ok {
		c.AbortWithStatus(404)
		_, _ = c.Writer.Write([]byte(e.Error()))
		return
	}
	if _, ok := e.(common.NotAllowedError); ok {
		c.AbortWithStatus(403)
		_, _ = c.Writer.Write([]byte(e.Error()))
		return
	}
	log.Println("unknown error", e)
	_ = c.AbortWithError(500, e)
}

func (dr driveRoute) list(path string) ([]entryJson, error) {
	path = fsPath.Clean(path)
	entries, e := dr.d.List(path)
	if e != nil {
		return nil, e
	}
	res := make([]entryJson, 0, len(entries))
	for _, v := range entries {
		res = append(res, *newEntryJson(v))
	}
	return res, nil
}

func (dr driveRoute) get(path string) (*entryJson, error) {
	path = fsPath.Clean(path)
	entry, e := dr.d.Get(path)
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func (dr driveRoute) makeDir(path string) (*entryJson, error) {
	path = fsPath.Clean(path)
	entry, e := dr.d.MakeDir(path)
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func (dr driveRoute) copy(from string, to string) (*entryJson, error) {
	fromEntry, e := dr.d.Get(from)
	if e != nil {
		return nil, e
	}
	entry, e := dr.d.Copy(fromEntry, to, func(loaded int64) {})
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func (dr driveRoute) move(from string, to string) (*entryJson, error) {
	from = fsPath.Clean(from)
	to = fsPath.Clean(to)
	entry, e := dr.d.Move(from, to)
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

func (dr driveRoute) delete(path string) error {
	path = fsPath.Clean(path)
	return dr.d.Delete(path)
}

func (dr driveRoute) upload(path string, size int64, overwrite bool) (*uploadConfig, error) {
	config, err := dr.d.Upload(path, size, overwrite)
	if err != nil {
		return nil, err
	}
	return newUploadConfig(config), nil
}

func (dr driveRoute) getContent(path string, w http.ResponseWriter, req *http.Request) error {
	path = fsPath.Clean(path)
	file, e := dr.d.Get(path)
	if e != nil {
		return e
	}
	content, ok := file.(common.IContent)
	if !ok {
		return common.NewNotAllowedError()
	}
	return common.DownloadIContent(content, w, req)
}

func (dr driveRoute) writeContent(path string, req *http.Request) (*entryJson, error) {
	file, _, err := req.FormFile("file")
	if err != nil {
		return nil, err
	}
	path = fsPath.Clean(path)
	entry, e := dr.d.Save(path, file, func(loaded int64) {})
	if e != nil {
		return nil, e
	}
	return newEntryJson(entry), nil
}

type entryJson struct {
	Path      string                 `json:"path"`
	Name      string                 `json:"name"`
	Type      common.EntryType       `json:"type"`
	Size      int64                  `json:"size"`
	Meta      map[string]interface{} `json:"meta"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
}

func newEntryJson(e common.IEntry) *entryJson {
	entryMeta := e.Meta()
	meta := make(map[string]interface{})
	meta["can_write"] = entryMeta.CanWrite()
	meta["can_read"] = entryMeta.CanRead()
	if entryMeta != nil {
		for k, v := range entryMeta.Props() {
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

func newUploadConfig(c *common.DriveUploadConfig) *uploadConfig {
	return &uploadConfig{c.Provider, c.Config}
}
