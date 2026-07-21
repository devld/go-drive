package ftp

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net"
	"net/textproto"
	path2 "path"
	"strconv"
	"sync"
	"time"

	ftpclient "github.com/jlaffaye/ftp"
)

func init() {
	t := i18n.TPrefix("drive.ftp.")
	driveutil.RegisterDrive(driveutil.DriveFactoryConfig{
		Type:        "ftp",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{Label: t("form.host.label"), Type: "text", Field: "host", Required: true, Description: t("form.host.description")},
			{Label: t("form.port.label"), Type: "text", Field: "port", Required: true, Description: t("form.port.description"), DefaultValue: "21"},
			{Label: t("form.user.label"), Type: "text", Field: "user", Description: t("form.user.description")},
			{Label: t("form.password.label"), Type: "password", Field: "password", Description: t("form.password.description")},
			{Label: t("form.concurrent.label"), Type: "text", Field: "concurrent", Description: t("form.concurrent.description")},
			{Label: t("form.timeout.label"), Type: "text", Field: "timeout", Description: t("form.timeout.description")},
			{Label: t("form.cache_ttl.label"), Type: "text", Field: "cache_ttl", Description: t("form.cache_ttl.description")},
		},
		Factory: driveutil.DriveFactory{Create: NewDrive},
	})
}

func NewDrive(ctx context.Context, config types.SM,
	driveUtils driveutil.DriveUtils) (types.IDrive, error) {
	cacheTTL := config.GetDuration("cache_ttl", -1)

	user, password := config["user"], config["password"]
	if user == "" {
		user = "anonymous"
	}
	if password == "" {
		password = "anonymous"
	}
	client := newClientPool(
		net.JoinHostPort(config["host"], strconv.Itoa(config.GetInt("port", 21))),
		user,
		password,
		config.GetDuration("timeout", 5*time.Second),
		config.GetInt("concurrent", 5),
	)
	ftp := &Drive{
		c:        client,
		cacheTTL: cacheTTL,
	}

	if cacheTTL <= 0 {
		ftp.cache = driveutil.DummyCache()
	} else {
		ftp.cache = driveUtils.CreateCache(ftp.deserializeEntry)
	}

	_, e := ftp.List(ctx, "")
	if e != nil {
		_ = client.Close()
		return nil, e
	}

	return ftp, nil
}

var _ types.IDrive = (*Drive)(nil)

type Drive struct {
	c        *clientPool
	cache    driveutil.DriveCache
	cacheTTL time.Duration
}

func (f *Drive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}

func (f *Drive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return &ftpEntry{d: f, isDir: true, modTime: -1}, nil
	}
	if cached, _ := f.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	parentPath := utils.PathParent(path)
	name := utils.PathBase(path)
	entries, e := f.list(ctx, parentPath)
	if e != nil {
		return nil, e
	}
	for _, found := range entries {
		if utils.PathBase(found.Path()) == name {
			_ = f.cache.PutEntry(found, f.cacheTTL)
			return found, nil
		}
	}
	return nil, err.NewNotFoundError()
}

func (f *Drive) Save(ctx types.TaskCtx, path string, _ int64, override bool, reader io.Reader) (types.IEntry, error) {
	if !override {
		if _, e := driveutil.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	c, e := f.c.get(ctx)
	if e != nil {
		return nil, e
	}
	e = c.Stor(path, driveutil.ProgressReader(reader, ctx))
	f.c.release(c, e == nil)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(path, false)
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *Drive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	c, e := f.c.get(ctx)
	if e != nil {
		return nil, e
	}
	e = c.MakeDir(path)
	f.c.release(c, e == nil)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *Drive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (f *Drive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = driveutil.GetSelfEntry(f, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	fromEntry := from.(*ftpEntry)
	if !override {
		if _, e := driveutil.RequireFileNotExists(ctx, f, to); e != nil {
			return nil, e
		}
	}
	c, e := f.c.get(ctx)
	if e != nil {
		return nil, e
	}
	e = c.Rename(fromEntry.path, to)
	f.c.release(c, e == nil)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(to, true)
	_ = f.cache.Evict(utils.PathParent(to), false)
	_ = f.cache.Evict(fromEntry.path, true)
	_ = f.cache.Evict(utils.PathParent(fromEntry.path), false)
	return f.Get(ctx, to)
}

func (f *Drive) list(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	c, e := f.c.get(ctx)
	if e != nil {
		return nil, e
	}
	stats, e := c.List(path)
	f.c.release(c, e == nil)
	if e != nil {
		return nil, mapError(e)
	}
	entries := make([]types.IEntry, len(stats))
	for i, s := range stats {
		entries[i] = f.newFTPEntry(path, s)
	}
	_ = f.cache.PutChildren(path, entries, f.cacheTTL)
	return entries, nil
}

func (f *Drive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	entries, e := f.list(ctx, path)
	if e != nil {
		return nil, e
	}
	if len(entries) == 0 {
		// we need to check whether the folder exists
		_, e := f.Get(ctx, path)
		if e != nil {
			return nil, e
		}
	}
	return entries, nil
}

func (f *Drive) Delete(ctx types.TaskCtx, path string) error {
	deleteRoot, e := f.Get(ctx, path)
	if e != nil {
		return e
	}
	tree, e := driveutil.BuildEntriesTree(ctx, deleteRoot, false)
	if e != nil {
		return e
	}
	entries := driveutil.FlattenEntriesTree(tree, false)
	c, e := f.c.get(ctx)
	if e != nil {
		return e
	}
	reusable := false
	defer func() { f.c.release(c, reusable) }()

	for i := len(entries) - 1; i >= 0; i-- {
		var e error
		if entries[i].Entry.Type().IsDir() {
			e = c.RemoveDir(entries[i].Entry.Path())
		} else {
			e = c.Delete(entries[i].Entry.Path())
		}
		if e != nil {
			return e
		}
		ctx.Progress(1, false)
	}
	reusable = true

	_ = f.cache.Evict(utils.PathParent(path), false)
	_ = f.cache.Evict(path, true)
	return nil
}

func (f *Drive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := driveutil.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (f *Drive) newFTPEntry(path string, stat *ftpclient.Entry) *ftpEntry {
	return &ftpEntry{
		d:       f,
		path:    path2.Join(path, stat.Name),
		size:    int64(stat.Size),
		isDir:   stat.Type == ftpclient.EntryTypeFolder,
		modTime: utils.Millisecond(stat.Time),
	}
}

func (f *Drive) deserializeEntry(ec driveutil.EntryCacheItem) (types.IEntry, error) {
	return &ftpEntry{path: ec.Path, d: f, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir()}, nil
}

func (f *Drive) Dispose() error {
	return f.c.Close()
}

var _ types.IEntry = (*ftpEntry)(nil)

type ftpEntry struct {
	d       *Drive
	path    string
	size    int64
	isDir   bool
	modTime int64
}

func (f *ftpEntry) Path() string {
	return f.path
}

func (f *ftpEntry) Type() types.EntryType {
	if f.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (f *ftpEntry) Size() int64 {
	if f.Type().IsDir() {
		return -1
	}
	return f.size
}

func (f *ftpEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}

func (f *ftpEntry) ModTime() int64 {
	return f.modTime
}

func (f *ftpEntry) Drive() types.IDrive {
	return f.d
}

func (f *ftpEntry) Name() string {
	return utils.PathBase(f.path)
}

func (f *ftpEntry) GetReader(ctx context.Context, start, size int64) (io.ReadCloser, error) {
	if start >= 0 || size > 0 {
		return nil, err.NewUnsupportedError()
	}
	return utils.NewLazyReader(func() (io.ReadCloser, error) {
		c, e := f.d.c.get(ctx)
		if e != nil {
			return nil, e
		}
		response, e := c.Retr(f.path)
		if e != nil {
			f.d.c.release(c, false)
			return nil, mapError(e)
		}
		return &pooledResponse{Response: response, release: func(reusable bool) { f.d.c.release(c, reusable) }}, nil
	}), nil
}

func (f *ftpEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}

func mapError(e error) error {
	var fe *textproto.Error
	if errors.As(e, &fe) {
		return fmt.Errorf("[%d] %s", fe.Code, fe.Msg)
	}
	return e
}

type pooledResponse struct {
	*ftpclient.Response
	release func(bool)
	once    sync.Once
	err     error
}

func (r *pooledResponse) Read(p []byte) (int, error) {
	n, e := r.Response.Read(p)
	if e != nil && !errors.Is(e, io.EOF) {
		r.err = e
	}
	return n, e
}

func (r *pooledResponse) Close() error {
	r.once.Do(func() {
		closeErr := r.Response.Close()
		if r.err == nil {
			r.err = closeErr
		}
		r.release(r.err == nil)
	})
	return r.err
}

type clientPool struct {
	addr     string
	user     string
	password string
	timeout  time.Duration

	slots chan struct{}
	idle  chan *ftpclient.ServerConn
	mu    sync.Mutex
	dead  bool
}

func newClientPool(addr, user, password string, timeout time.Duration, size int) *clientPool {
	if size < 1 {
		size = 1
	}
	p := &clientPool{
		addr: addr, user: user, password: password, timeout: timeout,
		slots: make(chan struct{}, size), idle: make(chan *ftpclient.ServerConn, size),
	}
	for range size {
		p.slots <- struct{}{}
	}
	return p
}

func (p *clientPool) get(ctx context.Context) (*ftpclient.ServerConn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.slots:
	}

	p.mu.Lock()
	if p.dead {
		p.mu.Unlock()
		p.slots <- struct{}{}
		return nil, errors.New("ftp client pool is closed")
	}
	p.mu.Unlock()

	select {
	case c := <-p.idle:
		return c, nil
	default:
	}

	dialer := net.Dialer{Timeout: p.timeout}
	c, e := ftpclient.Dial(p.addr,
		ftpclient.DialWithContext(ctx),
		ftpclient.DialWithShutTimeout(p.timeout),
		ftpclient.DialWithDialFunc(func(network, address string) (net.Conn, error) {
			conn, e := dialer.DialContext(ctx, network, address)
			if e != nil {
				return nil, e
			}
			return &deadlineConn{Conn: conn, timeout: p.timeout}, nil
		}),
	)
	if e == nil {
		e = c.Login(p.user, p.password)
	}
	if e != nil {
		if c != nil {
			_ = c.Quit()
		}
		p.slots <- struct{}{}
		return nil, mapError(e)
	}
	return c, nil
}

func (p *clientPool) release(c *ftpclient.ServerConn, reusable bool) {
	p.mu.Lock()
	if p.dead || !reusable {
		p.mu.Unlock()
		_ = c.Quit()
	} else {
		p.idle <- c
		p.mu.Unlock()
	}
	p.slots <- struct{}{}
}

// deadlineConn restores the old FTP driver's timeout semantics: the timeout
// applies to every control and data connection read/write, not only dialing.
type deadlineConn struct {
	net.Conn
	timeout time.Duration
}

func (c *deadlineConn) Read(p []byte) (int, error) {
	if c.timeout > 0 {
		if e := c.SetReadDeadline(time.Now().Add(c.timeout)); e != nil {
			return 0, e
		}
	}
	return c.Conn.Read(p)
}

func (c *deadlineConn) Write(p []byte) (int, error) {
	if c.timeout > 0 {
		if e := c.SetWriteDeadline(time.Now().Add(c.timeout)); e != nil {
			return 0, e
		}
	}
	return c.Conn.Write(p)
}

func (p *clientPool) Close() error {
	p.mu.Lock()
	if p.dead {
		p.mu.Unlock()
		return nil
	}
	p.dead = true
	p.mu.Unlock()

	var closeErr error
	for {
		select {
		case c := <-p.idle:
			closeErr = errors.Join(closeErr, c.Quit())
		default:
			return closeErr
		}
	}
}
