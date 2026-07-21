package driveutil

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	err "go-drive/common/errors"
	"go-drive/common/types"
)

type cacheTestEntry struct {
	path string
}

func (e cacheTestEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, err.NewUnsupportedError()
}

func (e cacheTestEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}

func (e cacheTestEntry) Name() string          { return e.path }
func (e cacheTestEntry) Size() int64           { return 0 }
func (e cacheTestEntry) ModTime() int64        { return 0 }
func (e cacheTestEntry) Path() string          { return e.path }
func (e cacheTestEntry) Type() types.EntryType { return types.TypeFile }
func (e cacheTestEntry) Meta() types.EntryMeta { return types.EntryMeta{} }
func (e cacheTestEntry) Drive() types.IDrive   { return nil }

func TestMemDriveCacheConcurrentChildrenAccess(t *testing.T) {
	mgr := NewMemDriveCacheManager(0)
	t.Cleanup(func() { _ = mgr.Dispose() })
	cache := mgr.GetCacheStore("test", nil)

	var wg sync.WaitGroup
	for worker := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				entries := []types.IEntry{
					cacheTestEntry{path: fmt.Sprintf("dir/%d-%d", worker, i)},
				}
				if e := cache.PutChildren("dir", entries, time.Minute); e != nil {
					t.Error(e)
				}
				if _, e := cache.GetChildrenRaw("dir"); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
}
