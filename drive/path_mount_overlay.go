package drive

import (
	"context"
	"errors"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"io"
	"log"
	path2 "path"
	"strings"
	"sync"
)

const maxMountDepth = 10

var _ types.IDrive = (*PathMountOverlayDrive)(nil)

// PathMountOverlayDrive provides the virtual path-mount namespace on top of a
// mount-agnostic dispatcher. Paths inside a mount are exclusive: a missing
// target is not allowed to fall back to the dispatcher's path with the same
// name. Paths above mounts merge physical and virtual children.
type PathMountOverlayDrive struct {
	lower        types.IDrive
	mountStorage *storage.PathMountDAO

	mountTree *utils.PathTreeNode[*types.PathMount]
	mountsMux *sync.RWMutex
}

func NewPathMountOverlayDrive(lower types.IDrive, mountStorage *storage.PathMountDAO) *PathMountOverlayDrive {
	return &PathMountOverlayDrive{
		lower:        lower,
		mountStorage: mountStorage,
		mountTree:    utils.NewPathTreeNodeNonLock[*types.PathMount](""),
		mountsMux:    &sync.RWMutex{},
	}
}

func (d *PathMountOverlayDrive) reloadMounts() error {
	mounts, e := d.mountStorage.GetMounts()
	if e != nil {
		return e
	}
	tree := utils.NewPathTreeNodeNonLock[*types.PathMount]("")
	for _, m := range mounts {
		mount := m
		tree.Add(path2.Join(*m.Path, m.Name), &mount)
	}
	d.mountsMux.Lock()
	d.mountTree = tree
	d.mountsMux.Unlock()
	return nil
}

func (d *PathMountOverlayDrive) tree() *utils.PathTreeNode[*types.PathMount] {
	d.mountsMux.RLock()
	defer d.mountsMux.RUnlock()
	return d.mountTree
}

func (d *PathMountOverlayDrive) Meta(ctx context.Context) (types.DriveMeta, error) {
	return d.lower.Meta(ctx)
}

func checkMountDepth(depth int) error {
	if depth > maxMountDepth {
		return errors.New("maximum mounting depth exceeded")
	}
	return nil
}

func (d *PathMountOverlayDrive) matchedMount(path string) (*types.PathMount, string) {
	var mount *types.PathMount
	var matchDepth int
	depth := 0
	d.tree().GetCb(path, func(node *utils.PathTreeNode[*types.PathMount]) {
		if node.Data != nil && node.Data.MountAt != "" {
			mount = node.Data
			matchDepth = depth
		}
		depth++
	})
	if mount == nil {
		return nil, ""
	}
	cleanPath := utils.CleanPath(path)
	segments := strings.Split(cleanPath, "/")
	prefix := strings.Join(segments[:matchDepth], "/")
	return mount, path2.Join(mount.MountAt, utils.CleanPath(cleanPath[len(prefix):]))
}

func (d *PathMountOverlayDrive) hasDescendantMounts(path string) bool {
	node, _ := d.tree().Get(path)
	return node != nil && len(node.Children()) > 0
}

func (d *PathMountOverlayDrive) phantomChildDirs(path string) []string {
	node, _ := d.tree().Get(path)
	if node == nil {
		return nil
	}
	var keys []string
	for _, child := range node.Children() {
		if child.Data == nil {
			keys = append(keys, child.Key())
		}
	}
	return keys
}

func (d *PathMountOverlayDrive) resolveMountedChildren(path string) ([]types.PathMount, bool) {
	node, _ := d.tree().Get(path)
	if node == nil {
		return nil, false
	}
	result := make([]types.PathMount, 0)
	isSelf := node.Data != nil
	node.Visit(func(n *utils.PathTreeNode[*types.PathMount]) {
		if n.Data != nil {
			result = append(result, *n.Data)
		}
	})
	return result, isSelf
}

// resolveMountTreePath follows path mounts until it reaches the namespace in
// which mount records for path are stored. A direct mount-tree match takes
// precedence over following that mount: callers that copy or move the mount
// point itself must update the record, not operate on its target.
func (d *PathMountOverlayDrive) resolveMountTreePath(path string, depth int) (string, []types.PathMount, bool, error) {
	if e := checkMountDepth(depth); e != nil {
		return "", nil, false, e
	}
	children, isSelf := d.resolveMountedChildren(path)
	if len(children) > 0 {
		return path, children, isSelf, nil
	}
	if _, target := d.matchedMount(path); target != "" {
		return d.resolveMountTreePath(target, depth+1)
	}
	return path, nil, false, nil
}

func isPathMountVirtualEntry(entry types.IEntry) bool {
	return driveutil.GetIEntry(entry, func(candidate types.IEntry) bool {
		_, ok := candidate.(*pathMountVirtualEntry)
		return ok
	}) != nil
}

func remapMounts(mounts []types.PathMount, from, to string, keepID bool) []types.PathMount {
	result := make([]types.PathMount, 0, len(mounts))
	for _, mount := range mounts {
		mountPath := path2.Join(*mount.Path, mount.Name)
		targetPath := path2.Join(to, strings.TrimPrefix(mountPath, from))
		targetParent := utils.PathParent(targetPath)
		if !keepID {
			mount.ID = 0
		}
		mount.Path = &targetParent
		mount.Name = utils.PathBase(targetPath)
		result = append(result, mount)
	}
	return result
}

func mergeMounts(first, second []types.PathMount) []types.PathMount {
	result := make([]types.PathMount, 0, len(first)+len(second))
	seen := make(map[uint]bool, len(first)+len(second))
	for _, mounts := range [][]types.PathMount{first, second} {
		for _, mount := range mounts {
			if mount.ID != 0 && seen[mount.ID] {
				continue
			}
			if mount.ID != 0 {
				seen[mount.ID] = true
			}
			result = append(result, mount)
		}
	}
	return result
}

func (d *PathMountOverlayDrive) replaceMounts(deletes, mounts []types.PathMount) error {
	if e := d.mountStorage.DeleteAndSaveMounts(deletes, mounts, true); e != nil {
		return e
	}
	return d.reloadMounts()
}

func (d *PathMountOverlayDrive) destinationMounts(path string, override bool) []types.PathMount {
	if !override {
		return nil
	}
	mounts, _ := d.resolveMountedChildren(path)
	return mounts
}

func (d *PathMountOverlayDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	return d.get(ctx, path, 0)
}

func (d *PathMountOverlayDrive) get(ctx context.Context, path string, depth int) (types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if mount, target := d.matchedMount(path); mount != nil {
		entry, e := d.get(ctx, target, depth+1)
		if e != nil {
			return nil, e
		}
		mountAt := ""
		if path2.Join(*mount.Path, mount.Name) == utils.CleanPath(path) {
			mountAt = mount.MountAt
		}
		return d.wrapEntry(entry, path, mountAt), nil
	}
	entry, e := d.lower.Get(ctx, path)
	if e != nil {
		if err.IsNotFoundError(e) && d.hasDescendantMounts(path) {
			return &pathMountVirtualEntry{drive: d, path: path}, nil
		}
		return nil, e
	}
	return d.wrapEntry(entry, path, ""), nil
}

func (d *PathMountOverlayDrive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	return d.save(ctx, path, size, override, reader, 0)
}

func (d *PathMountOverlayDrive) save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader, depth int) (types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(path); target != "" {
		entry, e := d.save(ctx, target, size, override, reader, depth+1)
		if e != nil {
			return nil, e
		}
		return d.wrapEntry(entry, path2.Join(utils.PathParent(path), utils.PathBase(entry.Path())), ""), nil
	}
	entry, e := d.lower.Save(ctx, path, size, override, reader)
	if e != nil {
		return nil, e
	}
	return d.wrapEntry(entry, entry.Path(), ""), nil
}

func (d *PathMountOverlayDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	return d.makeDir(ctx, path, 0)
}

func (d *PathMountOverlayDrive) makeDir(ctx context.Context, path string, depth int) (types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(path); target != "" {
		entry, e := d.makeDir(ctx, target, depth+1)
		if e != nil {
			return nil, e
		}
		return d.wrapEntry(entry, path, ""), nil
	}
	entry, e := d.lower.MakeDir(ctx, path)
	if e != nil {
		return nil, e
	}
	return d.wrapEntry(entry, path, ""), nil
}

func (d *PathMountOverlayDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	return d.list(ctx, path, 0)
}

func (d *PathMountOverlayDrive) list(ctx context.Context, path string, depth int) ([]types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(path); target != "" {
		entries, e := d.list(ctx, target, depth+1)
		if e != nil {
			return nil, e
		}
		return d.mergeMountChildren(ctx, path, d.remapEntries(entries, path), depth)
	}

	entries, e := d.lower.List(ctx, path)
	if e != nil {
		if err.IsNotFoundError(e) && d.hasDescendantMounts(path) {
			entries = make([]types.IEntry, 0)
		} else {
			return nil, e
		}
	} else {
		entries = d.wrapEntries(entries)
	}

	return d.mergeMountChildren(ctx, path, entries, depth)
}

func (d *PathMountOverlayDrive) mergeMountChildren(ctx context.Context, path string, entries []types.IEntry, depth int) ([]types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	mountNode, _ := d.tree().Get(path)
	if mountNode != nil {
		mounted := make(map[string]types.IEntry)
		for _, child := range mountNode.Children() {
			if child.Data == nil {
				continue
			}
			mount := child.Data
			entry, getErr := d.get(ctx, mount.MountAt, depth+1)
			if getErr != nil {
				if !err.IsNotFoundError(getErr) {
					log.Printf("get mounted entry(%s) error: %v", mount.MountAt, getErr)
				}
				continue
			}
			mounted[child.Key()] = d.wrapEntry(entry, path2.Join(path, child.Key()), mount.MountAt)
		}
		if len(mounted) > 0 {
			merged := make([]types.IEntry, 0, len(entries)+len(mounted))
			for _, entry := range entries {
				if mounted[utils.PathBase(entry.Path())] == nil {
					merged = append(merged, entry)
				}
			}
			for _, entry := range mounted {
				merged = append(merged, entry)
			}
			entries = merged
		}
	}

	existing := make(map[string]bool, len(entries))
	for _, entry := range entries {
		existing[utils.PathBase(entry.Path())] = true
	}
	for _, name := range d.phantomChildDirs(path) {
		if !existing[name] {
			entries = append(entries, &pathMountVirtualEntry{drive: d, path: path2.Join(path, name)})
		}
	}
	return entries, nil
}

func (d *PathMountOverlayDrive) Delete(ctx types.TaskCtx, path string) error {
	return d.delete(ctx, path, 0)
}

func (d *PathMountOverlayDrive) delete(ctx types.TaskCtx, path string, depth int) error {
	if e := checkMountDepth(depth); e != nil {
		return e
	}
	children, isSelf := d.resolveMountedChildren(path)
	hadMountChildren := len(children) > 0
	if hadMountChildren {
		if e := d.mountStorage.DeleteMounts(children); e != nil {
			return e
		}
		if e := d.reloadMounts(); e != nil {
			return e
		}
		if isSelf {
			return nil
		}
	}
	if _, target := d.matchedMount(path); target != "" {
		e := d.delete(ctx, target, depth+1)
		if e != nil && hadMountChildren && err.IsNotFoundError(e) {
			return nil
		}
		return e
	}
	e := d.lower.Delete(ctx, path)
	if e != nil && hadMountChildren && err.IsNotFoundError(e) {
		return nil
	}
	return e
}

func (d *PathMountOverlayDrive) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	return d.upload(ctx, path, size, override, config, 0)
}

func (d *PathMountOverlayDrive) upload(ctx context.Context, path string, size int64, override bool, config types.SM, depth int) (*types.DriveUploadConfig, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(path); target != "" {
		result, e := d.upload(ctx, target, size, override, config, depth+1)
		if result != nil && result.Path != "" {
			copy := *result
			copy.Path = path2.Join(utils.PathParent(path), utils.PathBase(result.Path))
			result = &copy
		}
		return result, e
	}
	return d.lower.Upload(ctx, path, size, override, config)
}

func (d *PathMountOverlayDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	source, mounts, isSelf, e := d.resolveMountTreePath(from.Path(), 0)
	if e != nil {
		return nil, e
	}
	if isSelf {
		destination := to
		if !override {
			destination, e = driveutil.FindNonExistsEntryName(ctx, d, to)
			if e != nil {
				return nil, e
			}
		}
		destinationMountPath, _, _, e := d.resolveMountTreePath(destination, 0)
		if e != nil {
			return nil, e
		}
		newMounts := remapMounts(mounts, source, destinationMountPath, false)
		if e := d.replaceMounts(d.destinationMounts(destinationMountPath, override), newMounts); e != nil {
			return nil, e
		}
		return d.Get(ctx, destination)
	}
	if len(mounts) == 0 {
		return d.copyWithoutMountChildren(ctx, from, to, override, 0)
	}

	e = driveutil.CopyAll(ctx, from, d, to,
		func(entry types.IEntry, _ types.IDrive, target string, taskCtx types.TaskCtx) error {
			_, copyErr := d.copyWithoutMountChildren(task.NewCtxWrapper(taskCtx, true, false), entry, target, override, 0)
			return copyErr
		}, nil)
	if e != nil {
		return nil, e
	}
	return d.Get(ctx, to)
}

func (d *PathMountOverlayDrive) copyWithoutMountChildren(ctx types.TaskCtx, from types.IEntry, to string, override bool, depth int) (types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(to); target != "" {
		entry, e := d.copyWithoutMountChildren(ctx, from, target, override, depth+1)
		if e != nil {
			return nil, e
		}
		return d.wrapEntry(entry, path2.Join(utils.PathParent(to), utils.PathBase(entry.Path())), ""), nil
	}
	entry, e := d.lower.Copy(ctx, from, to, override)
	if e != nil {
		return nil, e
	}
	return d.wrapEntry(entry, entry.Path(), ""), nil
}

func (d *PathMountOverlayDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	source, children, isSelf, e := d.resolveMountTreePath(from.Path(), 0)
	if e != nil {
		return nil, e
	}
	if len(children) > 0 {
		destination := to
		if !override {
			destination, e = driveutil.FindNonExistsEntryName(ctx, d, to)
			if e != nil {
				return nil, e
			}
		}
		destinationMountPath, _, _, e := d.resolveMountTreePath(destination, 0)
		if e != nil {
			return nil, e
		}
		movedMounts := remapMounts(children, source, destinationMountPath, true)
		deletes := mergeMounts(children, d.destinationMounts(destinationMountPath, override))
		if isSelf {
			if e := d.replaceMounts(deletes, movedMounts); e != nil {
				return nil, e
			}
			return d.Get(ctx, destination)
		}
		if isPathMountVirtualEntry(from) {
			if override {
				if _, getErr := d.lower.Get(ctx, destinationMountPath); getErr == nil {
					if deleteErr := d.lower.Delete(ctx, destinationMountPath); deleteErr != nil {
						return nil, deleteErr
					}
				} else if !err.IsNotFoundError(getErr) {
					return nil, getErr
				}
			}
			if _, e := d.lower.MakeDir(ctx, destinationMountPath); e != nil {
				return nil, e
			}
		} else {
			if _, moveErr := d.moveWithoutMountChildren(ctx, from, destinationMountPath, override, 0); moveErr != nil {
				return nil, moveErr
			}
		}
		if e := d.replaceMounts(deletes, movedMounts); e != nil {
			return nil, e
		}
		return d.Get(ctx, destination)
	}
	return d.moveWithoutMountChildren(ctx, from, to, override, 0)
}

func (d *PathMountOverlayDrive) moveWithoutMountChildren(ctx types.TaskCtx, from types.IEntry, to string, override bool, depth int) (types.IEntry, error) {
	if e := checkMountDepth(depth); e != nil {
		return nil, e
	}
	if _, target := d.matchedMount(to); target != "" {
		entry, e := d.moveWithoutMountChildren(ctx, from, target, override, depth+1)
		if e != nil {
			return nil, e
		}
		return d.wrapEntry(entry, path2.Join(utils.PathParent(to), utils.PathBase(entry.Path())), ""), nil
	}
	entry, e := d.lower.Move(ctx, from, to, override)
	if e != nil {
		return nil, e
	}
	return d.wrapEntry(entry, entry.Path(), ""), nil
}

func (d *PathMountOverlayDrive) wrapEntry(entry types.IEntry, path, mountAt string) types.IEntry {
	if entry == nil {
		return nil
	}
	return &pathMountEntry{IEntry: entry, drive: d, path: path, mountAt: mountAt}
}

func (d *PathMountOverlayDrive) wrapEntries(entries []types.IEntry) []types.IEntry {
	result := make([]types.IEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, d.wrapEntry(entry, entry.Path(), ""))
	}
	return result
}

func (d *PathMountOverlayDrive) remapEntries(entries []types.IEntry, dir string) []types.IEntry {
	result := make([]types.IEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, d.wrapEntry(entry, path2.Join(dir, utils.PathBase(entry.Path())), ""))
	}
	return result
}

var _ types.IEntryWrapper = (*pathMountEntry)(nil)

type pathMountEntry struct {
	types.IEntry
	drive   *PathMountOverlayDrive
	path    string
	mountAt string
}

func (e *pathMountEntry) Path() string            { return e.path }
func (e *pathMountEntry) Name() string            { return utils.PathBase(e.path) }
func (e *pathMountEntry) Drive() types.IDrive     { return e.drive }
func (e *pathMountEntry) GetIEntry() types.IEntry { return e.IEntry }
func (e *pathMountEntry) Meta() types.EntryMeta {
	meta := e.IEntry.Meta()
	if e.mountAt != "" {
		meta.Props = utils.MapCopy(meta.Props, nil)
		meta.Props["mountAt"] = e.mountAt
	}
	return meta
}

var _ types.IEntry = (*pathMountVirtualEntry)(nil)

type pathMountVirtualEntry struct {
	drive *PathMountOverlayDrive
	path  string
}

func (e *pathMountVirtualEntry) Path() string          { return e.path }
func (e *pathMountVirtualEntry) Name() string          { return utils.PathBase(e.path) }
func (e *pathMountVirtualEntry) Type() types.EntryType { return types.TypeDir }
func (e *pathMountVirtualEntry) Size() int64           { return -1 }
func (e *pathMountVirtualEntry) ModTime() int64        { return -1 }
func (e *pathMountVirtualEntry) Drive() types.IDrive   { return e.drive }
func (e *pathMountVirtualEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true}
}
func (e *pathMountVirtualEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return nil, err.NewNotAllowedError()
}
func (e *pathMountVirtualEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewNotAllowedError()
}
