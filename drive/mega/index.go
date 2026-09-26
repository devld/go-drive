package mega

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"

	megaapi "go-drive/drive/mega/internal/gomega"
)

var t = i18n.TPrefix("drive.mega.")

// contentBlockSize is the plaintext fill size for a MEGA content cache.
// MEGA's own chunks are at most 1MiB. A 1MiB fill can re-read a protocol chunk
// that straddles a boundary, including the smaller chunks at the start of a file.
const contentBlockSize int64 = 1 << 20

func RegisterDrive(driveRegistry *driveutil.DriveRegistry) {
	driveRegistry.RegisterDrive(driveutil.DriveFactoryConfig{
		Type:        "mega",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{Field: "email", Label: t("form.email.label"), Type: "text", Required: true, Description: t("form.email.description")},
			{Field: "password", Label: t("form.password.label"), Type: "password", Required: true, Description: t("form.password.description")},
			{Field: "root_path", Label: t("form.root_path.label"), Type: "text", Description: t("form.root_path.description")},
			{Field: "delete_mode", Label: t("form.delete_mode.label"), Type: "checkbox", Description: t("form.delete_mode.description")},
			{Field: "https", Label: t("form.https.label"), Type: "checkbox", Description: t("form.https.description")},
		},
		Factory: driveutil.DriveFactory{Create: NewDrive, InitConfig: InitConfig, Init: Init},
	})
}

// InitConfig signs in when the saved session or the password is enough and reports
// that state as configured. Accounts with two-factor authentication get a one-time code form.
func InitConfig(_ context.Context, config types.SM, driveUtils driveutil.DriveUtils) (*driveutil.DriveInitConfig, error) {
	email, password, credErr := accountCredentials(config)
	if credErr != nil {
		return nil, credErr
	}
	api := prepareAPI(config)
	defer api.Close()
	return probeSession(api, email, password, driveUtils.Data)
}

// Init submits the one-time two-factor code and stores only the resulting session.
func Init(_ context.Context, data types.SM, config types.SM, driveUtils driveutil.DriveUtils) error {
	email, password, credErr := accountCredentials(config)
	if credErr != nil {
		return credErr
	}
	api := prepareAPI(config)
	defer api.Close()
	return loginWithCode(api, email, password, data["mfa"], driveUtils.Data)
}

// NewDrive logs into a MEGA cloud drive and exposes it as a path-based drive.
func NewDrive(_ context.Context, config types.SM, driveUtils driveutil.DriveUtils) (types.IDrive, error) {
	remote, connectErr := connect(config, driveUtils.Data)
	if connectErr != nil {
		return nil, connectErr
	}
	drive := newDrive(remote, config["root_path"], permanentDelete(config))
	content, contentDir, cacheErr := newContentCache(driveUtils.Config)
	if cacheErr != nil {
		_ = remote.close()
		return nil, cacheErr
	}
	drive.content = content
	drive.contentDir = contentDir
	if _, resolveErr := drive.anchor(); resolveErr != nil {
		_ = drive.Dispose()
		return nil, resolveErr
	}
	return drive, nil
}

func newContentCache(config common.Config) (*driveutil.CacheFilePool, string, error) {
	base := config.TempDir
	if base == "" {
		base = os.TempDir()
	}
	parent := filepath.Join(base, "mega")
	if e := os.MkdirAll(parent, 0700); e != nil {
		return nil, "", e
	}
	dir, e := os.MkdirTemp(parent, "drive-")
	if e != nil {
		return nil, "", e
	}
	items := utils.PositiveOr(config.VFS.CacheItems, common.DefaultVFSCacheItems)
	maxBytes := config.VFS.CacheSize.DataSize(common.DefaultVFSCacheSize.DataSize(0))
	pool, e := driveutil.NewCacheFilePool(driveutil.CacheFilePoolOptions{
		MaxEntries:   items,
		MaxBytes:     maxBytes,
		BlockSize:    contentBlockSize,
		Dir:          dir,
		CleanStartup: true,
	})
	if e != nil {
		_ = os.RemoveAll(dir)
		return nil, "", e
	}
	return pool, dir, nil
}

func permanentDelete(config types.SM) bool {
	// "trash" is the previous select value for the rubbish bin.
	if config["delete_mode"] == "trash" {
		return false
	}
	return config.GetBool("delete_mode")
}

func newDrive(remote client, rootPath string, permanent bool) *Drive {
	return &Drive{
		remote:    remote,
		rootPath:  utils.CleanPath(rootPath),
		permanent: permanent,
	}
}

var _ types.IDrive = (*Drive)(nil)
var _ types.IDisposable = (*Drive)(nil)

// Drive is a MEGA cloud-drive backend. Inbox, rubbish, and incoming shares are not mounted.
type Drive struct {
	remote     client
	rootPath   string
	permanent  bool
	content    *driveutil.CacheFilePool
	contentDir string
}

func (d *Drive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}

func (d *Drive) Get(ctx context.Context, path string) (types.IEntry, error) {
	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return d.newEntry("", node{name: "", isDir: true}), nil
	}
	resolved, resolveErr := d.resolve(ctx, path)
	if resolveErr != nil {
		return nil, resolveErr
	}
	return d.newEntry(utils.PathParent(path), resolved), nil
}

func (d *Drive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	if size < 0 {
		return nil, err.NewBadRequestError(i18n.T("api.drive.invalid_file_size"))
	}
	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return nil, err.NewNotAllowedError()
	}
	parent, name, parentErr := d.parent(ctx, path)
	if parentErr != nil {
		return nil, parentErr
	}
	existing, found, lookupErr := d.lookup(ctx, parent.id, name)
	if lookupErr != nil {
		return nil, lookupErr
	}
	if found {
		if !override {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		if existing.isDir {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.copy_type_mismatch2", path, path))
		}
	}

	ctx.Total(size, true)
	created, putErr := d.remote.put(ctx, parent.id, name, size, reader, func(n int64) {
		ctx.Progress(n, false)
	})
	if putErr != nil {
		return nil, mapError(putErr)
	}
	if found && existing.id != created.id {
		if removeErr := d.remote.remove(existing.id, d.permanent); removeErr != nil {
			return nil, mapError(removeErr)
		}
	}
	return d.Get(ctx, path)
}

func (d *Drive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return nil, err.NewNotAllowedError()
	}
	parent, name, parentErr := d.parent(ctx, path)
	if parentErr != nil {
		return nil, parentErr
	}
	if _, found, lookupErr := d.lookup(ctx, parent.id, name); lookupErr != nil {
		return nil, lookupErr
	} else if found {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	if _, mkdirErr := d.remote.mkdir(parent.id, name); mkdirErr != nil {
		return nil, mapError(mkdirErr)
	}
	return d.Get(ctx, path)
}

func (d *Drive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (d *Drive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = driveutil.GetSelfEntry(d, from)
	if from == nil || from.Drive() != d {
		return nil, err.NewUnsupportedError()
	}
	fromPath := utils.CleanPath(from.Path())
	to = utils.CleanPath(to)
	if fromPath == to {
		return nil, err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_same_path_not_allowed"))
	}
	if from.Type().IsDir() && (to == fromPath || strings.HasPrefix(to, fromPath+"/")) {
		return nil, err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	}
	if strings.HasPrefix(fromPath, to+"/") {
		return nil, err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	}
	src, srcErr := d.resolve(ctx, fromPath)
	if srcErr != nil {
		return nil, srcErr
	}
	parent, name, parentErr := d.parent(ctx, to)
	if parentErr != nil {
		return nil, parentErr
	}
	existing, found, lookupErr := d.lookup(ctx, parent.id, name)
	if lookupErr != nil {
		return nil, lookupErr
	}
	if found && existing.id != src.id {
		if !override {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		if existing.isDir != src.isDir {
			if src.isDir {
				return nil, err.NewNotAllowedMessageError(i18n.T("drive.copy_type_mismatch1", fromPath, to))
			}
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.copy_type_mismatch2", fromPath, to))
		}
		if removeErr := d.remote.remove(existing.id, d.permanent); removeErr != nil {
			return nil, mapError(removeErr)
		}
	}
	srcParent, _, srcParentErr := d.parent(ctx, fromPath)
	if srcParentErr != nil {
		return nil, srcParentErr
	}
	if srcParent.id != parent.id {
		if moveErr := d.remote.move(src.id, parent.id); moveErr != nil {
			return nil, mapError(moveErr)
		}
	}
	if src.name != name {
		if renameErr := d.remote.rename(src.id, name); renameErr != nil {
			return nil, mapError(renameErr)
		}
	}
	return d.Get(ctx, to)
}

func (d *Drive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	path = utils.CleanPath(path)
	dir, resolveErr := d.resolve(ctx, path)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if !dir.isDir {
		return nil, err.NewNotAllowedMessageError(t("cannot_list_file"))
	}
	children, childrenErr := d.remote.children(dir.id)
	if childrenErr != nil {
		return nil, mapError(childrenErr)
	}
	visible := visibleNodes(children)
	entries := make([]types.IEntry, 0, len(visible))
	for _, child := range visible {
		entries = append(entries, d.newEntry(path, child))
	}
	return entries, nil
}

func (d *Drive) Delete(ctx types.TaskCtx, path string) error {
	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return err.NewNotAllowedMessageError(t("cannot_delete_root"))
	}
	target, resolveErr := d.resolve(ctx, path)
	if resolveErr != nil {
		return resolveErr
	}
	if removeErr := d.remote.remove(target.id, d.permanent); removeErr != nil {
		return mapError(removeErr)
	}
	return nil
}

func (d *Drive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, existsErr := driveutil.RequireFileNotExists(ctx, d, path); existsErr != nil {
			return nil, existsErr
		}
	}
	return types.UseLocalProvider(size), nil
}

func (d *Drive) Dispose() error {
	var disposeErr error
	if d.content != nil {
		disposeErr = d.content.Dispose()
		d.content = nil
	}
	if d.contentDir != "" {
		if e := os.RemoveAll(d.contentDir); disposeErr == nil {
			disposeErr = e
		}
		d.contentDir = ""
	}
	if d.remote != nil {
		if e := d.remote.close(); disposeErr == nil {
			disposeErr = e
		}
	}
	return disposeErr
}

func (d *Drive) anchor() (node, error) {
	current, rootErr := d.remote.root()
	if rootErr != nil {
		return node{}, mapError(rootErr)
	}
	for _, part := range pathParts(d.rootPath) {
		child, found, lookupErr := d.lookup(context.Background(), current.id, part)
		if lookupErr != nil {
			return node{}, lookupErr
		}
		if !found {
			return node{}, err.NewNotFoundMessageError(t("root_not_found", d.rootPath))
		}
		if !child.isDir {
			return node{}, err.NewNotFoundMessageError(t("root_not_dir", d.rootPath))
		}
		current = child
	}
	return current, nil
}

func (d *Drive) resolve(ctx context.Context, path string) (node, error) {
	current, anchorErr := d.anchor()
	if anchorErr != nil {
		return node{}, anchorErr
	}
	for _, part := range pathParts(path) {
		child, found, lookupErr := d.lookup(ctx, current.id, part)
		if lookupErr != nil {
			return node{}, lookupErr
		}
		if !found {
			return node{}, err.NewNotFoundError()
		}
		current = child
	}
	return current, nil
}

func (d *Drive) parent(ctx context.Context, path string) (node, string, error) {
	name := utils.PathBase(path)
	if !validName(name) {
		return node{}, "", err.NewBadRequestError(i18n.T("drive.invalid_path"))
	}
	parent, parentErr := d.resolve(ctx, utils.PathParent(path))
	if parentErr != nil {
		return node{}, "", parentErr
	}
	if !parent.isDir {
		return node{}, "", err.NewNotAllowedMessageError(t("not_a_directory"))
	}
	return parent, name, nil
}

func (d *Drive) lookup(ctx context.Context, parentID, name string) (node, bool, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return node{}, false, ctxErr
	}
	children, childrenErr := d.remote.children(parentID)
	if childrenErr != nil {
		return node{}, false, mapError(childrenErr)
	}
	found, ok := pickNewest(children, name)
	return found, ok, nil
}

func (d *Drive) newEntry(parentPath string, item node) *megaEntry {
	path := item.name
	if parentPath != "" {
		path = parentPath + "/" + item.name
	}
	size := item.size
	if item.isDir {
		size = -1
	}
	modTime := int64(0)
	if !item.modTime.IsZero() {
		modTime = utils.Millisecond(item.modTime)
	}
	return &megaEntry{drive: d, path: path, size: size, modTime: modTime, isDir: item.isDir}
}

func pathParts(path string) []string {
	path = utils.CleanPath(path)
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\")
}

func visibleNodes(nodes []node) []node {
	chosen := make(map[string]node, len(nodes))
	names := make([]string, 0, len(nodes))
	for _, item := range nodes {
		if !validName(item.name) {
			continue
		}
		previous, ok := chosen[item.name]
		if !ok {
			names = append(names, item.name)
			chosen[item.name] = item
			continue
		}
		if newerNode(item, previous) {
			chosen[item.name] = item
		}
	}
	slices.SortFunc(names, func(left, right string) int {
		return cmp.Compare(left, right)
	})
	result := make([]node, 0, len(names))
	for _, name := range names {
		result = append(result, chosen[name])
	}
	return result
}

func pickNewest(nodes []node, name string) (node, bool) {
	var found node
	ok := false
	for _, item := range nodes {
		if item.name != name || !validName(item.name) {
			continue
		}
		if !ok || newerNode(item, found) {
			found = item
			ok = true
		}
	}
	return found, ok
}

func newerNode(candidate, current node) bool {
	if candidate.modTime.After(current.modTime) {
		return true
	}
	if candidate.modTime.Equal(current.modTime) && candidate.id > current.id {
		return true
	}
	return false
}

func mapError(cause error) error {
	if cause == nil {
		return nil
	}
	var coded err.Error
	if errors.As(cause, &coded) {
		return cause
	}
	switch {
	case errors.Is(cause, megaapi.ENOENT):
		return err.NewNotFoundError()
	case errors.Is(cause, megaapi.EACCESS):
		return err.NewPermissionDeniedError(cause.Error())
	case errors.Is(cause, megaapi.EOVERQUOTA), errors.Is(cause, megaapi.EGOINGOVERQUOTA):
		return err.NewNotAllowedMessageError(t("quota"))
	case errors.Is(cause, megaapi.ECIRCULAR):
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	case errors.Is(cause, megaapi.ESID):
		return err.NewUnauthorizedError(t("session_expired"))
	default:
		return cause
	}
}

var _ types.IEntry = (*megaEntry)(nil)

type megaEntry struct {
	drive   *Drive
	path    string
	size    int64
	modTime int64
	isDir   bool
}

func (e *megaEntry) Path() string { return e.path }

func (e *megaEntry) Type() types.EntryType {
	if e.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (e *megaEntry) Size() int64 {
	if e.isDir {
		return -1
	}
	return e.size
}

func (e *megaEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}

func (e *megaEntry) ModTime() int64 { return e.modTime }

func (e *megaEntry) Drive() types.IDrive { return e.drive }

func (e *megaEntry) Name() string { return utils.PathBase(e.path) }

func (e *megaEntry) GetReader(ctx context.Context, rg types.ReaderRange) (io.ReadCloser, error) {
	if e.isDir {
		return nil, err.NewUnsupportedError()
	}
	if e.drive.content == nil {
		return e.openChunkReader(ctx, rg)
	}
	reader, readErr := e.drive.content.GetReader(ctx, contentKey(e), e.size, e.openChunkReader)
	if readErr != nil {
		return nil, readErr
	}
	if rg.IsFullRequest() {
		return reader, nil
	}
	start := rg.Start
	if start > e.size {
		start = e.size
	}
	if _, readErr = reader.Seek(start, io.SeekStart); readErr != nil {
		_ = reader.Close()
		return nil, readErr
	}
	if rg.Size <= 0 {
		return reader, nil
	}
	return driveutil.LimitReadCloser(reader, rg.Size), nil
}

func contentKey(entry *megaEntry) string {
	return fmt.Sprintf("%s,m:%d,s:%d", entry.path, entry.modTime, entry.size)
}

func (e *megaEntry) openChunkReader(ctx context.Context, rg types.ReaderRange) (io.ReadCloser, error) {
	return utils.NewLazyReader(func() (io.ReadCloser, error) {
		resolved, resolveErr := e.drive.resolve(ctx, e.path)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if resolved.isDir {
			return nil, err.NewUnsupportedError()
		}
		download, openErr := e.drive.remote.open(resolved.id)
		if openErr != nil {
			return nil, mapError(openErr)
		}
		return newChunkReader(ctx, download, rg), nil
	}), nil
}

func (e *megaEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}

type chunkReader struct {
	ctx       context.Context
	download  downloader
	offset    int64
	remaining int64
	index     int
	buf       []byte
	bufOff    int
	err       error
	closed    bool
}

func newChunkReader(ctx context.Context, download downloader, rg types.ReaderRange) *chunkReader {
	offset := int64(0)
	remaining := int64(-1)
	if !rg.IsFullRequest() {
		offset = rg.Start
		if rg.Size > 0 {
			remaining = rg.Size
		}
	}
	return &chunkReader{ctx: ctx, download: download, offset: offset, remaining: remaining}
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.err != nil {
		return 0, r.err
	}
	if r.remaining == 0 {
		r.err = io.EOF
		return 0, io.EOF
	}
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		r.err = ctxErr
		return 0, ctxErr
	}
	for r.bufOff >= len(r.buf) {
		if r.index >= r.download.chunkCount() {
			r.err = io.EOF
			return 0, io.EOF
		}
		position, chunkSize, locationErr := r.download.chunkAt(r.index)
		if locationErr != nil {
			r.err = locationErr
			return 0, locationErr
		}
		end := position + int64(chunkSize)
		if chunkSize == 0 || end <= r.offset {
			r.index++
			continue
		}
		chunk, readErr := r.download.readChunk(r.index)
		r.index++
		if readErr != nil {
			r.err = readErr
			return 0, readErr
		}
		if position < r.offset {
			skip := min(int(r.offset-position), len(chunk))
			chunk = chunk[skip:]
		}
		if len(chunk) == 0 {
			continue
		}
		r.buf = chunk
		r.bufOff = 0
	}
	available := len(r.buf) - r.bufOff
	if r.remaining > 0 && int64(available) > r.remaining {
		available = int(r.remaining)
	}
	n := copy(p, r.buf[r.bufOff:r.bufOff+available])
	r.bufOff += n
	if r.remaining > 0 {
		r.remaining -= int64(n)
	}
	return n, nil
}

func (r *chunkReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.download.finish()
}
