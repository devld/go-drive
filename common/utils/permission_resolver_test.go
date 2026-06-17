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

func newPermItem(path, subject string, perm types.Permission, policy uint8) *pathPermItem {
	p := path
	return &pathPermItem{
		PathPermission: types.PathPermission{
			Path:       &p,
			Subject:    subject,
			Permission: perm,
			Policy:     policy,
		},
		depth: int8(PathDepth(path)),
	}
}

// TestResolveAcceptedPermissions covers the merge semantics directly:
// items are processed from highest to lowest priority (depth desc, then
// user > group > anonymous, then reject before accept), and for every
// permission bit the highest-priority matching rule wins. A reject only
// blocks lower-priority accepts; it never revokes a bit already granted by
// a higher-priority accept.
func TestResolveAcceptedPermissions(t *testing.T) {
	grp := types.GroupSubject("g1")
	usr := types.UserSubject("alice")
	tests := []struct {
		name      string
		items     []*pathPermItem
		wantRead  bool
		wantWrite bool
	}{
		{
			name:      "empty",
			items:     nil,
			wantRead:  false,
			wantWrite: false,
		},
		{
			name:      "single accept rw",
			items:     []*pathPermItem{newPermItem("", types.AnySubject, types.PermissionReadWrite, types.PolicyAccept)},
			wantRead:  true,
			wantWrite: true,
		},
		{
			name:      "single reject only grants nothing",
			items:     []*pathPermItem{newPermItem("", types.AnySubject, types.PermissionReadWrite, types.PolicyReject)},
			wantRead:  false,
			wantWrite: false,
		},
		{
			name: "same path same subject reject wins over accept (policy order)",
			items: []*pathPermItem{
				newPermItem("a", types.AnySubject, types.PermissionReadWrite, types.PolicyAccept),
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			},
			wantRead:  true,
			wantWrite: false,
		},
		{
			name: "deeper reject blocks shallower accept",
			items: []*pathPermItem{
				newPermItem("", types.AnySubject, types.PermissionReadWrite, types.PolicyAccept),
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			},
			wantRead:  true,
			wantWrite: false,
		},
		{
			name: "deeper accept is not revoked by shallower reject",
			items: []*pathPermItem{
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyAccept),
				newPermItem("", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			},
			wantRead:  false,
			wantWrite: true,
		},
		{
			name: "user accept beats anonymous reject at same depth",
			items: []*pathPermItem{
				newPermItem("a", usr, types.PermissionWrite, types.PolicyAccept),
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			},
			wantRead:  false,
			wantWrite: true,
		},
		{
			name: "user reject beats anonymous accept at same depth",
			items: []*pathPermItem{
				newPermItem("a", usr, types.PermissionWrite, types.PolicyReject),
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyAccept),
			},
			wantRead:  false,
			wantWrite: false,
		},
		{
			name: "group accept beats anonymous reject at same depth",
			items: []*pathPermItem{
				newPermItem("a", grp, types.PermissionWrite, types.PolicyAccept),
				newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			},
			wantRead:  false,
			wantWrite: true,
		},
		{
			name: "user reject beats group accept at same depth",
			items: []*pathPermItem{
				newPermItem("a", usr, types.PermissionWrite, types.PolicyReject),
				newPermItem("a", grp, types.PermissionWrite, types.PolicyAccept),
			},
			wantRead:  false,
			wantWrite: false,
		},
		{
			name: "bits accumulate from different rules",
			items: []*pathPermItem{
				newPermItem("a", types.AnySubject, types.PermissionRead, types.PolicyAccept),
				newPermItem("", types.AnySubject, types.PermissionWrite, types.PolicyAccept),
			},
			wantRead:  true,
			wantWrite: true,
		},
		{
			name: "reject one bit keeps the other",
			items: []*pathPermItem{
				newPermItem("a", types.AnySubject, types.PermissionReadWrite, types.PolicyAccept),
				newPermItem("a", types.AnySubject, types.PermissionRead, types.PolicyReject),
			},
			wantRead:  false,
			wantWrite: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAcceptedPermissions(tt.items)
			if got.Readable() != tt.wantRead {
				t.Errorf("Readable() = %v, want %v", got.Readable(), tt.wantRead)
			}
			if got.Writable() != tt.wantWrite {
				t.Errorf("Writable() = %v, want %v", got.Writable(), tt.wantWrite)
			}
		})
	}
}

// TestResolveAcceptedPermissions_OrderIndependent ensures the result does not
// depend on the input order, because the function sorts items by priority.
func TestResolveAcceptedPermissions_OrderIndependent(t *testing.T) {
	build := func() []*pathPermItem {
		return []*pathPermItem{
			newPermItem("", types.AnySubject, types.PermissionReadWrite, types.PolicyAccept),
			newPermItem("a", types.AnySubject, types.PermissionWrite, types.PolicyReject),
			newPermItem("a", types.UserSubject("alice"), types.PermissionWrite, types.PolicyAccept),
		}
	}
	forward := resolveAcceptedPermissions(build())

	reversed := build()
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	backward := resolveAcceptedPermissions(reversed)

	if forward != backward {
		t.Errorf("result depends on input order: forward=%v backward=%v", forward, backward)
	}
	// user accept at depth 1 wins over anonymous reject at depth 1 -> writable.
	if !forward.Writable() || !forward.Readable() {
		t.Errorf("want read+write, got %v", forward)
	}
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
