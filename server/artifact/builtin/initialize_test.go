package builtin

import (
	"context"
	"go-drive/common"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"testing"
)

func TestInitializeRegistersBuiltInArtifactProcessors(t *testing.T) {
	components := registry.NewComponentHolder()
	config := common.Config{
		TempDir:           t.TempDir(),
		MaxConcurrentTask: 2,
		Thumbnail: common.ThumbnailConfig{
			Concurrent: 1,
		},
	}
	runner := task.NewPondRunner(config, components)
	t.Cleanup(func() { _ = components.Dispose() })

	artifacts, err := Initialize(config, runner, components)
	if err != nil {
		t.Fatal(err)
	}
	for _, handler := range []string{"thumbnail", "archive"} {
		_, err := artifacts.Fetch(context.Background(), artifact.Request{Handler: handler}, 0)
		if err == nil || err.Error() == "unknown artifact handler" {
			t.Fatalf("artifact handler %q was not registered: %v", handler, err)
		}
	}
	if components.Get(registry.KeyArtifact) != artifacts {
		t.Fatal("artifact service was not registered")
	}
	name, configValues, err := artifacts.SysConfig()
	if err != nil || name != "artifact" {
		t.Fatalf("artifact SysConfig() = %q %#v err=%v", name, configValues, err)
	}
	archiveConfig, ok := configValues["archive"].(types.M)
	if !ok || archiveConfig["extensions"] != "zip,7z,rar" {
		t.Fatalf("archive config = %#v", configValues["archive"])
	}
}
