package builtin

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/server/artifact"
	"io"
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
	runner := task.NewTaskRunner(config)
	t.Cleanup(func() { _ = components.Dispose() })

	files, err := driveutil.NewDriveFS(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Dispose() })
	artifacts, err := Initialize(config, nil, runner, files)
	if err != nil {
		t.Fatal(err)
	}
	components.Add(artifacts)
	source := &initTestEntry{}
	for _, handler := range []string{"thumbnail", "archive"} {
		_, err := artifacts.Fetch(context.Background(), artifact.Request{Handler: handler, Source: source}, 0)
		if err == nil || err.Error() == "unknown artifact handler" {
			t.Fatalf("artifact handler %q was not registered: %v", handler, err)
		}
	}
	found := registry.Gets[*artifact.Service](components)
	if len(found) != 1 || found[0] != artifacts {
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

type initTestEntry struct{}

func (initTestEntry) Path() string                               { return "file" }
func (initTestEntry) Name() string                               { return "file" }
func (initTestEntry) Type() types.EntryType                      { return types.TypeFile }
func (initTestEntry) Size() int64                                { return 1 }
func (initTestEntry) ModTime() int64                             { return 1 }
func (initTestEntry) Meta() types.EntryMeta                      { return types.EntryMeta{Readable: true} }
func (initTestEntry) Drive() types.IDrive                        { return nil }
func (initTestEntry) GetDispatchedDrive() (string, types.IDrive) { return "drive", nil }
func (initTestEntry) GetRealPath() string                        { return "drive/file" }
func (initTestEntry) GetReader(context.Context, types.ReaderRange) (io.ReadCloser, error) {
	return nil, nil
}
func (initTestEntry) GetURL(context.Context) (*types.ContentURL, error) { return nil, nil }
