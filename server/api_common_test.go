package server

import (
	"go-drive/common/registry"
	"go-drive/common/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type artifactConfigTestComponent struct{}

func (artifactConfigTestComponent) SysConfig() (string, types.M, error) {
	return "artifact", types.M{
		"thumbnail": types.M{"extensions": "jpg,png"},
		"archive":   types.M{"extensions": "zip,7z,rar", "maxSize": int64(2)},
	}, nil
}

func TestGetConfigUsesUnifiedArtifactNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	components := registry.NewComponentHolder()
	components.Add(artifactConfigTestComponent{})

	writer := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(writer)
	context.Request = httptest.NewRequest(http.MethodGet, "/config", nil)

	(&commonRoute{ch: components}).getConfig(context)
	result, ok := GetResult(context)
	if !ok {
		t.Fatal("getConfig() did not set a result")
	}
	config, ok := result.(types.M)
	if !ok {
		t.Fatalf("getConfig() result type = %T, want types.M", result)
	}
	if _, ok := config["thumbnail"]; ok {
		t.Fatal("getConfig() returned the legacy top-level thumbnail config")
	}
	if _, ok := config["archive"]; ok {
		t.Fatal("getConfig() returned the legacy top-level archive config")
	}
	artifactConfig, ok := config["artifact"].(types.M)
	if !ok {
		t.Fatalf("artifact config = %#v", config["artifact"])
	}
	if artifactConfig["archive"].(types.M)["extensions"] != "zip,7z,rar" {
		t.Fatalf("archive extensions = %#v", artifactConfig["archive"])
	}
}
