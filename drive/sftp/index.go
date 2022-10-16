package sftp

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net"
	"os"
	path2 "path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var t = i18n.TPrefix("drive.sftp.")

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "sftp",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{Label: t("form.host.label"), Type: "text", Field: "host", Required: true, Description: t("form.host.description")},
			{Label: t("form.port.label"), Type: "text", Field: "port", Description: t("form.port.description"), DefaultValue: "22"},
			{Label: t("form.user.label"), Type: "text", Field: "user", Required: true, Description: t("form.user.description")},
			{Label: t("form.password.label"), Type: "password", Field: "password", Secret: "-------HIDDEN-------", Description: t("form.password.description")},
			{Label: t("form.priv_key.label"), Type: "textarea", Field: "priv_key", Secret: "-------HIDDEN-------", Description: t("form.priv_key.description")},
			{Label: t("form.host_key.label"), Type: "textarea", Field: "host_key", Description: t("form.host_key.description")},
			{Label: t("form.root_path.label"), Type: "text", Field: "root_path", Description: t("form.root_path.description")},
			{Label: t("form.cache_ttl.label"), Type: "text", Field: "cache_ttl", Description: t("form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewDrive},
	})
}

func createAuthMethods(config types.SM) ([]ssh.AuthMethod, error) {
	auth := make([]ssh.AuthMethod, 0, 2)

	privKeyStr := config["priv_key"]
	if privKeyStr != "" {
		privKey := []byte(privKeyStr)
		s, e := ssh.ParsePrivateKey(privKey)
		if e != nil {
			return nil, e
		}
		auth = append(auth, ssh.PublicKeys(s))
	}

	passwordStr := config["password"]
	if passwordStr != "" {
		auth = append(auth, ssh.Password(passwordStr))
	}
	return auth, nil
}

func NewDrive(_ context.Context, config types.SM, driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	cacheTTL := config.GetDuration("cache_ttl", -1)
	hostKey := config["host_key"]
	rootPath := config["root_path"]
	if rootPath == "" {
		rootPath = "/"
	}
	host := config["host"]
	port := config["port"]
	if port == "" {
		port = "22"
	}

	if !strings.HasPrefix(rootPath, "/") {
		return nil, errors.New(t("invalid_root_path"))
	}

	auth, e := createAuthMethods(config)
	if e != nil {
		return nil, e
	}

	s := &Drive{
		cacheTTL: cacheTTL,
		rootPath: rootPath,
		addr:     fmt.Sprintf("%s:%s", host, port),
		sshConfig: &ssh.ClientConfig{
			User: config["user"],
			Auth: auth,
			HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				if hostKey == "" {
					return nil
				}
				hostKey, _, _, _, e := ssh.ParseAuthorizedKey([]byte(hostKey))
				if e != nil {
					return e
				}
				return ssh.FixedHostKey(hostKey)(hostname, remote, key)
			}),
		},
	}
	if cacheTTL <= 0 {
		s.cache = drive_util.DummyCache()
	} else {
		s.cache = driveUtils.CreateCache(s.deserializeEntry, nil)
	}

	_, e = s.getClient()
	if e != nil {
		return nil, e
	}

	return s, nil
}

func (f *Drive) getClient() (*sftp.Client, error) {
	if f.client != nil {
		return f.client, nil
	}
	f.clientMux.Lock()
	defer f.clientMux.Unlock()
	if f.client != nil {
		return f.client, nil
	}
	sshClient, e := ssh.Dial("tcp", f.addr, f.sshConfig)
	if e != nil {
		return nil, f.handleError(e)
	}
	f.ssh = sshClient
	client, e := sftp.NewClient(f.ssh)
	if e != nil {
		return nil, f.handleError(e)
	}
	f.client = client
	return f.client, nil
}

type Drive struct {
	cache    drive_util.DriveCache
	cacheTTL time.Duration
	rootPath string

	addr      string
	sshConfig *ssh.ClientConfig
	ssh       *ssh.Client
	client    *sftp.Client
	clientMux sync.Mutex
}

func (f *Drive) toRemotePath(path string) string {
	return path2.Join(f.rootPath, path)
}

func (f *Drive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{Writable: true}
}

func (f *Drive) Get(_ context.Context, path string) (types.IEntry, error) {
	if cached, _ := f.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	// Linux 路径格式必须 / 开始
	if utils.IsRootPath(path) {
		return &sftpEntry{d: f, isDir: true, modTime: -1}, nil
	}
	c, e := f.getClient()
	if e != nil {
		return nil, e
	}
	stat, e := c.Stat(f.toRemotePath(path))
	if e != nil {
		return nil, f.handleError(e)
	}
	entry := f.newSFTPEntry("", stat)
	entry.path = path
	_ = f.cache.PutEntry(entry, f.cacheTTL)
	return entry, nil
}

func (f *Drive) Save(ctx types.TaskCtx, path string, size int64, override bool, reader io.Reader) (types.IEntry, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	c, e := f.getClient()
	if e != nil {
		return nil, e
	}
	openMode := os.O_WRONLY | os.O_CREATE
	if override {
		openMode |= os.O_TRUNC
	}
	file, e := c.OpenFile(f.toRemotePath(path), openMode)
	if e != nil {
		return nil, f.handleError(e)
	}
	defer func() { _ = file.Close() }()
	writtenSize, e := file.ReadFrom(reader)
	if e != nil {
		return nil, f.handleError(e)
	}
	if writtenSize != size {
		return nil, errors.New("written size not equal to file size")
	}
	_ = f.cache.Evict(path, false)
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *Drive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	c, e := f.getClient()
	if e != nil {
		return nil, e
	}
	e = c.Mkdir(f.toRemotePath(path))
	if e != nil {
		return nil, f.handleError(e)
	}
	_ = f.cache.Evict(utils.PathParent(path), false)
	return f.Get(ctx, path)
}

func (f *Drive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (f *Drive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
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
	c, e := f.getClient()
	if e != nil {
		return nil, e
	}
	e = c.Rename(f.toRemotePath(fromEntry.path), f.toRemotePath(to))
	if e != nil {
		return nil, f.handleError(e)
	}
	_ = f.cache.Evict(to, true)
	_ = f.cache.Evict(utils.PathParent(to), false)
	_ = f.cache.Evict(fromEntry.path, true)
	_ = f.cache.Evict(utils.PathParent(fromEntry.path), false)
	return f.Get(ctx, to)
}

func (f *Drive) List(_ context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := f.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	c, e := f.getClient()
	if e != nil {
		return nil, e
	}
	stats, e := c.ReadDir(f.toRemotePath(path))
	if e != nil {
		return nil, f.handleError(e)
	}
	entries := make([]types.IEntry, len(stats))
	for i, s := range stats {
		entries[i] = f.newSFTPEntry(path, s)
	}
	_ = f.cache.PutChildren(path, entries, f.cacheTTL)
	return entries, nil
}

func (f *Drive) Delete(ctx types.TaskCtx, path string) error {
	c, e := f.getClient()
	if e != nil {
		return e
	}
	entry, e := f.Get(ctx, path)
	if e != nil {
		return e
	}
	if entry.Type().IsDir() {
		e = c.RemoveDirectory(path)
	} else {
		e = c.Remove(path)
	}
	if e != nil {
		return f.handleError(e)
	}
	_ = f.cache.Evict(utils.PathParent(path), false)
	_ = f.cache.Evict(path, true)
	return nil
}

func (f *Drive) Upload(ctx context.Context, path string, size int64, override bool, _ types.SM) (*types.DriveUploadConfig, error) {
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, f, path); e != nil {
			return nil, e
		}
	}
	return types.UseLocalProvider(size), nil
}

func (f *Drive) newSFTPEntry(parent string, stat os.FileInfo) *sftpEntry {
	return &sftpEntry{
		d:       f,
		path:    path2.Join(parent, stat.Name()),
		size:    stat.Size(),
		isDir:   stat.IsDir(),
		modTime: utils.Millisecond(stat.ModTime()),
	}
}

func (f *Drive) deserializeEntry(dat string) (types.IEntry, error) {
	ec, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	return &sftpEntry{path: ec.Path, d: f, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir()}, nil
}

var connectionLost = err.NewRemoteApiError(500, "Connection lost")

func (f *Drive) handleError(e error) error {
	switch e {
	case sftp.ErrSSHFxEOF:
	case io.EOF:
		return err.NewRemoteApiError(500, "EOF")
	case sftp.ErrSSHFxNoSuchFile:
	case os.ErrNotExist:
		return err.NewNotFoundError()
	case sftp.ErrSSHFxPermissionDenied:
		return err.NewPermissionDeniedError(e.Error())
	case sftp.ErrSSHFxBadMessage:
		return err.NewRemoteApiError(500, "Bad Message")
	case sftp.ErrSSHFxNoConnection:
		return err.NewRemoteApiError(500, "No Connection")
	case sftp.ErrSSHFxConnectionLost:
		f.clientMux.Lock()
		defer f.clientMux.Unlock()
		f.ssh = nil
		f.client = nil
		return connectionLost
	case sftp.ErrSSHFxOpUnsupported:
		return err.NewUnsupportedError()
	}
	return e
}

func (f *Drive) Dispose() error {
	if f.client != nil {
		_ = f.client.Close()
	}
	if f.ssh != nil {
		_ = f.ssh.Close()
	}
	return nil
}

type sftpEntry struct {
	d       *Drive
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
			c, e := f.d.getClient()
			if e != nil {
				_ = r.CloseWithError(e)
				return
			}
			file, e := c.Open(f.d.toRemotePath(f.path))
			if e != nil {
				_ = r.CloseWithError(e)
				return
			}
			defer func() { _ = file.Close() }()
			_, e = file.WriteTo(w)
			if e != nil {
				_ = r.CloseWithError(e)
				return
			}
			_ = w.Close()
		}()
		return r, nil
	}), nil
}

func (f *sftpEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
