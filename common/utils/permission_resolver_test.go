package utils

import (
	"go-drive/common/types"
	"testing"
)

func ptr(s string) *string { return &s }

func TestNewPermMap(t *testing.T) {
	root := ""
	perms := []types.PathPermission{
		{ID: 1, Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
		{ID: 2, Path: ptr("a"), Subject: types.UserSubject("alice"), Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
	}
	pm := NewPermMap(perms)
	if pm == nil {
		t.Fatal("NewPermMap returned nil")
	}
	if len(pm) != 2 {
		t.Errorf("PermMap len: want 2, got %d", len(pm))
	}
	// ResolvePath behavior is covered in TestPermMap_ResolvePath below.
	p := pm.ResolvePath("")
	if !p.Readable() {
		t.Errorf("root: want readable, got %v", p)
	}
	p2 := pm.ResolvePath("a")
	if !p2.Writable() {
		t.Errorf("a: want writable, got %v", p2)
	}
}

func TestPermMap_Filter(t *testing.T) {
	root := ""
	perms := []types.PathPermission{
		{ID: 1, Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
		{ID: 2, Path: ptr("private"), Subject: types.UserSubject("alice"), Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
	}
	pm := NewPermMap(perms)

	// Anonymous session: only ANY subject (root read); private inherits read but not write.
	sessionAnon := types.NewSession()
	filtered := pm.Filter(sessionAnon)
	if len(filtered) != 1 {
		t.Errorf("anonymous Filter: want 1 subject, got %d", len(filtered))
	}
	p := filtered.ResolvePath("private")
	if !p.Readable() {
		t.Errorf("anonymous on private: inherit read from root, got readable=%v", p.Readable())
	}
	if p.Writable() {
		t.Errorf("anonymous on private: want no write (no u:alice), got writable=%v", p.Writable())
	}

	// Logged-in user alice: should have ANY + u:alice.
	sessionAlice := types.Session{
		User:  types.User{Username: "alice", Groups: nil},
		Props: types.SM{},
	}
	filtered = pm.Filter(sessionAlice)
	if len(filtered) != 2 {
		t.Errorf("alice Filter: want 2 subjects, got %d", len(filtered))
	}
	p = filtered.ResolvePath("private")
	if !p.Readable() || !p.Writable() {
		t.Errorf("alice on private: want rw, got %v", p)
	}

	// Admin user: should get privilegedPermMap (root read+write).
	sessionAdmin := types.Session{
		User:  types.User{Username: "admin", Groups: []types.Group{{Name: types.AdminUserGroup}}},
		Props: types.SM{},
	}
	filtered = pm.Filter(sessionAdmin)
	// privilegedPermMap grants root read+write.
	rootPerm := filtered.ResolvePath("")
	if !rootPerm.Readable() || !rootPerm.Writable() {
		t.Errorf("admin root: want read+write, got %v", rootPerm)
	}
	anyPath := filtered.ResolvePath("any/path")
	if !anyPath.Readable() || !anyPath.Writable() {
		t.Errorf("admin any path: want read+write, got %v", anyPath)
	}
}

func TestPermMap_ResolvePath(t *testing.T) {
	root := ""
	tests := []struct {
		name        string
		permissions []types.PathPermission
		path        string
		wantRead    bool
		wantWrite   bool
	}{
		{
			name: "root accept read",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
			},
			path:      "",
			wantRead:  true,
			wantWrite: false,
		},
		{
			name: "root accept read_write",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			},
			path:      "",
			wantRead:  true,
			wantWrite: true,
		},
		{
			name: "subpath inherits from root",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			},
			path:      "a/b/c",
			wantRead:  true,
			wantWrite: true,
		},
		{
			name: "subpath override reject write",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
				{Path: ptr("a"), Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
			},
			path:      "a",
			wantRead:  true,
			wantWrite: false,
		},
		{
			name: "deeper path still rejected write",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
				{Path: ptr("a"), Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
			},
			path:      "a/b",
			wantRead:  true,
			wantWrite: false,
		},
		{
			name: "user overrides anonymous",
			permissions: []types.PathPermission{
				{Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
				{Path: ptr("private"), Subject: types.UserSubject("alice"), Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			},
			path:      "private",
			wantRead:  true,
			wantWrite: true,
		},
		{
			name: "no permission for path",
			permissions: []types.PathPermission{
				{Path: ptr("a"), Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
			},
			path:      "b",
			wantRead:  false,
			wantWrite: false,
		},
		{
			name:        "empty PermMap",
			permissions: nil,
			path:        "",
			wantRead:    false,
			wantWrite:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pm PermMap
			if tt.permissions != nil {
				pm = NewPermMap(tt.permissions)
			} else {
				pm = make(PermMap)
			}
			got := pm.ResolvePath(tt.path)
			if read := got.Readable(); read != tt.wantRead {
				t.Errorf("ResolvePath(%q).Readable() = %v, want %v", tt.path, read, tt.wantRead)
			}
			if write := got.Writable(); write != tt.wantWrite {
				t.Errorf("ResolvePath(%q).Writable() = %v, want %v", tt.path, write, tt.wantWrite)
			}
		})
	}
}

func TestPermMap_ResolveDescendant(t *testing.T) {
	root := ""
	t.Run("no descendants", func(t *testing.T) {
		perms := []types.PathPermission{
			{Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
		}
		pm := NewPermMap(perms)
		_, defined := pm.ResolveDescendant("a/b")
		if defined {
			t.Error("ResolveDescendant(a/b): want defined=false when no descendants")
		}
	})

	t.Run("descendants inherit and can restrict", func(t *testing.T) {
		perms := []types.PathPermission{
			{Path: &root, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			{Path: ptr("folder"), Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
			{Path: ptr("folder/secret"), Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject},
		}
		pm := NewPermMap(perms)
		perm, defined := pm.ResolveDescendant("folder")
		if !defined {
			t.Fatal("ResolveDescendant(folder): want defined=true")
		}
		if !perm.Readable() {
			t.Errorf("folder descendant: want readable, got %v", perm)
		}
		if perm.Writable() {
			t.Errorf("folder descendant: want not writable (secret rejects write), got writable")
		}
	})

	t.Run("root descendant is all paths", func(t *testing.T) {
		perms := []types.PathPermission{
			{Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
			{Path: ptr("a"), Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
		}
		pm := NewPermMap(perms)
		perm, defined := pm.ResolveDescendant("")
		if !defined {
			t.Fatal("ResolveDescendant(): want defined=true")
		}
		// Merges all descendants: at least read, and a has write.
		if !perm.Readable() {
			t.Errorf("root descendant: want readable, got %v", perm)
		}
		if !perm.Writable() {
			t.Errorf("root descendant: want writable from a, got %v", perm)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		pm := make(PermMap)
		_, defined := pm.ResolveDescendant("x")
		if defined {
			t.Error("empty PermMap ResolveDescendant: want defined=false")
		}
	})
}

// Same usage as permission_wrapper: Filter(session) first to get current user's PermMap, then ResolvePath/ResolveDescendant.
func TestPermMap_FilterThenResolve(t *testing.T) {
	root := ""
	perms := []types.PathPermission{
		{Path: &root, Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
		{Path: ptr("driveA"), Subject: types.AnySubject, Permission: types.PermissionRead, Policy: types.PolicyAccept},
		{Path: ptr("driveA/docs"), Subject: types.UserSubject("alice"), Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
	}
	pm := NewPermMap(perms)

	// Anonymous: can only read root and driveA, cannot write driveA/docs (no alice subject).
	sessionAnon := types.NewSession()
	f := pm.Filter(sessionAnon)
	if f.ResolvePath("").Readable() != true || f.ResolvePath("driveA").Readable() != true {
		t.Errorf("anonymous: root and driveA should be readable")
	}
	if f.ResolvePath("driveA/docs").Writable() {
		t.Errorf("anonymous: driveA/docs should not be writable")
	}

	// alice: driveA/docs is read-write.
	sessionAlice := types.Session{User: types.User{Username: "alice", Groups: nil}, Props: types.SM{}}
	f = pm.Filter(sessionAlice)
	if !f.ResolvePath("driveA/docs").Writable() {
		t.Errorf("alice: driveA/docs should be writable")
	}

	// requireDescendant semantics: path itself and all descendants must satisfy the permission.
	dp, defined := f.ResolveDescendant("driveA")
	if !defined {
		t.Fatal("driveA should have descendants")
	}
	// docs has write, so merged descendant should have write.
	if !dp.Writable() {
		t.Errorf("alice ResolveDescendant(driveA): want writable, got %v", dp)
	}
}
