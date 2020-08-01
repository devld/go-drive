package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	fsPath "path"
	"time"
)

type DriveRoute struct {
	d common.IDrive
}

func NewDriveRoute(d common.IDrive) DriveRoute {
	return DriveRoute{d}
}

func (dr DriveRoute) Init(r *gin.Engine) {
	// list entries/drives
	r.GET("/entries/*path", func(c *gin.Context) {
		path := c.Param("path")
		list, e := dr.List(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, list)
	})

	// get entry info
	r.GET("/entry/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.Get(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	// mkdir
	r.POST("/mkdir/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.MakeDir(path)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	r.POST("/copy", func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")
		entry, e := dr.Copy(from, to)
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
		entry, e := dr.Move(from, to)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})

	// delete entry
	r.DELETE("/entry/*path", func(c *gin.Context) {
		path := c.Param("path")
		e := dr.Delete(path)
		if e != nil {
			dr.handleError(e, c)
		}
	})

	r.GET("/upload/*path", func(c *gin.Context) {
		path := c.Param("path")
		overwrite := c.Query("overwrite")
		config, e := dr.Upload(path, overwrite != "")
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, config)
	})

	// get file content
	r.GET("/content/*path", func(c *gin.Context) {
		path := c.Param("path")
		if e := dr.GetContent(path, c.Writer, c.Request); e != nil {
			dr.handleError(e, c)
		}
	})

	// write file
	r.PUT("/content/*path", func(c *gin.Context) {
		path := c.Param("path")
		entry, e := dr.WriteContent(path, c.Request)
		if e != nil {
			dr.handleError(e, c)
			return
		}
		c.JSON(200, entry)
	})
}

func (dr DriveRoute) handleError(e error, c *gin.Context) {
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

func (dr DriveRoute) List(path string) ([]EntryJson, error) {
	path = fsPath.Clean(path)
	entries, e := dr.d.List(path)
	if e != nil {
		return nil, e
	}
	res := make([]EntryJson, 0, len(entries))
	for _, v := range entries {
		res = append(res, *NewEntryJson(v))
	}
	return res, nil
}

func (dr DriveRoute) Get(path string) (*EntryJson, error) {
	path = fsPath.Clean(path)
	entry, e := dr.d.Get(path)
	if e != nil {
		return nil, e
	}
	return NewEntryJson(entry), nil
}

func (dr DriveRoute) MakeDir(path string) (*EntryJson, error) {
	path = fsPath.Clean(path)
	entry, e := dr.d.MakeDir(path)
	if e != nil {
		return nil, e
	}
	return NewEntryJson(entry), nil
}

func (dr DriveRoute) Copy(from string, to string) (*EntryJson, error) {
	fromEntry, e := dr.d.Get(from)
	if e != nil {
		return nil, e
	}
	entry, e := dr.d.Copy(fromEntry, to, func(loaded int64) {})
	if e != nil {
		return nil, e
	}
	return NewEntryJson(entry), nil
}

func (dr DriveRoute) Move(from string, to string) (*EntryJson, error) {
	from = fsPath.Clean(from)
	to = fsPath.Clean(to)
	entry, e := dr.d.Move(from, to)
	if e != nil {
		return nil, e
	}
	return NewEntryJson(entry), nil
}

func (dr DriveRoute) Delete(path string) error {
	path = fsPath.Clean(path)
	return dr.d.Delete(path)
}

func (dr DriveRoute) Upload(path string, overwrite bool) (*UploadConfig, error) {
	config, err := dr.d.Upload(path, overwrite)
	if err != nil {
		return nil, err
	}
	return NewUploadConfig(config), nil
}

func (dr DriveRoute) GetContent(path string, w http.ResponseWriter, req *http.Request) error {
	path = fsPath.Clean(path)
	file, e := dr.d.Get(path)
	if e != nil {
		return e
	}
	url, proxy, e := file.GetURL()
	if e == nil {
		if proxy {
			e = dr.proxyRequest(url, w, req)
		} else {
			w.WriteHeader(302)
			w.Header().Set("Location", url)
		}
		return e
	}
	reader, e := file.GetReader()
	if e != nil {
		return e
	}
	defer func() { _ = reader.Close() }()
	readSeeker, ok := reader.(io.ReadSeeker)
	if ok {
		http.ServeContent(
			w, req, file.Name(),
			time.Unix(0, file.UpdatedAt()*int64(time.Millisecond)),
			readSeeker)
		return nil
	}

	w.Header().Set("Content-Length", string(file.Size()))
	_, e = io.Copy(w, reader)
	return e
}

func (dr DriveRoute) WriteContent(path string, req *http.Request) (*EntryJson, error) {
	file, _, err := req.FormFile("file")
	if err != nil {
		return nil, err
	}
	path = fsPath.Clean(path)
	entry, e := dr.d.Save(path, file, func(loaded int64) {})
	if e != nil {
		return nil, e
	}
	return NewEntryJson(entry), nil
}

func (dr DriveRoute) proxyRequest(url string, w http.ResponseWriter, req *http.Request) error {
	dest, e := url2.Parse(url)
	if e != nil {
		return e
	}
	proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
		r.URL = dest
		r.Header.Set("Host", dest.Host)
		r.Header.Del("Referer")
	}}

	proxy.ServeHTTP(w, req)
	return nil
}
