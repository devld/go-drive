package server

import (
	"crypto/md5"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/nfnt/resize"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/task"
	"go-drive/common/types"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	path2 "path"
	"path/filepath"
	"strings"
	"time"
)

const (
	thumbnailSize    = 320
	thumbnailQuality = 50
	thumbnailTimeout = 30 * time.Second
)

var supportedExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
}

func init() {
	common.R().Register("thumbnail", func(c *common.ComponentRegistry) interface{} {
		config := c.Get("config").(common.Config)
		dir, e := config.GetDir("thumbnails", true)
		common.PanicIfError(e)
		return NewThumbnail(dir, config.ThumbnailCacheTTl, config.ThumbnailConcurrent)
	}, 0)
}

type Thumbnail struct {
	cacheDir    string
	validity    time.Duration
	pool        *tunny.Pool
	stopCleaner func()
}

func NewThumbnail(dir string, validity time.Duration, concurrent int) *Thumbnail {
	t := &Thumbnail{
		cacheDir: dir,
		validity: validity,
	}
	t.pool = tunny.NewFunc(concurrent, t.createThumbnail_)
	t.stopCleaner = common.TimeTick(t.clean, 12*time.Hour)
	return t
}

func (t *Thumbnail) Create(entry types.IEntry) (*os.File, error) {
	if !supportedExtensions[path2.Ext(entry.Path())] {
		return nil, common.NewNotFoundError()
	}
	filePath := t.getFile(entry.Path())
	file, e := t.getCache(filePath)
	if e != nil {
		return nil, e
	}
	if file != nil {
		return file, nil
	}
	content, ok := entry.(types.IContent)
	if !ok {
		return nil, common.NewNotAllowedError()
	}
	r, e := t.pool.ProcessTimed(thumbnailTask{path: filePath, content: content}, thumbnailTimeout)
	if e == tunny.ErrJobTimedOut {
		return nil, common.NewTimeoutError("timeout")
	}
	if r != nil {
		return nil, r.(error)
	}
	return os.Open(filePath)
}

func (t *Thumbnail) Remove(path string) error {
	filePath := t.getFile(path)
	e := os.Remove(filePath)
	if os.IsNotExist(e) {
		return nil
	}
	return e
}

func (t *Thumbnail) createThumbnail_(payload interface{}) interface{} {
	tTask := payload.(thumbnailTask)
	return t.createThumbnail(tTask.content, tTask.path)
}

func (t *Thumbnail) createThumbnail(content types.IContent, filePath string) error {
	tempFile, e := drive_util.CopyIContentToTempFile(content, task.DummyContext())
	if e != nil {
		return e
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	img, _, e := image.Decode(tempFile)
	if e != nil {
		return e
	}
	resizedImg := resize.Thumbnail(thumbnailSize, thumbnailSize, img, resize.NearestNeighbor)
	dstFile, e := ioutil.TempFile(t.cacheDir, "temp-")
	if e != nil {
		return e
	}
	if e := jpeg.Encode(dstFile, resizedImg, &jpeg.Options{Quality: thumbnailQuality}); e != nil {
		_ = dstFile.Close()
		_ = os.Remove(dstFile.Name())
		return e
	}
	_ = dstFile.Close()
	if e := os.Rename(dstFile.Name(), filePath); e != nil {
		_ = os.Remove(dstFile.Name())
		return e
	}
	return nil
}

func (t *Thumbnail) getCache(filePath string) (*os.File, error) {
	stat, e := os.Stat(filePath)
	if e == nil {
		if !t.isExpired(stat.ModTime()) {
			return os.Open(filePath)
		}
	}
	if !os.IsNotExist(e) {
		return nil, e
	}
	return nil, nil
}

func (t *Thumbnail) isExpired(modTime time.Time) bool {
	return modTime.Before(time.Now().Add(-t.validity))
}

func (t *Thumbnail) getFile(path string) string {
	key := md5.Sum([]byte(path))
	return filepath.Join(t.cacheDir, fmt.Sprintf("%x.jpg", key))
}

func (t *Thumbnail) clean() {
	n := 0
	notBefore := time.Now().Add(-t.validity)
	e := filepath.Walk(t.cacheDir, func(path string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() || !strings.HasPrefix(filepath.Base(path), sessionPrefix) {
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

func (t *Thumbnail) Dispose() error {
	t.stopCleaner()
	t.pool.Close()
	return nil
}

type thumbnailTask struct {
	path    string
	content types.IContent
}
