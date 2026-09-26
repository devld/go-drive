package driveutil

import (
	"context"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io/fs"
)

// walkDriveFS lists drive entries for doublestar.GlobWalk. Opened files do not
// provide file bytes.
type walkDriveFS struct {
	ctx   context.Context
	drive types.IDrive
}

func newWalkDriveFS(ctx context.Context, drive types.IDrive) *walkDriveFS {
	return &walkDriveFS{ctx: ctx, drive: drive}
}

func (w *walkDriveFS) Open(name string) (fs.File, error) {
	info, e := w.stat(name)
	if e != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: e}
	}
	return &walkDriveFile{info: info}, nil
}

func (w *walkDriveFS) Stat(name string) (fs.FileInfo, error) {
	info, e := w.stat(name)
	if e != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: e}
	}
	return info, nil
}

func (w *walkDriveFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, e := w.drive.List(w.ctx, utils.CleanPath(name))
	if e != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: mapError(e)}
	}
	return utils.ArrayMap(entries, func(t *types.IEntry) fs.DirEntry { return entryFileInfo{*t} }), nil
}

func (w *walkDriveFS) stat(name string) (entryFileInfo, error) {
	entry, e := w.drive.Get(w.ctx, utils.CleanPath(name))
	if e != nil {
		return entryFileInfo{}, mapError(e)
	}
	return entryFileInfo{entry}, nil
}

type walkDriveFile struct {
	info entryFileInfo
}

func (f *walkDriveFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *walkDriveFile) Read([]byte) (int, error) {
	return 0, err.NewUnsupportedError()
}

func (f *walkDriveFile) Close() error {
	return nil
}
