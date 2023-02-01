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

var templateAllowedExt = map[string]bool{
	".html": true,
}

func newWebFiles(webDir string, config common.Config, options *storage.OptionsDAO) http.Handler {
	data := templateData{Options: options, Config: &config}

	tp := newTemplateProcessor(func(stat fs.FileInfo) bool {
		ok := templateAllowedExt[strings.ToLower(path2.Ext(stat.Name()))]
		return ok
	})
	preprocess := func(name string, file http.File) (string, error) {
		b, e := tp.Process(file, data)
		if e != nil {
			return "", e
		}
		return string(b), nil
	}
	return http.FileServer(newRootFileSystem(webDir, preprocess))
}

type templateData struct {
	Options *storage.OptionsDAO
	Config  *common.Config
}

func (templateData) Json(o interface{}) (string, error) {
	s, e := json.Marshal(o)
	if e != nil {
		return "", e
	}
	return string(s), nil
}

var ErrUnprocessed = errors.New("unprocessed")

func newRootFileSystem(root string,
	preprocess func(string, http.File) (string, error)) http.FileSystem {
	return rootFs{root: http.Dir(root), preprocess: preprocess}
}

type rootFs struct {
	root       http.FileSystem
	preprocess func(string, http.File) (string, error)
}

func (r rootFs) Open(name string) (http.File, error) {
	file, e := r.root.Open(name)
	if e != nil {
		return nil, e
	}
	content, e := r.preprocess(name, file)
	if e == ErrUnprocessed {
		return file, nil
	}
	defer func() { _ = file.Close() }()
	if e != nil {
		log.Println("error processing file", e)
		return nil, e
	}
	return newProcessedFile(name, []byte(content)), nil
}

func newTemplateProcessor(filter func(stat fs.FileInfo) bool) *templateProcessor {
	return &templateProcessor{
		filter: filter,
		cache:  cmap.New[*templateCache](),
	}
}

type templateProcessor struct {
	filter func(stat fs.FileInfo) bool
	cache  cmap.ConcurrentMap[string, *templateCache]
}

func (tp *templateProcessor) Process(file http.File, data interface{}) ([]byte, error) {
	stat, e := file.Stat()
	if e != nil {
		return nil, e
	}

	if tp.filter != nil && !tp.filter(stat) {
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
