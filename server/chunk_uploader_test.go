package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	apierr "go-drive/common/errors"
	"go-drive/common/task"
)

func Test_isValidUploadIdPart(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"f47ac10b-58cc-0372-8567-0e02b2c3d479", true},
		{"F47AC10B-58CC-0372-8567-0E02B2C3D479", false},
		{"f47ac10b58cc037285670e02b2c3d479", false},
		{"ABCDEF0123456789", false},
		{"", false},
		{"../secret", false},
		{"..", false},
		{"a/b", false},
		{"a\\b", false},
		{"name_with_underscore", false},
		{"g123", false}, // 'g' is not a hex char
		{"with space", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := isValidUploadIDPart(tt.in); got != tt.want {
				t.Errorf("isValidUploadIdPart(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestChunkUploader_getUpload_RejectsTraversalId(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	for _, id := range []string{"../secret_1_1", "..%2f_1_1", "a/b_1_1", "..\\x_1_1"} {
		if _, e := c.getUpload(id); e == nil {
			t.Errorf("expected error for malicious id %q", id)
		}
	}
}

// TestChunkUploader_DeleteUpload_DoesNotEscapeDir ensures a crafted upload id
// cannot cause DeleteUpload to remove a directory outside the upload root.
func TestChunkUploader_DeleteUpload_DoesNotEscapeDir(t *testing.T) {
	base := t.TempDir()
	uploadDir := filepath.Join(base, "upload")
	if e := os.Mkdir(uploadDir, 0755); e != nil {
		t.Fatal(e)
	}

	// a sibling directory the crafted id "../secret_1_1" would resolve to
	target := filepath.Join(base, "secret_1_1")
	if e := os.Mkdir(target, 0755); e != nil {
		t.Fatal(e)
	}
	sentinel := filepath.Join(target, "keep.txt")
	if e := os.WriteFile(sentinel, []byte("x"), 0644); e != nil {
		t.Fatal(e)
	}

	c := &ChunkUploader{dir: uploadDir}
	_ = c.DeleteUpload("../secret_1_1")

	if _, e := os.Stat(sentinel); e != nil {
		t.Fatalf("sentinel file should be preserved, but got: %v", e)
	}
}

func TestChunkUploader_ChunkUploadInvalidIDReturnsError(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	if e := c.ChunkUpload(context.Background(), "invalid", 0, io.NopCloser(bytes.NewReader(nil))); e == nil {
		t.Fatal("ChunkUpload() expected invalid ID error")
	}
}

func TestChunkUploader_PositionedUploadTracksConcurrentChunks(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	const chunkSize = int64(minChunkSize)
	upload, e := c.CreateUpload(3*chunkSize+7, chunkSize)
	if e != nil {
		t.Fatal(e)
	}

	chunks := make([][]byte, upload.Chunks)
	for seq := range chunks {
		chunks[seq] = bytes.Repeat([]byte{byte(seq + 1)}, int(upload.chunkSize(seq)))
	}

	var wg sync.WaitGroup
	errCh := make(chan error, upload.Chunks)
	for seq, chunk := range chunks {
		wg.Add(1)
		go func(seq int, chunk []byte) {
			defer wg.Done()
			errCh <- c.ChunkUpload(context.Background(), upload.ID, seq, io.NopCloser(bytes.NewReader(chunk)))
		}(seq, chunk)
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Fatal(e)
		}
	}

	bitmap, e := os.Open(c.getBitmap(upload))
	if e != nil {
		t.Fatal(e)
	}
	for seq := range chunks {
		complete, e := readChunkBitmapBit(bitmap, seq)
		if e != nil {
			_ = bitmap.Close()
			t.Fatal(e)
		}
		if !complete {
			_ = bitmap.Close()
			t.Fatalf("bitmap bit %d is not set", seq)
		}
	}
	if e := bitmap.Close(); e != nil {
		t.Fatal(e)
	}

	file, e := c.CompleteUpload(upload.ID, task.DummyContext())
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(file)
	if closeErr := file.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if e != nil {
		t.Fatal(e)
	}
	want := bytes.Join(chunks, nil)
	if !bytes.Equal(got, want) {
		t.Fatal("completed upload content does not match uploaded chunks")
	}
	if int64(len(got)) != upload.Size {
		t.Fatalf("completed file size = %d, want %d", len(got), upload.Size)
	}

	bitmapData, e := os.ReadFile(c.getBitmap(upload))
	if e != nil {
		t.Fatal(e)
	}
	if bitmapData[0]&chunkBitmapCompleteFlag == 0 {
		t.Fatal("completion flag was not recorded")
	}
	if e := c.ChunkUpload(context.Background(), upload.ID, 0, io.NopCloser(bytes.NewReader(chunks[0]))); e == nil {
		t.Fatal("ChunkUpload() after completion should fail")
	}
	if e := c.DeleteUpload(upload.ID); e != nil {
		t.Fatal(e)
	}
}

func TestChunkUploader_ShortChunkDoesNotSetBitmap(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	upload, e := c.CreateUpload(2*int64(minChunkSize), int64(minChunkSize))
	if e != nil {
		t.Fatal(e)
	}

	short := bytes.Repeat([]byte("x"), minChunkSize-1)
	if e := c.ChunkUpload(context.Background(), upload.ID, 0, io.NopCloser(bytes.NewReader(short))); e == nil {
		t.Fatal("short chunk upload should fail")
	}
	bitmap, e := c.openBitmap(upload)
	if e != nil {
		t.Fatal(e)
	}
	complete, e := readChunkBitmapBit(bitmap, 0)
	if closeErr := bitmap.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if e != nil {
		t.Fatal(e)
	}
	if complete {
		t.Fatal("failed chunk upload was marked complete")
	}
	if _, e := c.CompleteUpload(upload.ID, task.DummyContext()); e == nil {
		t.Fatal("completion with a missing chunk should fail")
	}
}

func TestChunkUploader_RejectsConcurrentSameChunk(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	const chunkSize = int64(minChunkSize)
	upload, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}

	chunk := bytes.Repeat([]byte("x"), minChunkSize)
	reader := &blockingChunkReader{
		reader:  bytes.NewReader(chunk),
		started: make(chan struct{}),
		unblock: make(chan struct{}),
	}
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- c.ChunkUpload(context.Background(), upload.ID, 0, reader)
	}()
	<-reader.started

	e = c.ChunkUpload(context.Background(), upload.ID, 0, io.NopCloser(bytes.NewReader(chunk)))
	notAllowed, ok := e.(apierr.NotAllowedError)
	if !ok || notAllowed.Code() != http.StatusForbidden {
		t.Fatalf("concurrent upload error = %v, want NotAllowedError with HTTP %d", e, http.StatusForbidden)
	}

	_ = reader.Close()
	if e := <-firstDone; e != nil {
		t.Fatal(e)
	}
	if e := c.ChunkUpload(context.Background(), upload.ID, 0, io.NopCloser(bytes.NewReader(chunk))); e != nil {
		t.Fatalf("chunk retry after first upload finished: %v", e)
	}
}

func TestChunkUploader_CancelContextInterruptsRead(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	const chunkSize = int64(minChunkSize)
	upload, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}

	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := io.Pipe()
	defer func() {
		cancel()
		_ = writer.Close()
	}()
	trackedReader := &notifiedPipeReader{PipeReader: reader, started: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- c.ChunkUpload(ctx, upload.ID, 0, trackedReader)
	}()
	<-trackedReader.started
	cancel()

	select {
	case e := <-done:
		if e != context.Canceled {
			t.Fatalf("ChunkUpload() error = %v, want context.Canceled", e)
		}
	case <-time.After(time.Second):
		t.Fatal("ChunkUpload() did not return after context cancellation")
	}

	bitmap, e := c.openBitmap(upload)
	if e != nil {
		t.Fatal(e)
	}
	complete, e := readChunkBitmapBit(bitmap, 0)
	if closeErr := bitmap.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if e != nil {
		t.Fatal(e)
	}
	if complete {
		t.Fatal("canceled chunk upload was marked complete")
	}
}

func TestChunkUploader_CleanupExpiredUploads(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	const chunkSize = int64(minChunkSize)
	expired, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}
	recent, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}
	orphan, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}
	pending, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}

	old := time.Now().Add(-chunkUploadMaxIdle - time.Hour)
	if e := os.Chtimes(c.getBitmap(expired), old, old); e != nil {
		t.Fatal(e)
	}
	if e := os.Remove(c.getBitmap(orphan)); e != nil {
		t.Fatal(e)
	}
	state, release := c.acquireState(pending.ID)
	state.mu.Lock()
	state.deleting = true
	state.mu.Unlock()
	c.statesMu.Lock()
	state.pendingDelete = true
	c.statesMu.Unlock()
	release()

	c.cleanupExpiredUploads(time.Now())
	if _, e := os.Stat(c.getDir(expired.ID)); !os.IsNotExist(e) {
		t.Fatalf("expired upload directory still exists, stat error = %v", e)
	}
	if _, e := os.Stat(c.getDir(recent.ID)); e != nil {
		t.Fatalf("recent upload directory was removed: %v", e)
	}
	if _, e := os.Stat(c.getDir(orphan.ID)); !os.IsNotExist(e) {
		t.Fatalf("orphan upload directory still exists, stat error = %v", e)
	}
	if _, e := os.Stat(c.getDir(pending.ID)); !os.IsNotExist(e) {
		t.Fatalf("pending deletion upload directory still exists, stat error = %v", e)
	}
}

func TestChunkUploader_CleanupExpiredUploadWithActiveChunk(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	const chunkSize = int64(minChunkSize)
	upload, e := c.CreateUpload(chunkSize, chunkSize)
	if e != nil {
		t.Fatal(e)
	}
	reader := &blockingChunkReader{
		reader:  bytes.NewReader(bytes.Repeat([]byte("x"), minChunkSize)),
		started: make(chan struct{}),
		unblock: make(chan struct{}),
	}
	done := make(chan error, 1)
	go func() {
		done <- c.ChunkUpload(context.Background(), upload.ID, 0, reader)
	}()
	<-reader.started

	old := time.Now().Add(-chunkUploadMaxIdle - time.Hour)
	if e := os.Chtimes(c.getBitmap(upload), old, old); e != nil {
		t.Fatal(e)
	}

	c.cleanupExpiredUploads(time.Now())
	_ = reader.Close()
	select {
	case e := <-done:
		if e != context.Canceled {
			t.Fatalf("ChunkUpload() error = %v, want context.Canceled", e)
		}
	case <-time.After(time.Second):
		t.Fatal("expired active chunk upload did not finish after reader closed")
	}
	if _, e := os.Stat(c.getDir(upload.ID)); !os.IsNotExist(e) {
		t.Fatalf("expired upload directory still exists, stat error = %v", e)
	}
}

type blockingChunkReader struct {
	reader    *bytes.Reader
	started   chan struct{}
	unblock   chan struct{}
	once      sync.Once
	closeOnce sync.Once
}

func (r *blockingChunkReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.unblock
	return r.reader.Read(p)
}

func (r *blockingChunkReader) Close() error {
	r.closeOnce.Do(func() { close(r.unblock) })
	return nil
}

type notifiedPipeReader struct {
	*io.PipeReader
	started chan struct{}
	once    sync.Once
}

func (r *notifiedPipeReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.PipeReader.Read(p)
}
