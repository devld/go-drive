package mega

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/req"
	"go-drive/common/types"

	megaapi "go-drive/drive/mega/internal/gomega"
)

const transferTimeout = 5 * time.Minute

// node is a decrypted MEGA filesystem node addressed by its handle.
type node struct {
	id      string
	name    string
	isDir   bool
	size    int64
	modTime time.Time
}

// downloader reads a MEGA file as encrypted chunks that are decrypted locally.
type downloader interface {
	chunkCount() int
	chunkAt(index int) (position int64, size int, err error)
	readChunk(index int) ([]byte, error)
	finish() error
}

// client is the filesystem subset of the MEGA API used by Drive.
type client interface {
	root() (node, error)
	children(parentID string) ([]node, error)
	mkdir(parentID, name string) (node, error)
	put(ctx context.Context, parentID, name string, size int64, reader io.Reader, progress func(int64)) (node, error)
	open(id string) (downloader, error)
	move(id, parentID string) error
	rename(id, name string) error
	remove(id string, permanent bool) error
	close() error
}

type megaClient struct {
	api *megaapi.Mega
}

func accountCredentials(config types.SM) (string, string, error) {
	email := strings.TrimSpace(config["email"])
	password := config["password"]
	if email == "" || password == "" {
		return "", "", err.NewBadRequestError(t("missing_credentials"))
	}
	return email, password, nil
}

func prepareAPI(config types.SM) *megaapi.Mega {
	api := megaapi.New()
	api.SetHTTPS(config.GetBool("https"))
	api.SetLogger(func(format string, args ...any) {
		logging.For("mega").Infof(format, args...)
	})
	api.SetDebugger(nil)
	api.SetClient(req.NewLoggingClient(&http.Client{
		Timeout:   transferTimeout,
		Transport: userAgentTransport{base: http.DefaultTransport},
	}))
	return api
}

func connect(config types.SM, data driveutil.DriveDataStore) (*megaClient, error) {
	email, password, credErr := accountCredentials(config)
	if credErr != nil {
		return nil, credErr
	}

	api := prepareAPI(config)
	client := &megaClient{api: api}
	if loginErr := restoreOrLogin(api, email, password, "", data); loginErr != nil {
		api.Close()
		return nil, loginErr
	}
	return client, nil
}

// userAgentTransport supplies the User-Agent expected by the MEGA API when the
// upstream client leaves it unset.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	if request.Header.Get("User-Agent") == "" {
		request.Header.Set("User-Agent", "go-drive")
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

func (m *megaClient) root() (node, error) {
	root := m.api.FS.GetRoot()
	converted, ok := convertNode(root)
	if !ok {
		return node{}, megaapi.ENOENT
	}
	return converted, nil
}

func (m *megaClient) children(parentID string) ([]node, error) {
	parent := m.api.FS.HashLookup(parentID)
	if parent == nil {
		return nil, megaapi.ENOENT
	}
	raw, childrenErr := m.api.FS.GetChildren(parent)
	if childrenErr != nil {
		return nil, childrenErr
	}
	nodes := make([]node, 0, len(raw))
	for _, child := range raw {
		converted, ok := convertNode(child)
		if ok && (child.GetType() == megaapi.FILE || child.GetType() == megaapi.FOLDER) {
			nodes = append(nodes, converted)
		}
	}
	return nodes, nil
}

func (m *megaClient) mkdir(parentID, name string) (node, error) {
	parent := m.api.FS.HashLookup(parentID)
	if parent == nil {
		return node{}, megaapi.ENOENT
	}
	created, mkdirErr := m.api.CreateDir(name, parent)
	if mkdirErr != nil {
		return node{}, mkdirErr
	}
	converted, ok := convertNode(created)
	if !ok {
		return node{}, megaapi.EINTERNAL
	}
	return converted, nil
}

func (m *megaClient) put(ctx context.Context, parentID, name string, size int64, reader io.Reader, progress func(int64)) (node, error) {
	parent := m.api.FS.HashLookup(parentID)
	if parent == nil {
		return node{}, megaapi.ENOENT
	}
	upload, uploadErr := m.api.NewUpload(parent, name, size)
	if uploadErr != nil {
		return node{}, uploadErr
	}
	for index := 0; index < upload.Chunks(); index++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return node{}, ctxErr
		}
		_, chunkSize, locationErr := upload.ChunkLocation(index)
		if locationErr != nil {
			return node{}, locationErr
		}
		chunk := make([]byte, chunkSize)
		if chunkSize > 0 {
			if _, readErr := io.ReadFull(reader, chunk); readErr != nil {
				return node{}, readErr
			}
		}
		if chunkErr := upload.UploadChunk(index, chunk); chunkErr != nil {
			return node{}, chunkErr
		}
		if progress != nil {
			progress(int64(chunkSize))
		}
	}
	created, finishErr := upload.Finish()
	if finishErr != nil {
		return node{}, finishErr
	}
	converted, ok := convertNode(created)
	if !ok {
		return node{}, megaapi.EINTERNAL
	}
	return converted, nil
}

func (m *megaClient) open(id string) (downloader, error) {
	src := m.api.FS.HashLookup(id)
	if src == nil {
		return nil, megaapi.ENOENT
	}
	download, downloadErr := m.api.NewDownload(src)
	if downloadErr != nil {
		return nil, downloadErr
	}
	return &megaDownload{download: download}, nil
}

func (m *megaClient) move(id, parentID string) error {
	src := m.api.FS.HashLookup(id)
	parent := m.api.FS.HashLookup(parentID)
	if src == nil || parent == nil {
		return megaapi.ENOENT
	}
	return m.api.Move(src, parent)
}

func (m *megaClient) rename(id, name string) error {
	src := m.api.FS.HashLookup(id)
	if src == nil {
		return megaapi.ENOENT
	}
	return m.api.Rename(src, name)
}

func (m *megaClient) close() error {
	if m.api != nil {
		m.api.Close()
	}
	return nil
}

func (m *megaClient) remove(id string, permanent bool) error {
	src := m.api.FS.HashLookup(id)
	if src == nil {
		return megaapi.ENOENT
	}
	// permanent uses MEGA's destroy flag. Otherwise the node moves to the rubbish bin,
	// which this drive does not mount.
	return m.api.Delete(src, permanent)
}

func convertNode(src *megaapi.Node) (node, bool) {
	if src == nil {
		return node{}, false
	}
	kind := src.GetType()
	if kind != megaapi.FILE && kind != megaapi.FOLDER && kind != megaapi.ROOT {
		return node{}, false
	}
	return node{
		id:      src.GetHash(),
		name:    src.GetName(),
		isDir:   kind != megaapi.FILE,
		size:    src.GetSize(),
		modTime: src.GetTimeStamp(),
	}, true
}

type megaDownload struct {
	download *megaapi.Download
}

func (d *megaDownload) chunkCount() int {
	return d.download.Chunks()
}

func (d *megaDownload) chunkAt(index int) (int64, int, error) {
	return d.download.ChunkLocation(index)
}

func (d *megaDownload) readChunk(index int) ([]byte, error) {
	return d.download.DownloadChunk(index)
}

func (d *megaDownload) finish() error {
	return d.download.Finish()
}
