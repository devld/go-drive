package sftp

import (
	"context"
	"errors"
	"fmt"
	"github.com/secsy/goftp"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net"
	"os"
	path2 "path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func init() {
	t := i18n.TPrefix("drive.sftp.")
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "sftp",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{Label: t("form.host.label"), Type: "text", Field: "host", Required: true, Description: t("form.host.description")},
			{Label: t("form.port.label"), Type: "text", Field: "port", Required: true, Description: t("form.port.description"), DefaultValue: "22"},
			{Label: t("form.user.label"), Type: "text", Field: "user", Required: true, Description: t("form.user.description")},
			{Label: t("form.password.label"), Type: "password", Field: "password", Required: true, Description: t("form.password.description")},
			{Label: t("form.cache_ttl.label"), Type: "text", Field: "cache_ttl", Description: t("form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewSftpDrive},
	})
}

func NewSftpDrive(_ context.Context, config types.SM, driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	cacheTTL := config.GetDuration("cache_ttl", -1)

	sshConfig := &ssh.ClientConfig{
		User: config["user"],
		Auth: []ssh.AuthMethod{
			ssh.Password(config["password"]),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	p, e := ConcurrentPool("tcp", fmt.Sprintf("%v:%v", config["host"], config["port"]), sshConfig)
	if e != nil {
		return nil, e
	}

	conn, e := p.GetConn()
	if conn == nil || e != nil {
		return nil, e
	}

	client, e := sftp.NewClient(conn)
	if e != nil {
		return nil, e
	}

	s := &SFTPDrive{
		c:        client,
		p:        p,
		cacheTTL: cacheTTL,
	}
	if cacheTTL <= 0 {
		s.cache = drive_util.DummyCache()
	} else {
		s.cache = driveUtils.CreateCache(s.deserializeEntry, nil)
	}

	return s, nil
}

func isSftpRootPath(path string) bool {
	return path == "/"
}

type SFTPDrive struct {
	c        *sftp.Client
	p        Pool
	cache    drive_util.DriveCache
	cacheTTL time.Duration
}

func (f *SFTPDrive) InitConn() {
	if f.p.IsClosed() {
		conn, e := f.p.GetConn()
		if conn == nil || e != nil {
			panic(e)
		}
		client, e := sftp.NewClient(conn)
		if e != nil {
			panic(e)
		}

		f.c = client
	}
}

func (f *SFTPDrive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{Writable: true}
}

func (f *SFTPDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	// Linux 路径格式必须 / 开始
	path = "/" + path
	if isSftpRootPath(path) {
		return &sftpEntry{d: f, isDir: true, modTime: -1}, nil
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

func (f *SFTPDrive) Save(ctx types.TaskCtx, path string, _ int64, override bool, reader io.Reader) (types.IEntry, error) {
	path = "/" + path
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	f.InitConn()
	file, e := f.c.Create(path)
	if e != nil {
		return nil, mapError(e)
	}
	_, e = file.ReadFrom(reader)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(path, false)
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *SFTPDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	path = "/" + path
	f.InitConn()
	e := f.c.Mkdir(path)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *SFTPDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (f *SFTPDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(f, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	fromEntry := from.(*sftpEntry)
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, to); e != nil {
			return nil, e
		}
	}
	f.InitConn()
	e := f.c.Rename(fromEntry.path, to)
	if e != nil {
		return nil, mapError(e)
	}
	_ = f.cache.Evict(to, true)
	_ = f.cache.Evict(utils.PathParent(to), false)
	_ = f.cache.Evict(fromEntry.path, true)
	_ = f.cache.Evict(utils.PathParent(fromEntry.path), false)
	return f.Get(ctx, to)
}

func (f *SFTPDrive) list(_ context.Context, path string) ([]types.IEntry, error) {
	path = "/" + path
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	f.InitConn()
	stats, e := f.c.ReadDir(path)
	if e != nil {
		return nil, mapError(e)
	}
	entries := make([]types.IEntry, len(stats))
	for i, s := range stats {
		entries[i] = f.newSFTPEntry(path, s)
	}
	_ = f.cache.PutChildren(path, entries, f.cacheTTL)
	return entries, nil
}

func (f *SFTPDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	path = "/" + path
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	f.InitConn()
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

func (f *SFTPDrive) Delete(ctx types.TaskCtx, path string) error {
	path = "/" + path
	deleteRoot, e := f.Get(ctx, path)
	if e != nil {
		return e
	}
	tree, e := drive_util.BuildEntriesTree(ctx, deleteRoot, false)
	if e != nil {
		return e
	}
	entries := drive_util.FlattenEntriesTree(tree)

	for i := len(entries) - 1; i >= 0; i-- {
		var e error
		if entries[i].Type().IsDir() {
			e = f.c.RemoveDirectory(entries[i].Path())
		} else {
			e = f.c.Remove(entries[i].Path())
		}
		if e != nil {
			return e
		}
		ctx.Progress(1, false)
	}

	_ = f.cache.Evict(utils.PathParent(path), false)
	_ = f.cache.Evict(path, true)
	return nil
}

func (f *SFTPDrive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	path = "/" + path
	f.InitConn()
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (f *SFTPDrive) newSFTPEntry(path string, stat os.FileInfo) *sftpEntry {
	path = "/" + path
	return &sftpEntry{
		d:       f,
		path:    path2.Join(path, stat.Name()),
		size:    stat.Size(),
		isDir:   stat.IsDir(),
		modTime: utils.Millisecond(stat.ModTime()),
	}
}

func (f *SFTPDrive) deserializeEntry(dat string) (types.IEntry, error) {
	ec, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	return &sftpEntry{path: ec.Path, d: f, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir()}, nil
}

type sftpEntry struct {
	d       *SFTPDrive
	path    string
	size    int64
	isDir   bool
	modTime int64
}

func (f *sftpEntry) Path() string {
	return f.path
}

func (f *sftpEntry) Type() types.EntryType {
	if f.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (f *sftpEntry) Size() int64 {
	if f.Type().IsDir() {
		return -1
	}
	return f.size
}

func (f *sftpEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}

func (f *sftpEntry) ModTime() int64 {
	return f.modTime
}

func (f *sftpEntry) Drive() types.IDrive {
	return f.d
}

func (f *sftpEntry) Name() string {
	return utils.PathBase(f.path)
}

func (f *sftpEntry) GetReader(context.Context) (io.ReadCloser, error) {
	return utils.NewLazyReader(func() (io.ReadCloser, error) {
		r, w := io.Pipe()
		go func() {
			file, e := f.d.c.Open(f.path)
			if e != nil {
				_ = r.CloseWithError(e)
			}
			_, e = file.WriteTo(w)
			if e != nil {
				_ = r.CloseWithError(e)
			}
			_ = w.Close()
		}()
		return r, nil
	}), nil
}

func (f *sftpEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}

var connectionTimeout = time.Minute * 10

type Pool interface {
	GetConn() (*ssh.Client, error)
	Clean()
	IsClosed() bool
}

func ConcurrentPool(protocol, addr string, config *ssh.ClientConfig) (Pool, error) {
	sshClient, e := ssh.Dial(protocol, addr, config)

	if e != nil {
		return &concurPool{}, e
	}
	pool := &concurPool{
		protocol: protocol,
		addr:     addr,
		config:   config,
		client:   sshClient,
		isClosed: false,
	}
	pool.timeout = utils.TimeTick(pool.Clean, connectionTimeout)
	return pool, nil
}

type concurPool struct {
	protocol string
	addr     string
	config   *ssh.ClientConfig
	client   *ssh.Client
	timeout  func()
	isClosed bool
}

func (p *concurPool) GetConn() (*ssh.Client, error) {
	if p.isClosed {
		p.isClosed = false
		sshClient, e := ssh.Dial(p.protocol, p.addr, p.config)
		if e != nil {
			return &ssh.Client{}, e
		}
		p.client = sshClient
	}
	return p.client, nil
}

func (p *concurPool) Clean() {
	p.isClosed = true
	_ = p.client.Close()
}

func (p *concurPool) IsClosed() bool {
	return p.isClosed
}

func mapError(e error) error {
	fe, ok := e.(goftp.Error)
	if !ok {
		return e
	}
	return errors.New(fmt.Sprintf("[%d] %s", fe.Code(), fe.Message()))
}
