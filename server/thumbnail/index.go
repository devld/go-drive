package thumbnail

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Jeffail/tunny"
	"github.com/bmatcuk/doublestar/v4"
)

const FolderType = "/"

const (
	lockSuffix           = ".lock"
	optKeyHandlerMapping = "thumbnail.handlersMapping"
)

var errMaking = errors.New("making")

type Maker struct {
	// handlers is a registry for TypeHandler, map[extension]map[handlerName]TypeHandler
	handlers map[string]map[string]TypeHandler

	cacheDir string
	apiPath  string

	pool    *tunny.Pool
	options *storage.OptionsDAO

	validity    time.Duration
	stopCleaner func()
}

func NewMaker(config common.Config, optionsDAO *storage.OptionsDAO,
	ch *registry.ComponentsHolder) (*Maker, error) {
	dir, e := config.GetTempDir("thumbnails", true)
	if e != nil {
		return nil, e
	}

	handlers, e := createHandlers(config.Thumbnail.Handlers)
	if e != nil {
		return nil, e
	}

	m := &Maker{
		handlers: handlers,
		options:  optionsDAO,
		cacheDir: dir,
		apiPath:  config.APIPath,
		validity: config.Thumbnail.TTL,
	}

	_ = filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, e error) error {
		if e == nil && !info.IsDir() && strings.HasSuffix(info.Name(), lockSuffix) {
			_ = os.Remove(filepath.Join(m.cacheDir, info.Name()))
		}
		return nil
	})

	m.stopCleaner = utils.TimeTick(m.clean, 12*time.Hour)
	m.pool = tunny.NewFunc(config.Thumbnail.Concurrent, m.executeTask)
	ch.Add("thumbnail", m)
	return m, nil
}

var tagRegexp = regexp.MustCompile("^[A-z0-9-_]*$")

func createHandlers(items []common.ThumbnailHandlerItem) (map[string]map[string]TypeHandler, error) {
	hs := make(map[string]map[string]TypeHandler)
	for _, item := range items {
		factory, ok := typeHandlerFactories[item.Type]
		if !ok {
			availableTypes := make([]string, 0, len(typeHandlerFactories))
			for t := range typeHandlerFactories {
				availableTypes = append(availableTypes, t)
			}

			return nil, fmt.Errorf("unknown handler type: %s. Available types are %v",
				item.Type, availableTypes)
		}
		h, e := factory(item.Config)
		if e != nil {
			return nil, errors.New("failed to create thumbnail type handler " + item.Type + ": " + e.Error())
		}
		for _, ext := range strings.Split(item.FileTypes, ",") {
			ext = strings.ToLower(strings.TrimSpace(ext))
			extHs, ok := hs[ext]
			if !ok {
				extHs = make(map[string]TypeHandler)
				hs[ext] = extHs
			}
			for _, tag := range strings.Split(item.Tags, ",") {
				if !tagRegexp.Match([]byte(tag)) {
					return nil, errors.New("invalid handler tag: " + tag)
				}
				tag = strings.ToLower(strings.TrimSpace(tag))
				extHs[tag] = h
			}
		}
	}
	return hs, nil
}

func (m *Maker) Make(ctx context.Context, entry types.IEntry) (Thumbnail, error) {
	thumbnailEntry := m.createThumbnailEntry(entry)

	itemPath := m.getItem(thumbnailEntry)
	t, e := m.tryToGetFromCache(thumbnailEntry, itemPath)
	if e != nil {
		return nil, e
	}
	if t != nil {
		return t, nil
	}

	return m.doMake(ctx, thumbnailEntry, itemPath)
}

func (m *Maker) createThumbnailEntry(entry types.IEntry) ThumbnailEntry {
	// we need to use the absolute path of this entry to generate thumbnail cache key
	// so we get the wrapped IDispatcherEntry here
	dispatcherEntry := drive_util.GetIEntry(entry, func(e types.IEntry) bool {
		_, ok := e.(types.IDispatcherEntry)
		return ok
	})
	if dispatcherEntry == nil {
		panic("types.IDispatcherEntry required")
	}

	externalURL := m.apiPath

	if entry.Type().IsDir() {
		externalURL += "/entries/"
	} else {
		externalURL += "/content/"
	}
	externalURL += entry.Path()

	ako, ok := entry.Meta().Props["accessKey"]
	ak := ""
	if ok {
		if t, ok := ako.(string); ok {
			ak = t
		}
	}
	if ak != "" {
		externalURL += "?" + common.SignatureQueryKey + "=" + url.QueryEscape(ak)
	}

	return &thumbnailEntry{
		IEntry:           entry,
		IDispatcherEntry: dispatcherEntry.(types.IDispatcherEntry),
		externalURL:      externalURL,
	}
}

func (m *Maker) resolveHandler(entry ThumbnailEntry) (TypeHandler, error) {
	te := GetWrappedThumbnailEntry(entry)
	if te != nil {
		return entrySelfThumbnailTypeHandler, nil
	}
	fType := ""
	if entry.Type().IsDir() {
		fType = FolderType
	} else {
		fType = utils.PathExt(entry.Path())
	}

	hs, ok := m.handlers[fType]
	if !ok {
		return nil, err.NewNotFoundError()
	}
	mappingStr, e := m.options.Get(optKeyHandlerMapping)
	if e != nil {
		return nil, e
	}
	for _, m := range utils.SplitLines(mappingStr) {
		if m == "" {
			continue
		}
		temp := strings.SplitN(m, ":", 2)
		if len(temp) != 2 {
			log.Println("invalid handler mapping:", m)
			continue
		}
		if ok, e := doublestar.Match(temp[1], entry.GetRealPath()); ok && e == nil {
			tags := strings.Split(temp[0], ",")
			for _, tag := range tags {
				if h, ok := hs[tag]; ok {
					return h, nil
				}
			}
		}
	}

	for _, h := range hs {
		return h, nil
	}

	panic("no handlers found")
}

func (m *Maker) tryToGetFromCache(entry ThumbnailEntry, path string) (Thumbnail, error) {
	exists, e := utils.FileExists(path)
	if e != nil {
		return nil, e
	}
	if exists {
		f, e := os.Open(path)
		if e != nil {
			return nil, e
		}
		meta, headerSize, e := m.readItemMeta(f)
		if e != nil || m.isThumbnailExpired(entry, *meta) {
			_ = f.Close()
			if e := m.remove(path); e != nil {
				return nil, e
			}
		} else {
			return m.openFile("", f, meta, headerSize)
		}
	}
	return nil, nil
}

func (m *Maker) doMake(ctx context.Context, entry ThumbnailEntry, path string) (Thumbnail, error) {
	h, e := m.resolveHandler(entry)
	if e != nil {
		return nil, e
	}
	task := &taskWrapper{h: h, entry: entry, dest: path}

	c := make(chan error)
	go func() {
		taskErr := m.pool.Process(task)
		if taskErr == errMaking {
			c <- waitPendingTask(ctx, task.dest, task.dest+lockSuffix)
			return
		}
		if taskErr == nil {
			c <- nil
		} else {
			c <- taskErr.(error)
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case e = <-c:
		if e != nil {
			return nil, e
		}
	}

	return m.openFile(path, nil, nil, 0)
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

	lockFile := task.dest + lockSuffix
	exists, e = utils.FileExists(lockFile)
	if e != nil {
		return e
	}
	if exists {
		return errMaking
	}

	item, e := m.createItem(task.entry, lockFile, task.h.MimeType())
	if e != nil {
		return e
	}

	ctx := context.Background()
	var ctxCancel context.CancelFunc
	if task.h.Timeout() > 0 {
		ctx, ctxCancel = context.WithTimeout(ctx, task.h.Timeout())
	}
	if ctxCancel != nil {
		defer ctxCancel()
	}

	e = task.h.CreateThumbnail(ctx, task.entry, item)
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

func (m *Maker) openFile(path string, file *os.File, meta *itemMeta, headerSize uint32) (Thumbnail, error) {
	var e error
	if file == nil {
		file, e = os.Open(path)
		if e != nil {
			return nil, e
		}
		meta, headerSize, e = m.readItemMeta(file)
		if e != nil {
			_ = file.Close()
			return nil, e
		}
	}

	f, e := newThumbnailFile(int64(headerSize), file, meta.MimeType)
	if e != nil {
		_ = file.Close()
		return nil, e
	}
	return f, nil
}

func (m *Maker) isThumbnailExpired(entry ThumbnailEntry, meta itemMeta) bool {
	return entry.Type().IsDir() != meta.IsDir ||
		entry.ModTime() != meta.ModTime ||
		entry.Size() != meta.Size
}

func (m *Maker) readItemMeta(file *os.File) (*itemMeta, uint32, error) {
	var meta itemMeta
	r := bufio.NewReader(file)
	var headerSize uint32
	if e := binary.Read(r, binary.LittleEndian, &headerSize); e != nil {
		return nil, headerSize, e
	}
	e := gob.NewDecoder(io.LimitReader(r, int64(headerSize))).Decode(&meta)
	return &meta, 4 + headerSize, e
}

func (m *Maker) createItem(entry ThumbnailEntry, path, mimeType string) (io.WriteCloser, error) {
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
	buf := bytes.NewBuffer(nil)
	if e := gob.NewEncoder(buf).Encode(itemMeta{
		IsDir:    entry.Type().IsDir(),
		Size:     entry.Size(),
		ModTime:  entry.ModTime(),
		MimeType: mimeType,
	}); e != nil {
		return nil, e
	}
	headerSize := uint32(buf.Len())
	if e := binary.Write(file, binary.LittleEndian, headerSize); e != nil {
		return nil, e
	}
	if _, e = file.Write(buf.Bytes()); e != nil {
		return nil, e
	}

	f, e := newThumbnailFile(int64(4+headerSize), file, mimeType)
	if e != nil {
		return nil, e
	}

	return f, nil
}

func (m *Maker) remove(path string) error {
	return os.Remove(path)
}

func (m *Maker) getItem(entry ThumbnailEntry) string {
	key := md5.Sum([]byte(entry.GetRealPath()))
	return filepath.Join(m.cacheDir, fmt.Sprintf("%x", key))
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
		log.Printf("%d expired thumbnails cleaned", n)
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
	for k := range m.handlers {
		ext[k] = true
	}
	return "thumbnail", types.M{
		"extensions": ext,
	}, nil
}

type itemMeta struct {
	IsDir    bool
	Size     int64
	ModTime  int64
	MimeType string
}

type taskWrapper struct {
	h     TypeHandler
	entry ThumbnailEntry
	dest  string
}

func newThumbnailFile(start int64, file *os.File, mimeType string) (*thumbnailFile, error) {
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

		mimeType: mimeType,
	}, nil
}

type thumbnailFile struct {
	start   int64
	size    int64
	modTime time.Time
	file    *os.File
	current int64

	mimeType string
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
