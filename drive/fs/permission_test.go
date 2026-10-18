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

func TestLocalFsRecursiveDeletePermissions(t *testing.T) {
	for _, tc := range []struct {
		name         string
		denyWrite    bool
		allowSibling bool
	}{
		{name: "protected descendant", denyWrite: true},
		{name: "protected descendant with deeper sibling grant", denyWrite: true, allowSibling: true},
		{name: "writable descendants"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			secret := filepath.Join(root, "sub", "locked", "secret.txt")
			public := filepath.Join(root, "sub", "public", "file.txt")
			for _, path := range []string{secret, public} {
				if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(path, []byte("original"), 0644); e != nil {
					t.Fatal(e)
				}
			}
			local, e := fs.NewDrive(context.Background(), types.SM{"path": root}, driveutil.DriveEnv{Config: common.Config{FreeFs: true}})
			if e != nil {
				t.Fatal(e)
			}
			rootPath, locked, sibling := "", "sub/locked", "sub/public/deeper"
			rules := []types.PathPermission{
				{Path: &rootPath, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			}
			if tc.denyWrite {
				rules = append(rules, types.PathPermission{Path: &locked, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject})
			}
			if tc.allowSibling {
				rules = append(rules, types.PathPermission{Path: &sibling, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept})
			}
			d := drive.NewPermissionWrapperDrive(local, utils.NewPermMap(rules))
			if tc.denyWrite {
				if e := d.Delete(task.DummyContext(), locked); !err.IsPermissionDeniedNotFoundError(e) {
					t.Fatalf("direct Delete: %v", e)
				}
			}
			e = d.Delete(task.DummyContext(), "sub")
			if tc.denyWrite {
				if !err.IsNotAllowedError(e) {
					t.Fatalf("recursive Delete: %v", e)
				}
				for _, path := range []string{secret, public} {
					body, e := os.ReadFile(path)
					if e != nil || string(body) != "original" {
						t.Fatalf("Delete changed %s: %q, %v", path, body, e)
					}
				}
			} else {
				if e != nil {
					t.Fatal(e)
				}
				if _, e := os.Stat(filepath.Join(root, "sub")); !os.IsNotExist(e) {
					t.Fatalf("writable directory not deleted: %v", e)
				}
			}
		})
	}
}
