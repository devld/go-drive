package drive

import (
	"context"
	"go-drive/common/event"
	"go-drive/common/types"
	"io"
)

var _ types.IDrive = (*ListenerWrapper)(nil)

type ListenerWrapper struct {
	types.IDrive

	ctx types.DriveListenerContext
	bus event.Bus
}

func NewListenerWrapper(drive types.IDrive, ctx types.DriveListenerContext, bus event.Bus) *ListenerWrapper {
	return &ListenerWrapper{
		drive,
		ctx,
		bus,
	}
}

func (d *ListenerWrapper) Get(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := d.IDrive.Get(ctx, path)
	if e == nil {
		d.bus.PublishEntryAccessed(d.ctx, path)
	}
	return entry, e
}

func (d *ListenerWrapper) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	entry, e := d.IDrive.Save(ctx, path, size, override, reader)
	if e == nil {
		d.bus.PublishEntryUpdated(d.ctx, path, false)
	}
	return entry, e
}

func (d *ListenerWrapper) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	entry, e := d.IDrive.MakeDir(ctx, path)
	if e == nil {
		d.bus.PublishEntryUpdated(d.ctx, path, false)
	}
	return entry, e
}

func (d *ListenerWrapper) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := d.IDrive.Copy(ctx, from, to, override)
	if e == nil {
		d.bus.PublishEntryAccessed(d.ctx, from.Path())
		d.bus.PublishEntryUpdated(d.ctx, entry.Path(), true)
	}
	return entry, e
}

func (d *ListenerWrapper) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	entry, e := d.IDrive.Move(ctx, from, to, override)
	if e == nil {
		d.bus.PublishEntryDeleted(d.ctx, from.Path())
		d.bus.PublishEntryUpdated(d.ctx, entry.Path(), true)
	}
	return entry, e
}

func (d *ListenerWrapper) List(ctx context.Context, path string) ([]types.IEntry, error) {
	entries, e := d.IDrive.List(ctx, path)
	if e == nil {
		d.bus.PublishEntryAccessed(d.ctx, path)
	}
	return entries, e
}

func (d *ListenerWrapper) Delete(ctx types.TaskCtx, path string) error {
	e := d.IDrive.Delete(ctx, path)
	if e == nil {
		d.bus.PublishEntryDeleted(d.ctx, path)
	}
	return e
}
