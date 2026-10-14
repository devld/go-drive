package fs_test

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/drive/fs"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFsPermissionAliases(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "privatestorage"), 0755); e != nil {
		t.Fatal(e)
	}
	secret := filepath.Join(root, "privatestorage", "secret.txt")
	if e := os.WriteFile(secret, []byte("secret"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "public.txt"), []byte("public"), 0644); e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	local, e := fs.NewDrive(ctx, types.SM{"path": root}, driveutil.DriveEnv{Config: common.Config{FreeFs: true}})
	if e != nil {
		t.Fatal(e)
	}
	rootPath, restricted := "", "privatestorage"
	permissions := utils.NewPermMap([]types.PathPermission{
		{Path: &rootPath, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		{Path: &restricted, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyReject},
	})
	d := drive.NewPermissionWrapperDrive(local, permissions)
	public, e := d.Get(ctx, "public.txt")
	if e != nil {
		t.Fatal(e)
	}
	reader, e := public.GetReader(ctx, types.FullReaderRange())
	if e != nil {
		t.Fatal(e)
	}
	body, e := io.ReadAll(reader)
	_ = reader.Close()
	if e != nil || string(body) != "public" {
		t.Fatalf("public read = %q, %v", body, e)
	}
	for _, directory := range []string{restricted, "PRIVATESTORAGE", "PRIVAT~1", "privatestorage.", "privatestorage "} {
		t.Run(directory, func(t *testing.T) {
			path := directory + "/secret.txt"
			if _, e := d.Get(ctx, path); !err.IsNotFoundError(e) {
				t.Fatalf("Get restricted path: %v", e)
			}
			if _, e := d.Save(task.DummyContext(), path, 4, true, strings.NewReader("evil")); !err.IsNotFoundError(e) {
				t.Fatalf("Save restricted path: %v", e)
			}
			content, e := os.ReadFile(secret)
			if e != nil || string(content) != "secret" {
				t.Fatalf("secret changed = %q, %v", content, e)
			}
		})
	}
}
