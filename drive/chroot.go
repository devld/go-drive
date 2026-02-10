package drive

import (
	"context"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	path2 "path"
	"strings"
)

type Chroot struct {
	Root       string
	NameFilter map[string]struct{}
}

// NewChroot creates Chroot
//
// root is the jail root path
//
// entries are the visible entry names inside root
func NewChroot(root string, entries []string) *Chroot {
	var em map[string]struct{} = nil
	if len(entries) > 0 {
		em = make(map[string]struct{})
		for _, e := range entries {
			em[e] = struct{}{}
		}
	}
	return &Chroot{Root: root, NameFilter: em}
}

// WrapPath add root prefix to path
func (c *Chroot) WrapPath(path string) (string, error) {
	if c.NameFilter != nil && !utils.IsRootPath(path) {
		firstNode := path
		index := strings.Index(firstNode, "/")
		if index >= 0 {
			firstNode = firstNode[:index]
		}
		if _, ok := c.NameFilter[firstNode]; !ok {
			return "", err.NewNotFoundError()
		}
	}
	return path2.Join(c.Root, path), nil
}

// UnwrapPath remove root prefix from path
func (c *Chroot) UnwrapPath(path string) string {
	if !strings.HasPrefix(path, c.Root) {
		return path
	}
	return utils.CleanPath(path[len(c.Root):])
}

func (c *Chroot) FilterEntries(unwrappedPath string, entries []types.IEntry) []types.IEntry {
	if c.NameFilter == nil || len(entries) == 0 || !utils.IsRootPath(unwrappedPath) {
		return entries
	}
	filtered := make([]types.IEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := c.NameFilter[utils.PathBase(c.UnwrapPath(entry.Path()))]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

var _ types.IDrive = (*ChrootWrapper)(nil)

type ChrootWrapper struct {
	d types.IDrive
	*Chroot
}

func NewChrootWrapper(d types.IDrive, chroot *Chroot) *ChrootWrapper {
	return &ChrootWrapper{d: d, Chroot: chroot}
}

func (c *ChrootWrapper) Meta(ctx context.Context) (types.DriveMeta, error) {
	return c.d.Meta(ctx)
}

func (c *ChrootWrapper) Get(ctx context.Context, path string) (types.IEntry, error) {
	p, e := c.WrapPath(path)
	if e != nil {
		return nil, e
	}
	entry, e := c.d.Get(ctx, p)
	return c.wrapEntry(entry), e
}

func (c *ChrootWrapper) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	p, e := c.WrapPath(path)
	if e != nil {
		return nil, e
	}
	entry, e := c.d.Save(ctx, p, size, override, reader)
	return c.wrapEntry(entry), e
}

func (c *ChrootWrapper) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	p, e := c.WrapPath(path)
	if e != nil {
		return nil, e
	}
	entry, e := c.d.MakeDir(ctx, p)
	return c.wrapEntry(entry), e
}

func (c *ChrootWrapper) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	toP, e := c.WrapPath(to)
	if e != nil {
		return nil, e
	}
	entry, e := c.d.Copy(ctx, unwrapEntry(from), toP, override)
	return c.wrapEntry(entry), e
}

func (c *ChrootWrapper) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	toP, e := c.WrapPath(to)
	if e != nil {
		return nil, e
	}
	entry, e := c.d.Move(ctx, unwrapEntry(from), toP, override)
	return c.wrapEntry(entry), e
}

func (c *ChrootWrapper) List(ctx context.Context, path string) ([]types.IEntry, error) {
	p, e := c.WrapPath(path)
	if e != nil {
		return nil, e
	}
	entries, e := c.d.List(ctx, p)
	if e != nil {
		return nil, e
	}
	entries = c.FilterEntries(path, entries)
	return c.wrapEntries(entries), e
}

func (c *ChrootWrapper) Delete(ctx types.TaskCtx, path string) error {
	p, e := c.WrapPath(path)
	if e != nil {
		return e
	}
	return c.d.Delete(ctx, p)
}

func (c *ChrootWrapper) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	p, e := c.WrapPath(path)
	if e != nil {
		return nil, e
	}
	return c.d.Upload(ctx, p, size, override, config)
}

func (c *ChrootWrapper) wrapEntry(e types.IEntry) types.IEntry {
	if e == nil {
		return nil
	}
	return &chrootEntry{
		IEntry: e,
		path:   c.UnwrapPath(e.Path()),
		c:      c,
	}
}

func unwrapEntry(e types.IEntry) types.IEntry {
	ee := drive_util.GetIEntry(e, func(entry types.IEntry) bool {
		_, ok := e.(*chrootEntry)
		return ok
	})
	if ee != nil {
		return ee.(*chrootEntry).IEntry
	}
	return e
}

func (c *ChrootWrapper) wrapEntries(es []types.IEntry) []types.IEntry {
	if es == nil {
		return nil
	}
	result := make([]types.IEntry, 0, len(es))
	for _, e := range es {
		result = append(result, c.wrapEntry(e))
	}
	return result
}

var _ types.IEntryWrapper = (*chrootEntry)(nil)

type chrootEntry struct {
	types.IEntry
	path string
	c    *ChrootWrapper
}

func (c *chrootEntry) Path() string {
	return c.path
}

func (c *chrootEntry) Drive() types.IDrive {
	return c.c
}

func (c *chrootEntry) GetIEntry() types.IEntry {
	return c.IEntry
}
