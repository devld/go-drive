package server

import (
	"context"
	"go-drive/common/types"
	"io"
)

type DriveListenerWrapper struct {
	drive types.IDrive

	ctx       types.DriveListenerContext
	listeners []types.IDriveListener
}

func NewDriveListenerWrapper(drive types.IDrive, ctx types.DriveListenerContext, listeners []types.IDriveListener) *DriveListenerWrapper {
	return &DriveListenerWrapper{
		drive:     drive,
		listeners: listeners,
		ctx:       ctx,
	}
}

func (d *DriveListenerWrapper) Meta(ctx context.Context) types.DriveMeta {
	return d.drive.Meta(ctx)
}

func (d *DriveListenerWrapper) Get(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := d.drive.Get(ctx, path)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnAccess(d.ctx, entry)
		}
	}
	return entry, e
}

func (d *DriveListenerWrapper) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	entry, e := d.drive.Save(ctx, path, size, override, reader)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnUpdated(d.ctx, entry, false)
		}
	}
	return entry, e
}

func (d *DriveListenerWrapper) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := d.drive.MakeDir(ctx, path)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnUpdated(d.ctx, entry, false)
		}
	}
	return entry, e
}

func (d *DriveListenerWrapper) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := d.drive.Copy(ctx, from, to, override)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnUpdated(d.ctx, entry, true)
		}
	}
	return entry, e
}

func (d *DriveListenerWrapper) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := d.drive.Move(ctx, from, to, override)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnDeleted(d.ctx, from.Path())
			listener.OnUpdated(d.ctx, entry, true)
		}
	}
	return entry, e
}

func (d *DriveListenerWrapper) List(ctx context.Context, path string) ([]types.IEntry, error) {
	entries, e := d.drive.List(ctx, path)
	if e == nil {
		entry, e := d.drive.Get(ctx, path)
		if e == nil {
			for _, listener := range d.listeners {
				listener.OnAccess(d.ctx, entry)
			}
		}
	}
	return entries, e
}

func (d *DriveListenerWrapper) Delete(ctx types.TaskCtx, path string) error {
	e := d.drive.Delete(ctx, path)
	if e == nil {
		for _, listener := range d.listeners {
			listener.OnDeleted(d.ctx, path)
		}
	}
	return e
}

func (d *DriveListenerWrapper) Upload(ctx context.Context, path string, size int64, override bool, config types.SM) (*types.DriveUploadConfig, error) {
	return d.drive.Upload(ctx, path, size, override, config)
}
