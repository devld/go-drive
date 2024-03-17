package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"go-drive/common"
	"go-drive/storage"
	"io"
	"io/fs"
	"log"
	"net/http"
	path2 "path"
	"strconv"
	"strings"
	"sync"
	tTmpl "text/template"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

func isAllowedTemplate(name string) bool {
	ext := strings.ToLower(path2.Ext(name))
	// currently only html is allowed
	return ext == ".html"
}

func setFileCacheControl(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	if name == "/" {
		name = "/index.html"
	}
	if !isAllowedTemplate(name) {
		w.Header().Add("Cache-Control", " public, max-age=31536000")
	}
}

func newWebFiles(webDir string, config common.Config, options *storage.OptionsDAO) http.HandlerFunc {
	data := templateData{Options: options, Config: &config}

	tp := newTemplateProcessor(isAllowedTemplate)
	fileProcess := func(path string, file http.File) ([]byte, error) { return tp.Process(path, file, data) }

	handler := http.FileServer(&rootFs{root: http.Dir(webDir), fileProcess: fileProcess})

	return func(w http.ResponseWriter, r *http.Request) {
		setFileCacheControl(w, r)
		handler.ServeHTTP(w, r)
	}
}

// templateData is provided to the template
type templateData struct {
	Options *storage.OptionsDAO
	Config  *common.Config
}

// Json is called by html template
func (templateData) Json(o interface{}) (string, error) {
	s, e := json.Marshal(o)
	if e != nil {
		return "", e
	}
	return string(s), nil
}

var ErrUnprocessed = errors.New("unprocessed")

type rootFs struct {
	root        http.FileSystem
	fileProcess func(string, http.File) ([]byte, error)
}

func (r *rootFs) Open(name string) (http.File, error) {
	file, e := r.root.Open(name)
	if e != nil {
		return nil, e
	}
	content, e := r.fileProcess(name, file)
	if e == ErrUnprocessed {
		return file, nil
	}
	defer func() { _ = file.Close() }()
	if e != nil {
		log.Println("error processing file", e)
		return nil, e
	}
	return newProcessedFile(name, content), nil
}

func newTemplateProcessor(filter func(name string) bool) *templateProcessor {
	return &templateProcessor{
		filter: filter,
		cache:  cmap.New[*templateCache](),
	}
}

type templateProcessor struct {
	filter func(name string) bool
	cache  cmap.ConcurrentMap[string, *templateCache]
}

func (tp *templateProcessor) Process(path string, file http.File, data interface{}) ([]byte, error) {
	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}

	if stat.IsDir() || (tp.filter != nil && !tp.filter(path)) {
		return nil, ErrUnprocessed
	}

	name := stat.Name()
	tag := strconv.FormatInt(stat.ModTime().UnixMilli(), 10) + strconv.FormatInt(stat.Size(), 10)

	cached, ok := tp.cache.Get(name)
	if !ok {
		cached = newTemplateCache(stat.Name())
		tp.cache.Set(name, cached)
	}

	if tag != cached.tag {
		cached.l.Lock()
		defer cached.l.Unlock()
		content, e := io.ReadAll(file)
		if e != nil {
			return nil, e
		}
		if e := cached.Parse(string(content)); e != nil {
			return nil, e
		}
		cached.tag = tag
	}

	buf := bytes.NewBuffer(nil)

	if e := cached.Execute(buf, data); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}

func newTemplateCache(name string) *templateCache {
	return &templateCache{name: strings.ToLower(name), l: sync.Mutex{}}
}

type templateCache struct {
	t *tTmpl.Template

	name string
	tag  string
	l    sync.Mutex
}

func (c *templateCache) Execute(w io.Writer, data interface{}) error {
	if c.t == nil {
		panic("not initialized")
	}
	return c.t.Execute(w, data)
}

func (c *templateCache) Parse(text string) error {
	c.t = tTmpl.New("")
	_, e := c.t.Parse(text)
	return e
}

func newProcessedFile(name string, content []byte) http.File {
	r := bytes.NewReader(content)
	return &processedFile{name: name, ReadSeeker: r, size: r.Size()}
}

type processedFile struct {
	io.ReadSeeker
	name string
	size int64
}

func (p processedFile) Close() error {
	return nil
}

func (p processedFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (p processedFile) Stat() (fs.FileInfo, error) {
	return processedFileStat{name: p.name, size: p.size}, nil
}

type processedFileStat struct {
	name string
	size int64
}

func (p processedFileStat) Name() string {
	return p.name
}

func (p processedFileStat) Size() int64 {
	return p.size
}

func (p processedFileStat) Mode() fs.FileMode {
	return 0444
}

func (p processedFileStat) ModTime() time.Time {
	return time.Now()
}

func (p processedFileStat) IsDir() bool {
	return false
}

func (p processedFileStat) Sys() interface{} {
	return nil
}
