package server

import (
	"context"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"math"
	"os"
	path2 "path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const minChunkSize = 5 * 1024 * 1024

type ChunkUploader struct {
	dir string
}

func NewChunkUploader(config common.Config) (*ChunkUploader, error) {
	dir, e := config.GetTempDir("upload", true)
	if e != nil {
		return nil, e
	}
	return &ChunkUploader{dir}, nil
}

func (c *ChunkUploader) CreateUpload(size, chunkSize int64) (ChunkUpload, error) {
	if chunkSize < minChunkSize {
		return ChunkUpload{},
			err.NewBadRequestError(i18n.T("api.chunk_uploader.chunk_size_cannot_less_than", strconv.Itoa(minChunkSize)))
	}
	if size <= 0 {
		return ChunkUpload{}, err.NewBadRequestError(i18n.T("api.chunk_uploader.invalid_file_size"))
	}
	id := c.generateUploadId(size, chunkSize)
	dir := c.getDir(id)
	exists, e := utils.FileExists(dir)
	if e != nil {
		return ChunkUpload{}, e
	}
	if exists {
		return ChunkUpload{}, err.NewNotAllowedError()
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
		return err.NewNotAllowedMessageError(i18n.T("api.chunk_uploader.invalid_chunk_seq"))
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
		return err.NewBadRequestError(i18n.T("api.chunk_uploader.expected__bytes_but__bytes",
			strconv.FormatInt(chunkSize, 10), strconv.FormatInt(written, 10)))
	}
	success = true
	return nil
}

func (c *ChunkUploader) CompleteUpload(id string, ctx types.TaskCtx) (*os.File, error) {
	upload, e := c.getUpload(id)
	if e != nil {
		return nil, e
	}
	for seq := 0; seq < upload.Chunks; seq++ {
		exists, e := utils.FileExists(c.getChunk(upload, seq))
		if e != nil {
			return nil, e
		}
		if !exists {
			return nil, err.NewNotAllowedMessageError(i18n.T("missing_chunks"))
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
	ctx.Total(upload.Size, true)
	for seq := 0; seq < upload.Chunks; seq++ {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		chunk, e := os.Open(c.getChunk(upload, seq))
		if e != nil {
			return nil, e
		}
		w, e := io.Copy(file, chunk)
		_ = chunk.Close()
		if c.isMarkedDelete(upload) {
			return nil, context.Canceled
		}
		if e != nil {
			return nil, e
		}
		ctx.Progress(w, false)
	}
	allSuccess = true
	_ = file.Close()
	if !allSuccess {
		return nil, context.Canceled
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
		return nil, err.NewBadRequestError(i18n.T("api.chunk_uploader.invalid_upload_id"))
	}
	dir := c.getDir(id)
	exists, e := utils.FileExists(dir)
	if e != nil {
		return nil, e
	}
	if !exists {
		return nil, err.NewNotFoundError()
	}
	size := utils.ToInt64(temp[1], -1)
	chunkSize := utils.ToInt64(temp[2], -1)
	if size <= 0 || chunkSize <= 0 {
		return nil, err.NewBadRequestError(i18n.T("api.chunk_uploader.invalid_upload_id"))
	}
	return newChunkUpload(id, size, chunkSize), nil
}

func (c *ChunkUploader) isMarkedDelete(upload *ChunkUpload) bool {
	deleteFile := c.getDeleteMark(upload)
	exists, _ := utils.FileExists(deleteFile)
	return exists
}

func (c *ChunkUploader) markDeleted(upload *ChunkUpload) error {
	deleteFile := c.getDeleteMark(upload)
	exists, e := utils.FileExists(deleteFile)
	if e != nil {
		return e
	}
	if !exists {
		if e := os.WriteFile(deleteFile, []byte(""), 0644); e != nil {
			return e
		}
	}
	return nil
}

func (c *ChunkUploader) getFile(upload *ChunkUpload) string {
	return path2.Join(c.getDir(upload.Id), "file")
}

func (c *ChunkUploader) getChunk(upload *ChunkUpload, seq int) string {
	return path2.Join(c.getDir(upload.Id), strconv.Itoa(seq))
}

func (c *ChunkUploader) getDeleteMark(upload *ChunkUpload) string {
	return path2.Join(c.getDir(upload.Id), "deleted")
}

func (c *ChunkUploader) getDir(id string) string {
	return path2.Join(c.dir, filepath.Clean(id))
}

type ChunkUpload struct {
	Id        string `json:"id"`
	Size      int64  `json:"size"`
	ChunkSize int64  `json:"chunkSize"`
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
