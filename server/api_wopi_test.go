package server

import (
	"bytes"
	"context"
	"encoding/json"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	pathpkg "path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWOPIFileLifecycleAndExternalConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	d := newWOPIMemoryDrive()
	d.set("docs/file.docx", []byte("initial"))
	entry, _ := d.Get(context.Background(), "docs/file.docx")
	service := &wopiService{
		config:       common.Config{APIPath: "/api", TempDir: t.TempDir()},
		access:       wopiStaticAccess{drive: d},
		userDAO:      wopiStaticUsers{},
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(4),
		discovery: &wopiDiscoveryClient{cache: &wopiDiscovery{
			loaded: time.Now(),
			actions: map[string]map[string]wopiDiscoveryAction{
				"docx": {
					"view": {Name: "view", URLSrc: "https://office.test/view"},
					"edit": {Name: "edit", URLSrc: "https://office.test/edit"},
				},
			},
		}},
	}
	session, token, e := service.createSession(
		"alice", "docs/file.docx", "https://drive.test", canonicalWOPIResourceKey(entry), true, time.Time{},
	)
	if e != nil {
		t.Fatal(e)
	}

	route := &wopiRoute{service: service}
	engine := gin.New()
	engine.GET("/wopi/files/:id", route.checkFileInfo)
	engine.GET("/wopi/files/:id/contents", route.getFile)
	engine.POST("/wopi/files/:id", route.fileOperation)
	engine.POST("/wopi/files/:id/contents", route.putFile)
	baseURL := "/wopi/files/" + session.id + "?access_token=" + url.QueryEscape(token)

	response := serveWOPIRequest(engine, http.MethodGet, baseURL, nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("CheckFileInfo status=%d, body=%s", response.Code, response.Body.String())
	}
	var info map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &info); e != nil {
		t.Fatal(e)
	}
	if info["BaseFileName"] != "file.docx" || info["UserCanWrite"] != true || info["SupportsLocks"] != true {
		t.Fatalf("unexpected CheckFileInfo: %#v", info)
	}

	response = serveWOPIRequest(engine, http.MethodGet,
		"/wopi/files/"+session.id+"/contents?access_token="+url.QueryEscape(token), nil, nil)
	if response.Code != http.StatusOK || response.Body.String() != "initial" {
		t.Fatalf("GetFile status=%d, body=%q", response.Code, response.Body.String())
	}

	response = serveWOPIRequest(engine, http.MethodPost, baseURL, nil, map[string]string{
		headerWOPIOverride: "LOCK",
		headerWOPILock:     "lock-1",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("Lock status=%d", response.Code)
	}

	putURL := "/wopi/files/" + session.id + "/contents?access_token=" + url.QueryEscape(token)
	response = serveWOPIRequest(engine, http.MethodPost, putURL, strings.NewReader("updated"), map[string]string{
		headerWOPIOverride: "PUT",
		headerWOPILock:     "lock-1",
		"X-WOPI-Size":      "7",
	})
	if response.Code != http.StatusOK || response.Header().Get(headerWOPIItemVersion) == "" {
		t.Fatalf("PutFile status=%d, headers=%v", response.Code, response.Header())
	}
	var putResult map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &putResult); e != nil {
		t.Fatal(e)
	}
	if putResult["LastModifiedTime"] == "" {
		t.Fatalf("PutFile response missing LastModifiedTime: %#v", putResult)
	}
	if got := string(d.content("docs/file.docx")); got != "updated" {
		t.Fatalf("saved content=%q", got)
	}

	d.onNextGet(func() { d.set("docs/file.docx", []byte("external")) })
	response = serveWOPIRequest(engine, http.MethodPost, putURL, strings.NewReader("overwrite"), map[string]string{
		headerWOPIOverride: "PUT",
		headerWOPILock:     "lock-1",
		"X-WOPI-Size":      "9",
	})
	if response.Code != http.StatusConflict || response.Header().Get(headerWOPILock) != "lock-1" ||
		!strings.Contains(response.Header().Get(headerWOPILockFailureReason), "outside WOPI") {
		t.Fatalf("external conflict status=%d, headers=%v", response.Code, response.Header())
	}
	if got := string(d.content("docs/file.docx")); got != "external" {
		t.Fatalf("external content was overwritten: %q", got)
	}

	response = serveWOPIRequest(engine, http.MethodPost, baseURL, strings.NewReader("copy"), map[string]string{
		headerWOPIOverride:       "PUT_RELATIVE",
		"X-WOPI-SuggestedTarget": ".pdf",
		"X-WOPI-Size":            "4",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("PutRelativeFile status=%d, body=%s", response.Code, response.Body.String())
	}
	if got := string(d.content("docs/file.pdf")); got != "copy" {
		t.Fatalf("relative content=%q", got)
	}
	var relative map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &relative); e != nil {
		t.Fatal(e)
	}
	if relative["Name"] != "file.pdf" || !strings.Contains(relative["Url"].(string), "access_token=") {
		t.Fatalf("unexpected PutRelativeFile response: %#v", relative)
	}

	response = serveWOPIRequest(engine, http.MethodPost, baseURL, nil, map[string]string{
		headerWOPIOverride: "UNLOCK",
		headerWOPILock:     "lock-1",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("Unlock status=%d", response.Code)
	}
}

func TestWOPIRejectsWrongTokenAndSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	d := newWOPIMemoryDrive()
	d.set("file.docx", []byte("x"))
	entry, _ := d.Get(context.Background(), "file.docx")
	service := &wopiService{
		config:       common.Config{TempDir: t.TempDir()},
		access:       wopiStaticAccess{drive: d},
		userDAO:      wopiStaticUsers{},
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(2),
	}
	session, token, _ := service.createSession("alice", "file.docx", "https://drive.test", canonicalWOPIResourceKey(entry), true, time.Time{})
	route := &wopiRoute{service: service}
	engine := gin.New()
	engine.GET("/wopi/files/:id/contents", route.getFile)

	response := serveWOPIRequest(engine, http.MethodGet,
		"/wopi/files/"+session.id+"/contents?access_token=wrong", nil, nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-token status=%d", response.Code)
	}
	response = serveWOPIRequest(engine, http.MethodGet,
		"/wopi/files/"+session.id+"/contents?access_token="+url.QueryEscape(token), nil,
		map[string]string{"X-WOPI-MaxExpectedSize": "0"})
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("max-size status=%d", response.Code)
	}
}

func TestWOPIReadOnlySessionCannotLockOrWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	d := newWOPIMemoryDrive()
	d.set("file.docx", []byte("initial"))
	entry, _ := d.Get(context.Background(), "file.docx")
	service := &wopiService{
		config:       common.Config{TempDir: t.TempDir()},
		access:       wopiStaticAccess{drive: d},
		userDAO:      wopiStaticUsers{},
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(2),
	}
	session, token, _ := service.createSession(
		"alice", "file.docx", "https://drive.test", canonicalWOPIResourceKey(entry), false, time.Time{},
	)
	route := &wopiRoute{service: service}
	engine := gin.New()
	engine.POST("/wopi/files/:id", route.fileOperation)
	engine.POST("/wopi/files/:id/contents", route.putFile)
	baseURL := "/wopi/files/" + session.id + "?access_token=" + url.QueryEscape(token)

	response := serveWOPIRequest(engine, http.MethodPost, baseURL, nil, map[string]string{
		headerWOPIOverride: "LOCK",
		headerWOPILock:     "lock-1",
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("read-only Lock status=%d", response.Code)
	}
	response = serveWOPIRequest(engine, http.MethodPost,
		"/wopi/files/"+session.id+"/contents?access_token="+url.QueryEscape(token),
		strings.NewReader("updated"), nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("read-only PutFile status=%d", response.Code)
	}
}

func TestWOPICreateSessionUsesCurrentOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	d := newWOPIMemoryDrive()
	d.set("docs/file.docx", []byte("initial"))
	service := &wopiService{
		config:       common.Config{APIPath: "/api"},
		access:       wopiStaticAccess{drive: d},
		userDAO:      wopiStaticUsers{},
		sessions:     make(map[string]wopiSession),
		locks:        make(map[string]wopiLock),
		resourceLock: utils.NewKeyLock(2),
		discovery: &wopiDiscoveryClient{cache: &wopiDiscovery{
			loaded: time.Now(),
			actions: map[string]map[string]wopiDiscoveryAction{
				"docx": {"edit": {Name: "edit", URLSrc: "https://office.test/edit?WOPISrc=WOPI_SOURCE"}},
			},
		}},
	}
	route := &wopiRoute{service: service}
	engine := gin.New()
	engine.POST("/wopi/session/*path", func(c *gin.Context) {
		SetPrincipal(c, types.Principal{User: types.User{Username: "alice"}, AuthType: types.AuthTypeToken})
		c.Next()
	}, route.createSession)

	for _, host := range []string{"a.example", "b.example"} {
		req := httptest.NewRequest(http.MethodPost, "/wopi/session/docs/file.docx", nil)
		req.Host = host
		req.Header.Set("Origin", "https://"+host)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("host %s: status=%d, body=%s", host, response.Code, response.Body.String())
		}
		var result wopiSessionResponse
		if e := json.Unmarshal(response.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		actionURL, e := url.Parse(result.ActionURL)
		if e != nil {
			t.Fatal(e)
		}
		wopiSrc := actionURL.Query().Get("WOPISrc")
		if !strings.HasPrefix(wopiSrc, "https://"+host+"/api/wopi/files/") {
			t.Fatalf("host %s: WOPISrc=%q", host, wopiSrc)
		}
		if strings.Contains(result.ActionURL, result.AccessToken) {
			t.Fatal("access token leaked into the action URL")
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/wopi/session/docs/file.docx", nil)
	req.Host = "a.example"
	req.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("mismatched origin status=%d", response.Code)
	}
}

func serveWOPIRequest(handler http.Handler, method, target string, body io.Reader,
	headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

type wopiStaticAccess struct{ drive types.IDrive }

func (a wopiStaticAccess) GetDrive(types.Principal) (types.IDrive, error) { return a.drive, nil }

type wopiStaticUsers struct{}

func (wopiStaticUsers) GetUser(username string) (types.User, error) {
	return types.User{Username: username}, nil
}

type wopiMemoryDrive struct {
	mu      sync.Mutex
	files   map[string][]byte
	version int64
	getHook func()
}

func newWOPIMemoryDrive() *wopiMemoryDrive {
	return &wopiMemoryDrive{files: make(map[string][]byte)}
}

func (d *wopiMemoryDrive) set(path string, content []byte) {
	d.mu.Lock()
	d.version++
	d.files[path] = bytes.Clone(content)
	d.mu.Unlock()
}

func (d *wopiMemoryDrive) content(path string) []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return bytes.Clone(d.files[path])
}

func (d *wopiMemoryDrive) onNextGet(hook func()) {
	d.mu.Lock()
	d.getHook = hook
	d.mu.Unlock()
}

func (d *wopiMemoryDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{Writable: true}, nil
}

func (d *wopiMemoryDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	d.mu.Lock()
	content, ok := d.files[path]
	if !ok {
		d.mu.Unlock()
		return nil, err.NewNotFoundError()
	}
	entry := &wopiMemoryEntry{drive: d, path: path, content: bytes.Clone(content), modTime: d.version}
	hook := d.getHook
	d.getHook = nil
	d.mu.Unlock()
	if hook != nil {
		hook()
	}
	return entry, nil
}

func (d *wopiMemoryDrive) Save(_ types.TaskCtx, path string, size int64, override bool,
	reader io.Reader) (types.IEntry, error) {
	content, e := io.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	if int64(len(content)) != size {
		return nil, err.NewBadRequestError("size mismatch")
	}
	d.mu.Lock()
	if _, exists := d.files[path]; exists && !override {
		d.mu.Unlock()
		return nil, err.NewNotAllowedError()
	}
	d.version++
	d.files[path] = bytes.Clone(content)
	version := d.version
	d.mu.Unlock()
	return &wopiMemoryEntry{drive: d, path: path, content: bytes.Clone(content), modTime: version}, nil
}

func (d *wopiMemoryDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	panic("not used")
}
func (d *wopiMemoryDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *wopiMemoryDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *wopiMemoryDrive) List(context.Context, string) ([]types.IEntry, error) {
	panic("not used")
}
func (d *wopiMemoryDrive) Delete(types.TaskCtx, string) error { panic("not used") }
func (d *wopiMemoryDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	panic("not used")
}

type wopiMemoryEntry struct {
	drive   *wopiMemoryDrive
	path    string
	content []byte
	modTime int64
}

func (e *wopiMemoryEntry) Path() string          { return e.path }
func (e *wopiMemoryEntry) Name() string          { return pathpkg.Base(e.path) }
func (e *wopiMemoryEntry) Type() types.EntryType { return types.TypeFile }
func (e *wopiMemoryEntry) Size() int64           { return int64(len(e.content)) }
func (e *wopiMemoryEntry) Meta() types.EntryMeta {
	return types.EntryMeta{Readable: true, Writable: true}
}
func (e *wopiMemoryEntry) ModTime() int64      { return e.modTime }
func (e *wopiMemoryEntry) Drive() types.IDrive { return e.drive }
func (e *wopiMemoryEntry) GetReader(context.Context, int64, int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(e.content)), nil
}
func (e *wopiMemoryEntry) GetURL(context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
