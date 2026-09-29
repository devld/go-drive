package server

import (
	"context"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const minChunkSize = 5 * 1024 * 1024

const (
	chunkUploadCleanupInterval = 3 * time.Hour
	chunkUploadMaxIdle         = 24 * time.Hour
)

// The bitmap starts with a flags byte, followed by one bit per chunk. The data
// file is separate so its exact bytes can be passed to the drive without
// including upload state.
const (
	chunkBitmapHeaderSize   = int64(1)
	chunkBitmapCompleteFlag = byte(1 << 0)
)

type ChunkUploader struct {
	dir string

	statesMu  sync.Mutex
	states    map[string]*chunkUploadState
	timerStop func()
}

type chunkUploadState struct {
	mu            sync.Mutex
	uploading     map[int]struct{}
	deleting      bool
	refs          int  // protected by statesMu
	pendingDelete bool // protected by statesMu
}

func NewChunkUploader(config common.Config) (*ChunkUploader, error) {
	dir, e := config.GetTempDir("upload", true)
	if e != nil {
		return nil, e
	}
	uploader := &ChunkUploader{dir: dir}
	uploader.timerStop = utils.TimeTick(func() {
		uploader.cleanupExpiredUploads(time.Now())
	}, chunkUploadCleanupInterval)
	return uploader, nil
}

func (c *ChunkUploader) Dispose() error {
	c.timerStop()
	return nil
}

func (c *ChunkUploader) cleanupExpiredUploads(now time.Time) {
	entries, e := os.ReadDir(c.dir)
	if e != nil {
		logging.For("chunk-uploader").Warnf("read upload directory failed: %v", e)
		return
	}
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		upload, e := parseUploadID(entry.Name())
		if e != nil {
			continue
		}
		deleted, e := c.deleteUploadIfExpired(upload, now)
		if e != nil {
			logging.For("chunk-uploader").Warnf("remove expired upload failed id=%s: %v", upload.ID, e)
		} else if deleted {
			removed++
		}
	}
	if removed > 0 {
		logging.For("chunk-uploader").Infof("removed expired uploads count=%d", removed)
	}
}

func (c *ChunkUploader) CreateUpload(size, chunkSize int64) (ChunkUpload, error) {
	if chunkSize < minChunkSize {
		return ChunkUpload{}, err.NewBadRequestError(fmt.Sprintf("chunk size cannot be less than %d", minChunkSize))
	}
	if size <= 0 {
		return ChunkUpload{}, err.NewBadRequestError("invalid file size")
	}
	chunks, e := calculateChunkCount(size, chunkSize)
	if e != nil {
		return ChunkUpload{}, err.NewBadRequestError(e.Error())
	}
	id := generateUploadID(size, chunkSize)
	// Keep cleanup from treating the directory as orphaned before the bitmap exists.
	state, release := c.acquireState(id)
	defer release()
	state.mu.Lock()
	defer state.mu.Unlock()
	dir := c.getDir(id)
	if e := os.Mkdir(dir, 0700); e != nil {
		return ChunkUpload{}, e
	}
	upload := ChunkUpload{ID: id, Size: size, ChunkSize: chunkSize, Chunks: chunks}
	if e := c.initializeUpload(upload); e != nil {
		_ = os.RemoveAll(dir)
		return ChunkUpload{}, e
	}
	return upload, nil
}

func (c *ChunkUploader) initializeUpload(upload ChunkUpload) error {
	data, e := os.OpenFile(c.getFile(upload), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	if e := data.Close(); e != nil {
		return e
	}

	bitmap, e := os.OpenFile(c.getBitmap(upload), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	if e := bitmap.Truncate(chunkBitmapFileSize(upload)); e != nil {
		_ = bitmap.Close()
		return e
	}
	return bitmap.Close()
}

func (c *ChunkUploader) ChunkUpload(ctx context.Context, id string, seq int, reader io.ReadCloser) error {
	upload, e := c.getUpload(id)
	if e != nil {
		return e
	}
	if seq < 0 || seq >= upload.Chunks {
		return err.NewNotAllowedMessageError("invalid chunk seq")
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	return c.uploadChunkAt(ctx, upload, seq, reader)
}

func (c *ChunkUploader) uploadChunkAt(ctx context.Context, upload ChunkUpload, seq int, reader io.ReadCloser) error {
	state, release := c.acquireState(upload.ID)
	defer release()
	state.mu.Lock()
	if state.deleting {
		state.mu.Unlock()
		return context.Canceled
	}
	if _, uploading := state.uploading[seq]; uploading {
		state.mu.Unlock()
		_ = reader.Close()
		return err.NewConflictError("chunk is already being uploaded")
	}
	state.uploading[seq] = struct{}{}
	state.mu.Unlock()
	defer func() {
		state.mu.Lock()
		delete(state.uploading, seq)
		state.mu.Unlock()
	}()
	if e := ctx.Err(); e != nil {
		return e
	}
	stopClose := context.AfterFunc(ctx, func() { _ = reader.Close() })
	defer stopClose()

	data, e := os.OpenFile(c.getFile(upload), os.O_RDWR, 0)
	if e != nil {
		return e
	}
	defer func() { _ = data.Close() }()
	bitmap, e := c.openBitmap(upload)
	if e != nil {
		return e
	}
	defer func() { _ = bitmap.Close() }()

	state.mu.Lock()
	if state.deleting {
		state.mu.Unlock()
		return context.Canceled
	}
	flags, e := readChunkBitmapFlags(bitmap)
	if e == nil && flags&chunkBitmapCompleteFlag != 0 {
		e = err.NewNotAllowedMessageError("upload already completed")
	}
	if e == nil {
		e = writeChunkBitmapBit(bitmap, seq, false)
	}
	state.mu.Unlock()
	if e != nil {
		return e
	}

	chunkSize := upload.chunkSize(seq)
	written, e := io.CopyN(io.NewOffsetWriter(data, int64(seq)*upload.ChunkSize), reader, chunkSize)
	if e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e != io.EOF && e != io.ErrUnexpectedEOF {
			return e
		}
		return expectedChunkSizeError(chunkSize, written)
	}
	var extra [1]byte
	extraRead, extraErr := io.ReadFull(reader, extra[:])
	if extraRead != 0 {
		return expectedChunkSizeError(chunkSize, written+int64(extraRead))
	}
	if extraErr != io.EOF {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return extraErr
	}
	if e := data.Close(); e != nil {
		return e
	}
	if e := ctx.Err(); e != nil {
		return e
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.deleting {
		return context.Canceled
	}
	return writeChunkBitmapBit(bitmap, seq, true)
}

func (c *ChunkUploader) CompleteUpload(id string, ctx types.TaskCtx) (*os.File, error) {
	upload, e := c.getUpload(id)
	if e != nil {
		return nil, e
	}
	return c.completePositionedUpload(upload, ctx)
}

func (c *ChunkUploader) completePositionedUpload(upload ChunkUpload, ctx types.TaskCtx) (*os.File, error) {
	state, release := c.acquireState(upload.ID)
	defer release()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.deleting {
		return nil, context.Canceled
	}
	bitmap, e := c.openBitmap(upload)
	if e != nil {
		return nil, e
	}
	defer func() { _ = bitmap.Close() }()
	for seq := 0; seq < upload.Chunks; seq++ {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		complete, e := readChunkBitmapBit(bitmap, seq)
		if e != nil {
			return nil, e
		}
		if !complete {
			return nil, err.NewNotAllowedMessageError("missing chunks")
		}
	}
	data, e := os.Open(c.getFile(upload))
	if e != nil {
		return nil, e
	}
	stat, e := data.Stat()
	if e != nil {
		_ = data.Close()
		return nil, e
	}
	if stat.Size() != upload.Size {
		_ = data.Close()
		return nil, fmt.Errorf("uploaded file size is %d, expected %d", stat.Size(), upload.Size)
	}
	flags, e := readChunkBitmapFlags(bitmap)
	if e != nil {
		_ = data.Close()
		return nil, e
	}
	if flags&chunkBitmapCompleteFlag == 0 {
		_, e := bitmap.WriteAt([]byte{flags | chunkBitmapCompleteFlag}, 0)
		if e != nil {
			_ = data.Close()
			return nil, e
		}
	}
	ctx.Total(upload.Size, true)
	ctx.Progress(upload.Size, false)
	return data, nil
}

func (c *ChunkUploader) DeleteUpload(id string) error {
	upload, e := c.getUpload(id)
	if e != nil {
		return e
	}
	state, release := c.acquireState(upload.ID)
	defer release()
	state.mu.Lock()
	defer state.mu.Unlock()
	state.deleting = true
	_, e = c.removeUploadDirectory(upload, state)
	return e
}

func (c *ChunkUploader) deleteUploadIfExpired(upload ChunkUpload, now time.Time) (bool, error) {
	state, release := c.acquireState(upload.ID)
	defer release()
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.deleting {
		info, e := os.Stat(c.getBitmap(upload))
		if e != nil && !os.IsNotExist(e) {
			return false, e
		}
		if e == nil && now.Sub(info.ModTime()) <= chunkUploadMaxIdle {
			return false, nil
		}
		state.deleting = true
	}
	return c.removeUploadDirectory(upload, state)
}

// removeUploadDirectory is called with state.mu held. A failed removal keeps
// the in-memory deleting state so the next scan can retry it.
func (c *ChunkUploader) removeUploadDirectory(upload ChunkUpload, state *chunkUploadState) (bool, error) {
	c.statesMu.Lock()
	state.pendingDelete = true
	c.statesMu.Unlock()
	if e := os.RemoveAll(c.getDir(upload.ID)); e != nil {
		return false, e
	}
	c.statesMu.Lock()
	state.pendingDelete = false
	c.statesMu.Unlock()
	return true, nil
}

func (c *ChunkUploader) getUpload(id string) (ChunkUpload, error) {
	upload, e := parseUploadID(id)
	if e != nil {
		return ChunkUpload{}, e
	}
	dir := c.getDir(id)
	exists, e := utils.FileExists(dir)
	if e != nil {
		return ChunkUpload{}, e
	}
	if !exists {
		return ChunkUpload{}, err.NewNotFoundError()
	}
	return upload, nil
}

func (c *ChunkUploader) getFile(upload ChunkUpload) string {
	return filepath.Join(c.getDir(upload.ID), "file")
}

func (c *ChunkUploader) getBitmap(upload ChunkUpload) string {
	return filepath.Join(c.getDir(upload.ID), "bitmap")
}

func (c *ChunkUploader) getDir(id string) string {
	return filepath.Join(c.dir, id)
}

func (c *ChunkUploader) openBitmap(upload ChunkUpload) (*os.File, error) {
	bitmap, e := os.OpenFile(c.getBitmap(upload), os.O_RDWR, 0)
	if e != nil {
		return nil, e
	}
	stat, e := bitmap.Stat()
	if e != nil {
		_ = bitmap.Close()
		return nil, e
	}
	if !stat.Mode().IsRegular() || stat.Size() != chunkBitmapFileSize(upload) {
		_ = bitmap.Close()
		return nil, fmt.Errorf("invalid chunk upload bitmap")
	}
	return bitmap, nil
}

func (c *ChunkUploader) acquireState(id string) (*chunkUploadState, func()) {
	c.statesMu.Lock()
	if c.states == nil {
		c.states = make(map[string]*chunkUploadState)
	}
	state := c.states[id]
	if state == nil {
		state = &chunkUploadState{uploading: make(map[int]struct{})}
		c.states[id] = state
	}
	state.refs++
	c.statesMu.Unlock()
	return state, func() { c.releaseState(id, state) }
}

func (c *ChunkUploader) releaseState(id string, state *chunkUploadState) {
	c.statesMu.Lock()
	defer c.statesMu.Unlock()
	state.refs--
	if state.refs == 0 && !state.pendingDelete {
		delete(c.states, id)
	}
}

func readChunkBitmapFlags(bitmap *os.File) (byte, error) {
	var flags [chunkBitmapHeaderSize]byte
	if _, e := bitmap.ReadAt(flags[:], 0); e != nil {
		return 0, e
	}
	if flags[0]&^chunkBitmapCompleteFlag != 0 {
		return 0, fmt.Errorf("invalid chunk upload bitmap flags")
	}
	return flags[0], nil
}

func readChunkBitmapBit(bitmap *os.File, seq int) (bool, error) {
	var value [1]byte
	offset := chunkBitmapHeaderSize + int64(seq/8)
	if _, e := bitmap.ReadAt(value[:], offset); e != nil {
		return false, e
	}
	return value[0]&(1<<uint(seq%8)) != 0, nil
}

func writeChunkBitmapBit(bitmap *os.File, seq int, complete bool) error {
	var value [1]byte
	offset := chunkBitmapHeaderSize + int64(seq/8)
	if _, e := bitmap.ReadAt(value[:], offset); e != nil {
		return e
	}
	mask := byte(1 << uint(seq%8))
	if complete {
		value[0] |= mask
	} else {
		value[0] &^= mask
	}
	_, e := bitmap.WriteAt(value[:], offset)
	return e
}

func chunkBitmapFileSize(upload ChunkUpload) int64 {
	bitBytes := upload.Chunks / 8
	if upload.Chunks%8 != 0 {
		bitBytes++
	}
	return chunkBitmapHeaderSize + int64(bitBytes)
}

func calculateChunkCount(size, chunkSize int64) (int, error) {
	if size <= 0 || chunkSize <= 0 {
		return 0, fmt.Errorf("invalid upload size or chunk size")
	}
	count := (size-1)/chunkSize + 1
	maxInt := int64(^uint(0) >> 1)
	if count > maxInt {
		return 0, fmt.Errorf("too many upload chunks")
	}
	return int(count), nil
}

func expectedChunkSizeError(expected, written int64) error {
	return err.NewBadRequestError(fmt.Sprintf("expected %d bytes, received %d", expected, written))
}

func generateUploadID(size, chunkSize int64) string {
	return fmt.Sprintf("%s_%d_%d", uuid.New().String(), size, chunkSize)
}

func parseUploadID(id string) (ChunkUpload, error) {
	parts := strings.Split(id, "_")
	if len(parts) != 3 || !isValidUploadIDPart(parts[0]) {
		return ChunkUpload{}, err.NewBadRequestError("invalid upload id")
	}
	size := utils.ToInt64(parts[1], -1)
	chunkSize := utils.ToInt64(parts[2], -1)
	if size <= 0 || chunkSize < minChunkSize {
		return ChunkUpload{}, err.NewBadRequestError("invalid upload id")
	}
	chunks, e := calculateChunkCount(size, chunkSize)
	if e != nil {
		return ChunkUpload{}, err.NewBadRequestError(e.Error())
	}
	return ChunkUpload{ID: id, Size: size, ChunkSize: chunkSize, Chunks: chunks}, nil
}

// isValidUploadIDPart accepts only the format produced by uuid.New().String().
func isValidUploadIDPart(s string) bool {
	id, e := uuid.Parse(s)
	return e == nil && id.String() == s
}

type ChunkUpload struct {
	ID        string `json:"id"`
	Size      int64  `json:"size"`
	ChunkSize int64  `json:"chunkSize"`
	Chunks    int    `json:"chunks"`
}

func (u ChunkUpload) chunkSize(seq int) int64 {
	if seq == u.Chunks-1 {
		if rem := u.Size % u.ChunkSize; rem != 0 {
			return rem
		}
	}
	return u.ChunkSize
}
