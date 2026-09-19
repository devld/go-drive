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
	for _, typ := range []artifact.ArtifactType{
		"thumbnail", "archive-index", "archive-content",
	} {
		_, err := artifacts.Get(context.Background(), artifact.ArtifactRequest{Type: typ}, 0)
		if err == nil || err.Error() == "invalid artifact type" {
			t.Fatalf("artifact type %q was not registered: %v", typ, err)
		}
	}
	if components.Get(registry.KeyThumbnail) == nil {
		t.Fatal("thumbnail component was not initialized")
	}
	if components.Get(registry.KeyArchive) == nil {
		t.Fatal("archive component was not initialized")
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
