package server

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go-drive/common"
	"go-drive/common/task"
	"io"
	"io/ioutil"
	"math"
	"os"
	fsPath "path"
	"strconv"
	"strings"
)

const minChunkSize = 5 * 1024 * 1024

type ChunkUploader struct {
	dir string
}

func NewChunkUploader(dir string) (*ChunkUploader, error) {
	exists, e := common.FileExists(dir)
	if e != nil {
		return nil, e
	}
	if !exists {
		return nil, errors.New("root dir does not exists")
	}
	return &ChunkUploader{dir}, nil
}

func (c *ChunkUploader) CreateUpload(size, chunkSize int64) (ChunkUpload, error) {
	if chunkSize < minChunkSize {
		return ChunkUpload{},
			common.NewBadRequestError(fmt.Sprintf("chunk size cannot be less than %d", minChunkSize))
	}
	if size <= 0 {
		return ChunkUpload{}, common.NewBadRequestError("invalid file size")
	}
	id := c.generateUploadId(size, chunkSize)
	dir := c.getDir(id)
	exists, e := common.FileExists(dir)
	if e != nil {
		return ChunkUpload{}, e
	}
	if exists {
		return ChunkUpload{}, common.NewNotAllowedError()
	}
	if e := os.Mkdir(dir, 0755); e != nil {
		return ChunkUpload{}, e
	}
	upload, e := c.getUpload(id)
	if e != nil {
		panic(e)
	}
	return *upload, nil
}

func (c *ChunkUploader) ChunkUpload(id string, seq int, reader io.Reader) error {
	upload, e := c.getUpload(id)

	defer func() {
		if c.isMarkedDelete(upload) {
			_ = c.DeleteUpload(upload.Id)
		}
	}()

	if e != nil {
		return e
	}
	if seq < 0 || seq >= upload.Chunks {
		return common.NewNotAllowedMessageError("invalid chunk seq")
	}
	chunkSize := upload.ChunkSize
	if seq == upload.Chunks-1 {
		chunkSize = upload.Size % upload.ChunkSize
	}
	chunk, e := os.OpenFile(c.getChunk(upload, seq), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	success := false
	defer func() {
		_ = chunk.Close()
		if !success {
			_ = os.Remove(chunk.Name())
		}
	}()
	written, e := io.Copy(chunk, reader)
	if e != nil {
		return e
	}
	if written != chunkSize {
		return common.NewBadRequestError(fmt.Sprintf("expect %d bytes, but %d bytes received", chunkSize, written))
	}
	success = true
	return nil
}

func (c *ChunkUploader) CompleteUpload(id string, ctx task.Context) (*os.File, error) {
	upload, e := c.getUpload(id)
	if e != nil {
		return nil, e
	}
	for seq := 0; seq < upload.Chunks; seq++ {
		exists, e := common.FileExists(c.getChunk(upload, seq))
		if e != nil {
			return nil, e
		}
		if !exists {
			return nil, common.NewNotAllowedMessageError("missing chunks")
		}
	}
	file, e := os.OpenFile(c.getFile(upload), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if e != nil {
		return nil, e
	}
	allSuccess := false
	defer func() {
		_ = file.Close()
		if !allSuccess {
			_ = os.Remove(file.Name())
		}
		if c.isMarkedDelete(upload) {
			_ = c.DeleteUpload(upload.Id)
		}
	}()
	ctx.Total(upload.Size)
	var loaded int64 = 0
	for seq := 0; seq < upload.Chunks; seq++ {
		if ctx.Canceled() {
			return nil, task.ErrorCanceled
		}
		chunk, e := os.Open(c.getChunk(upload, seq))
		if e != nil {
			return nil, e
		}
		w, e := io.Copy(file, chunk)
		_ = chunk.Close()
		if c.isMarkedDelete(upload) {
			return nil, task.ErrorCanceled
		}
		if e != nil {
			return nil, e
		}
		loaded += w
		ctx.Progress(loaded)
	}
	allSuccess = true
	_ = file.Close()
	if !allSuccess {
		return nil, task.ErrorCanceled
	}
	return os.Open(c.getFile(upload))
}

func (c *ChunkUploader) DeleteUpload(id string) error {
	upload, e := c.getUpload(id)
	if e != nil {
		return e
	}
	e = os.RemoveAll(c.getDir(id))
	if e != nil {
		if e = c.markDeleted(upload); e != nil {
			return e
		}
	}
	return nil
}

func (c ChunkUploader) generateUploadId(size, chunkSize int64) string {
	return fmt.Sprintf("%s_%d_%d", uuid.New().String(), size, chunkSize)
}

func (c *ChunkUploader) getUpload(id string) (*ChunkUpload, error) {
	temp := strings.Split(id, "_")
	if len(temp) != 3 {
		return nil, common.NewBadRequestError("invalid upload id")
	}
	dir := c.getDir(id)
	exists, e := common.FileExists(dir)
	if e != nil {
		return nil, e
	}
	if !exists {
		return nil, common.NewNotFoundError()
	}
	var size, chunkSize int64
	var e1, e2 error
	size, e1 = strconv.ParseInt(temp[1], 10, 64)
	chunkSize, e2 = strconv.ParseInt(temp[2], 10, 64)
	if e1 != nil || e2 != nil {
		return nil, common.NewBadRequestError("invalid upload id")
	}
	return newChunkUpload(id, size, chunkSize), nil
}

func (c *ChunkUploader) isMarkedDelete(upload *ChunkUpload) bool {
	deleteFile := c.getDeleteMark(upload)
	exists, _ := common.FileExists(deleteFile)
	return exists
}

func (c *ChunkUploader) markDeleted(upload *ChunkUpload) error {
	deleteFile := c.getDeleteMark(upload)
	exists, e := common.FileExists(deleteFile)
	if e != nil {
		return e
	}
	if !exists {
		if e := ioutil.WriteFile(deleteFile, []byte(""), 0644); e != nil {
			return e
		}
	}
	return nil
}

func (c *ChunkUploader) getFile(upload *ChunkUpload) string {
	return fsPath.Join(c.getDir(upload.Id), "file")
}

func (c *ChunkUploader) getChunk(upload *ChunkUpload, seq int) string {
	return fsPath.Join(c.getDir(upload.Id), strconv.Itoa(seq))
}

func (c *ChunkUploader) getDeleteMark(upload *ChunkUpload) string {
	return fsPath.Join(c.getDir(upload.Id), "deleted")
}

func (c *ChunkUploader) getDir(id string) string {
	return fsPath.Join(c.dir, id)
}

type ChunkUpload struct {
	Id        string `json:"id"`
	Size      int64  `json:"size"`
	ChunkSize int64  `json:"chunk_size"`
	Chunks    int    `json:"chunks"`
}

func newChunkUpload(id string, size, chunkSize int64) *ChunkUpload {
	return &ChunkUpload{
		Id:        id,
		Size:      size,
		ChunkSize: chunkSize,
		Chunks:    int(math.Ceil(float64(size) / float64(chunkSize))),
	}
}
