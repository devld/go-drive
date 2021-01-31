package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var Unprocessed = errors.New("unprocessed")

func NewRootFileSystem(root string,
	preProcess func(string, http.File) (string, error)) http.FileSystem {
	return rootFs{root: http.Dir(root), preprocess: preProcess}
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
	if e == Unprocessed {
		return file, nil
	}
	defer func() { _ = file.Close() }()
	if e != nil {
		return nil, e
	}
	return newIndexFile(content), nil
}

func newIndexFile(content string) http.File {
	r := strings.NewReader(content)
	return indexFile{
		s:    r,
		info: &indexStat{size: int64(r.Len())},
	}
}

type indexFile struct {
	s    io.ReadSeeker
	info *indexStat
}

func (i indexFile) Close() error {
	return nil
}

func (i indexFile) Read(p []byte) (n int, err error) {
	return i.s.Read(p)
}

func (i indexFile) Seek(offset int64, whence int) (int64, error) {
	return i.s.Seek(offset, whence)
}

func (i indexFile) Readdir(int) ([]os.FileInfo, error) {
	panic("")
}

func (i indexFile) Stat() (os.FileInfo, error) {
	return i.info, nil
}

type indexStat struct {
	size int64
}

func (i *indexStat) Name() string {
	return "index.html"
}

func (i *indexStat) Size() int64 {
	return i.size
}

func (i *indexStat) Mode() os.FileMode {
	return 0444
}

func (i *indexStat) ModTime() time.Time {
	return time.Now()
}

func (i *indexStat) IsDir() bool {
	return false
}

func (i *indexStat) Sys() interface{} {
	return nil
}
