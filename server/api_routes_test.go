package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-drive/common"
	"go-drive/common/utils"

	"github.com/gin-gonic/gin"
)

func TestSignatureAuthBindsQueryPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	signer := utils.NewSigner()
	signature := MakeSignature(signer, "a/b", "", time.Minute)

	tests := []struct {
		name   string
		target string
		called bool
	}{
		{
			name:   "valid canonical path",
			target: "/download?path=a%2F.%2Fb&_k=" + signature,
			called: true,
		},
		{
			name:   "tampered path",
			target: "/download?path=a%2Fc&_k=" + signature,
		},
		{
			name:   "missing path",
			target: "/download?_k=" + signature,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			router := gin.New()
			router.GET("/download", SignatureAuth(signer, nil, true), func(*gin.Context) {
				called = true
			})
			router.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, tt.target, nil),
			)
			if called != tt.called {
				t.Fatalf("handler called = %v, want %v", called, tt.called)
			}
		})
	}
}

func TestCreateChunkUploadAcceptsJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/chunk-uploads",
		strings.NewReader(`{"size":10485760,"chunkSize":5242880}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	dr := driveRoute{chunkUploader: &ChunkUploader{dir: t.TempDir()}}
	dr.createChunkUpload(c)

	if len(c.Errors) != 0 {
		t.Fatalf("createChunkUpload() errors = %v", c.Errors)
	}
	result, ok := GetResult(c)
	if !ok {
		t.Fatal("createChunkUpload() did not set a result")
	}
	upload, ok := result.(ChunkUpload)
	if !ok {
		t.Fatalf("createChunkUpload() result type = %T, want ChunkUpload", result)
	}
	if upload.Size != 10485760 || upload.ChunkSize != 5242880 || upload.Chunks != 2 {
		t.Fatalf("createChunkUpload() result = %+v", upload)
	}
}

func TestCommonRoutesUsePluralTaskResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	if e := InitCommonRoutes(nil, router, nil, nil, nil); e != nil {
		t.Fatalf("InitCommonRoutes() error = %v", e)
	}

	assertRegisteredRoutes(t, router,
		"GET /tasks",
		"GET /tasks/:id",
		"DELETE /tasks/:id",
	)
	assertRoutesNotRegistered(t, router,
		"GET /task/:id",
		"DELETE /task/:id",
	)
}

func TestAdminRoutesUseNormalizedResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	if e := InitAdminRoutes(
		router, nil, common.Config{}, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	); e != nil {
		t.Fatalf("InitAdminRoutes() error = %v", e)
	}

	if got := len(router.Routes()); got != 51 {
		t.Fatalf("registered admin route count = %d, want 51", got)
	}
	assertRegisteredRoutes(t, router,
		"GET /admin/users",
		"POST /admin/users",
		"GET /admin/users/:username",
		"PUT /admin/users/:username",
		"DELETE /admin/users/:username",
		"GET /admin/groups",
		"POST /admin/groups",
		"GET /admin/groups/:name",
		"PUT /admin/groups/:name",
		"DELETE /admin/groups/:name",
		"GET /admin/drives",
		"POST /admin/drives",
		"PUT /admin/drives/:name",
		"DELETE /admin/drives/:name",
		"POST /admin/drives/:name/init-config",
		"POST /admin/drives/:name/init",
		"POST /admin/drives/reload",
		"GET /admin/path-permissions",
		"PUT /admin/path-permissions",
		"GET /admin/path-metadata",
		"PUT /admin/path-metadata",
		"DELETE /admin/path-metadata",
		"PUT /admin/path-mounts",
		"PUT /admin/search-indexes",
		"POST /admin/maintenance/path-rules/cleanup",
		"DELETE /admin/drives/:name/cache",
		"GET /admin/drive-scripts/available",
		"GET /admin/drive-scripts/installed",
		"PUT /admin/drive-scripts/:name",
		"DELETE /admin/drive-scripts/:name",
		"GET /admin/drive-scripts/:name/content",
		"PUT /admin/drive-scripts/:name/content",
		"GET /admin/job-definitions",
		"GET /admin/job-executions",
		"POST /admin/job-executions",
		"POST /admin/job-executions/:id/cancel",
		"DELETE /admin/job-executions/:id",
		"DELETE /admin/job-executions",
		"POST /admin/job-script-evaluations",
		"GET /admin/file-buckets",
		"POST /admin/file-buckets",
		"PUT /admin/file-buckets/:name",
		"DELETE /admin/file-buckets/:name",
	)
	assertRoutesNotRegistered(t, router,
		"GET /admin/user/:username",
		"POST /admin/drive",
		"GET /admin/path-meta",
		"POST /admin/mount/*to",
		"GET /admin/path-permissions/*path",
		"PUT /admin/path-permissions/*path",
		"PUT /admin/path-metadata/*path",
		"DELETE /admin/path-metadata/*path",
		"PUT /admin/path-mounts/*path",
		"PUT /admin/search-indexes/*path",
		"GET /admin/jobs/executions",
		"POST /admin/scripts/install/:name",
		"POST /admin/file-bucket",
	)
}

func TestDriveRoutesUseRPCOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	if e := InitDriveRoutes(
		router, nil, nil, common.Config{}, nil, nil, nil, nil, nil, nil, nil, nil,
	); e != nil {
		t.Fatalf("InitDriveRoutes() error = %v", e)
	}

	if got := len(router.Routes()); got != 19 {
		t.Fatalf("registered drive route count = %d, want 19", got)
	}
	assertRegisteredRoutes(t, router,
		"GET /stat",
		"GET /list",
		"POST /mkdir",
		"POST /copy",
		"POST /move",
		"POST /delete",
		"POST /prepare-upload",
		"POST /write",
		"GET /download",
		"HEAD /download",
		"GET /thumbnail",
		"GET /search",
		"POST /archive",
		"POST /chunk-uploads",
		"PUT /chunk-uploads/:id/chunks/:seq",
		"POST /chunk-uploads/:id/completion",
		"DELETE /chunk-uploads/:id",
	)
	assertRoutesNotRegistered(t, router,
		"GET /entry/*path",
		"DELETE /entry/*path",
		"GET /entries/*path",
		"POST /mkdir/*path",
		"POST /upload/*path",
		"GET /content/*path",
		"PUT /content/*path",
		"GET /thumbnail/*path",
		"GET /search/*path",
		"POST /zip",
		"POST /chunk",
		"PUT /chunk/:id/:seq",
		"POST /chunk-content/*path",
		"DELETE /chunk/:id",
	)
}

func TestGetQueryPathRequiresParameterAndAllowsRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/stat", nil)
		if _, e := getQueryPath(c, "path"); e == nil {
			t.Fatal("getQueryPath() error = nil, want missing parameter error")
		}
	})

	t.Run("root", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/stat?path=", nil)
		path, e := getQueryPath(c, "path")
		if e != nil {
			t.Fatalf("getQueryPath() error = %v", e)
		}
		if path != "" {
			t.Fatalf("getQueryPath() = %q, want root path", path)
		}
	})

	t.Run("clean", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/stat?path=a%2F.%2Fb%2F..%2Fc", nil)
		path, e := getQueryPath(c, "path")
		if e != nil {
			t.Fatalf("getQueryPath() error = %v", e)
		}
		if path != "a/c" {
			t.Fatalf("getQueryPath() = %q, want %q", path, "a/c")
		}
	})
}

func assertRegisteredRoutes(t *testing.T, router *gin.Engine, expected ...string) {
	t.Helper()
	routes := registeredRouteSet(router)
	for _, route := range expected {
		if _, ok := routes[route]; !ok {
			t.Errorf("route %q is not registered", route)
		}
	}
}

func assertRoutesNotRegistered(t *testing.T, router *gin.Engine, unexpected ...string) {
	t.Helper()
	routes := registeredRouteSet(router)
	for _, route := range unexpected {
		if _, ok := routes[route]; ok {
			t.Errorf("legacy route %q is still registered", route)
		}
	}
}

func registeredRouteSet(router *gin.Engine) map[string]struct{} {
	result := make(map[string]struct{})
	for _, route := range router.Routes() {
		result[route.Method+" "+route.Path] = struct{}{}
	}
	return result
}
