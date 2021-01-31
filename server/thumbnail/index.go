package thumbnail

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Jeffail/tunny"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Thumbnail interface {
	io.ReadSeeker
	io.Closer
	ModTime() time.Time
	Size() int64
	MimeType() string
	Name() string
}

const FolderType = "/"

const localSuffix = ".lock"

var errMaking = errors.New("making")

// TypeHandler creates thumbnail for entry, and writes to dest
type TypeHandler struct {
	Create   func(ctx context.Context, entry types.IEntry, dest io.Writer) error
	MimeType string
	Name     string
}

// handlers is a registry for TypeHandler, which the key is like 'jpg'
var handlers = make(map[string]TypeHandler)

func Register(ext string, h TypeHandler) {
	ext = strings.ToLower(ext)
	if _, ok := handlers[ext]; ok {
		panic(fmt.Sprintf("handler for %s already registered", ext))
	}
	handlers[ext] = h
}

type Maker struct {
	cacheDir string

	pool *tunny.Pool

	validity    time.Duration
	stopCleaner func()
}

func NewMaker(config common.Config, ch *registry.ComponentsHolder) (*Maker, error) {
	dir, e := config.GetDir("thumbnails", true)
	if e != nil {
		return nil, e
	}

	m := &Maker{
		cacheDir: dir,
		validity: config.ThumbnailTTL,
	}

	_ = filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, e error) error {
		if e == nil && !info.IsDir() && strings.HasSuffix(info.Name(), localSuffix) {
			_ = os.Remove(filepath.Join(m.cacheDir, info.Name()))
		}
		return nil
	})

	m.stopCleaner = utils.TimeTick(m.clean, 12*time.Hour)
	m.pool = tunny.NewFunc(config.ThumbnailConcurrent, m.executeTask)
	ch.Add("thumbnail", m)
	return m, nil
}

func (m *Maker) Make(ctx context.Context, entry types.IEntry) (Thumbnail, error) {
	fType := ""
	if entry.Type().IsDir() {
		fType = FolderType
	} else {
		fType = utils.PathExt(entry.Path())
	}

	h, ok := handlers[fType]
	if !ok {
		return nil, err.NewNotFoundError()
	}

	file, e := m.getItem(entry)
	if e != nil {
		return nil, e
	}
	exists, e := utils.FileExists(file)
	if e != nil {
		return nil, e
	}
	if exists {
		expired, e := m.isThumbnailExpired(entry, file)
		if e != nil {
			return nil, e
		}
		if expired {
			if e := m.remove(file); e != nil {
				return nil, e
			}
		} else {
			return m.openFile(file, h.MimeType, h.Name)
		}
	}

	task := &taskWrapper{h: h, entry: entry, dest: file}

	c := make(chan error)
	go func() {
		taskErr := m.pool.Process(task)
		if e == errMaking {
			c <- waitPendingTask(ctx, task.dest, task.dest+localSuffix)
			return
		}
		if e == nil {
			c <- nil
		} else {
			c <- taskErr.(error)
		}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("timeout")
	case e = <-c:
		if e != nil {
			return nil, e
		}
	}

	return m.openFile(file, h.MimeType, h.Name)
}

func (m *Maker) executeTask(v interface{}) interface{} {
	task := v.(*taskWrapper)
	exists, e := utils.FileExists(task.dest)
	if e != nil {
		return e
	}
	if exists {
		return nil
	}

	lockFile := task.dest + localSuffix
	exists, e = utils.FileExists(lockFile)
	if e != nil {
		return e
	}
	if exists {
		return errMaking
	}

	item, e := m.createItem(task.entry, lockFile)
	if e != nil {
		return e
	}

	e = task.h.Create(context.Background(), task.entry, item)
	_ = item.Close()
	if e == nil {
		e = os.Rename(lockFile, task.dest)
	}

	if e != nil {
		_ = os.Remove(lockFile)
		_ = os.Remove(task.dest)
	}
	return e
}

func waitPendingTask(ctx context.Context, path string, lock string) error {
	c := make(chan error)
	canceled := false

	go func() {
		for !canceled {
			exists, e := utils.FileExists(lock)
			if e != nil {
				c <- e
				break
			}
			if !exists {
				close(c)
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	var e error
	select {
	case <-ctx.Done():
		canceled = true
		e = ctx.Err()
	case e = <-c:
	}
	if e != nil {
		return e
	}
	exists, e := utils.FileExists(path)
	if e != nil {
		return e
	}
	if !exists {
		return errors.New("failed to create thumbnail")
	}
	return nil
}

// fileOffset is the start of thumbnail file
// | isDir(1) | modTime(8) | size(8) |
const fileOffset = 1 + 8 + 8

func (m *Maker) openFile(path, mimeType, name string) (Thumbnail, error) {
	file, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	f, e := newThumbnailFile(fileOffset, file)
	if e != nil {
		_ = file.Close()
		return nil, e
	}
	f.mimeType = mimeType
	f.name = name
	return f, nil
}

func (m *Maker) isThumbnailExpired(entry types.IEntry, path string) (bool, error) {
	file, e := os.Open(path)
	if e != nil {
		return false, e
	}
	defer func() { _ = file.Close() }()
	var isDir bool
	var modTime, size int64
	e = binary.Read(file, binary.LittleEndian, &isDir)
	if e != nil {
		return false, e
	}
	e = binary.Read(file, binary.LittleEndian, &modTime)
	if e != nil {
		return false, e
	}
	e = binary.Read(file, binary.LittleEndian, &size)
	if e != nil {
		return false, e
	}
	return entry.Type().IsDir() != isDir || entry.ModTime() != modTime || entry.Size() != size, nil
}

func (m *Maker) createItem(entry types.IEntry, path string) (io.WriteCloser, error) {
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if e != nil {
		return nil, e
	}
	defer func() {
		if e != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()
	e = binary.Write(file, binary.LittleEndian, entry.Type().IsDir())
	if e != nil {
		return nil, e
	}
	e = binary.Write(file, binary.LittleEndian, entry.ModTime())
	if e != nil {
		return nil, e
	}
	e = binary.Write(file, binary.LittleEndian, entry.Size())
	if e != nil {
		return nil, e
	}

	f, e := newThumbnailFile(fileOffset, file)
	if e != nil {
		return nil, e
	}

	return f, nil
}

func (m *Maker) remove(path string) error {
	return os.Remove(path)
}

func (m *Maker) getItem(entry types.IEntry) (string, error) {
	key := md5.Sum([]byte(entry.Path()))
	return filepath.Join(m.cacheDir, fmt.Sprintf("%x", key)), nil
}

func (m *Maker) clean() {
	n := 0
	notBefore := time.Now().Add(-m.validity)
	e := filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(notBefore) {
			if e := os.Remove(path); e != nil {
				log.Println("failed to delete file", e)
			}
			n++
		}
		return nil
	})
	if n > 0 {
		log.Println(fmt.Sprintf("%d expired thumbnails cleaned", n))
	}
	if e != nil {
		log.Println("error when cleaning expired thumbnails", e)
	}
}

func (m *Maker) Dispose() error {
	m.stopCleaner()
	m.pool.Close()
	return nil
}

func (m *Maker) SysConfig() (string, types.M, error) {
	ext := make(types.M)
	for k := range handlers {
		ext[k] = true
	}
	return "thumbnail", types.M{
		"extensions": ext,
	}, nil
}

type taskWrapper struct {
	h     TypeHandler
	entry types.IEntry
	dest  string
}

func newThumbnailFile(start int64, file *os.File) (*thumbnailFile, error) {
	stat, e := os.Stat(file.Name())
	if e != nil {
		return nil, e
	}
	_, e = file.Seek(start, io.SeekStart)
	if e != nil {
		return nil, e
	}
	return &thumbnailFile{
		start:   start,
		size:    stat.Size() - start,
		modTime: stat.ModTime(),
		file:    file,
	}, nil
}

type thumbnailFile struct {
	start   int64
	size    int64
	modTime time.Time
	file    *os.File
	current int64

	mimeType string
	name     string
}

func (l *thumbnailFile) Read(p []byte) (n int, err error) {
	return l.file.Read(p)
}

func (l *thumbnailFile) Write(p []byte) (n int, err error) {
	return l.file.Write(p)
}

func (l *thumbnailFile) Seek(offset int64, whence int) (int64, error) {
	c := l.current
	switch whence {
	case io.SeekStart:
		c = offset
	case io.SeekCurrent:
		c += offset
	case io.SeekEnd:
		c = l.size + offset
	}
	_, e := l.file.Seek(c+l.start, io.SeekStart)
	if e != nil {
		return 0, e
	}
	l.current = c
	return l.current, nil
}

func (l *thumbnailFile) Close() error {
	return l.file.Close()
}

func (l *thumbnailFile) ModTime() time.Time {
	return l.modTime
}

func (l *thumbnailFile) Size() int64 {
	return l.size
}

func (l *thumbnailFile) MimeType() string {
	return l.mimeType
}

func (l *thumbnailFile) Name() string {
	return l.name
}
