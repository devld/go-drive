package drive

import (
	"context"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	path2 "path"
	"strings"
)

type Chroot struct {
	d    types.IDrive
	root string
}

func NewChroot(d types.IDrive, root string) *Chroot {
	return &Chroot{d: d, root: root}
}

func (c *Chroot) getPath(path string) string {
	return path2.Join(c.root, path)
}

func (c *Chroot) Meta(ctx context.Context) types.DriveMeta {
	return c.d.Meta(ctx)
}

func (c *Chroot) Get(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := c.d.Get(ctx, c.getPath(path))
	return c.wrapEntry(entry), e
}

func (c *Chroot) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	entry, e := c.d.Save(ctx, c.getPath(path), size, override, reader)
	return c.wrapEntry(entry), e
}

func (c *Chroot) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := c.d.MakeDir(ctx, c.getPath(path))
	return c.wrapEntry(entry), e
}

func (c *Chroot) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := c.d.Copy(ctx, from, c.getPath(to), override)
	return c.wrapEntry(entry), e
}

func (c *Chroot) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := c.d.Move(ctx, from, c.getPath(to), override)
	return c.wrapEntry(entry), e
}

func (c *Chroot) List(ctx context.Context, path string) ([]types.IEntry, error) {
	entries, e := c.d.List(ctx, c.getPath(path))
	return c.wrapEntries(entries), e
}

func (c *Chroot) Delete(ctx types.TaskCtx, path string) error {
	return c.d.Delete(ctx, c.getPath(path))
}

func (c *Chroot) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	return c.d.Upload(ctx, c.getPath(path), size, override, config)
}

func (c *Chroot) wrapEntry(e types.IEntry) types.IEntry {
	if e == nil {
		return nil
	}
	path := e.Path()
	if !strings.HasPrefix(path, c.root) {
		panic("unexpected path: " + path + ", but the chroot is: " + c.root)
	}
	path = utils.CleanPath(path[len(c.root):])
	return &chrootEntry{
		entry: e,
		path:  path,
		c:     c,
	}
}

func (c *Chroot) wrapEntries(es []types.IEntry) []types.IEntry {
	if es == nil {
		return nil
	}
	result := make([]types.IEntry, 0, len(es))
	for _, e := range es {
		result = append(result, c.wrapEntry(e))
	}
	return result
}

type chrootEntry struct {
	entry types.IEntry
	path  string
	c     *Chroot
}

func (c *chrootEntry) Path() string {
	return c.path
}

func (c *chrootEntry) Drive() types.IDrive {
	return c.c
}

func (c *chrootEntry) GetIEntry() types.IEntry {
	return c.entry
}

func (c *chrootEntry) Type() types.EntryType {
	return c.entry.Type()
}

func (c *chrootEntry) Size() int64 {
	return c.entry.Size()
}

func (c *chrootEntry) Meta() types.EntryMeta {
	return c.entry.Meta()
}

func (c *chrootEntry) ModTime() int64 {
	return c.entry.ModTime()
}

func (c *chrootEntry) Name() string {
	return utils.PathBase(c.Path())
}

func (c *chrootEntry) GetReader(ctx context.Context) (io.ReadCloser, error) {
	if content, ok := c.entry.(types.IContent); ok {
		return content.GetReader(ctx)
	}
	return nil, err.NewNotAllowedError()
}

func (c *chrootEntry) GetURL(ctx context.Context) (*types.ContentURL, error) {
	if content, ok := c.entry.(types.IContent); ok {
		return content.GetURL(ctx)
	}
	return nil, err.NewNotAllowedError()
}
