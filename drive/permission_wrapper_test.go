package drive_test

import (
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"testing"
)

func TestPermissionWrapperRecursiveOperations(t *testing.T) {
	operations := []struct {
		name           string
		restrictedRoot string
		require        types.Permission
		run            func(*drive.PermissionWrapperDrive) error
	}{
		{"delete", "source", types.PermissionReadWrite, func(d *drive.PermissionWrapperDrive) error {
			return d.Delete(task.DummyContext(), "source")
		}},
		{"move source", "source", types.PermissionReadWrite, func(d *drive.PermissionWrapperDrive) error {
			_, e := d.Move(task.DummyContext(), permissionOperationEntry{path: "source"}, "target", true)
			return e
		}},
		{"move target", "target", types.PermissionReadWrite, func(d *drive.PermissionWrapperDrive) error {
			_, e := d.Move(task.DummyContext(), permissionOperationEntry{path: "source"}, "target", true)
			return e
		}},
		{"copy source", "source", types.PermissionRead, func(d *drive.PermissionWrapperDrive) error {
			_, e := d.Copy(task.DummyContext(), permissionOperationEntry{path: "source"}, "target", true)
			return e
		}},
		{"copy target", "target", types.PermissionReadWrite, func(d *drive.PermissionWrapperDrive) error {
			_, e := d.Copy(task.DummyContext(), permissionOperationEntry{path: "source"}, "target", true)
			return e
		}},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			for _, scenario := range []struct {
				name         string
				deny         types.Permission
				siblingGrant bool
				userOverride bool
			}{
				{name: "unrestricted"},
				{name: "restricted child", deny: op.require},
				{name: "deeper sibling grant", deny: op.require, siblingGrant: true},
				{name: "same path user override", deny: op.require, userOverride: true},
				{name: "read only child", deny: types.PermissionWrite},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					root, locked, sibling := "", op.restrictedRoot+"/locked", op.restrictedRoot+"/public/deeper"
					group := types.GroupSubject("users")
					rules := []types.PathPermission{
						{Path: &root, Subject: group, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept},
					}
					if scenario.deny != 0 {
						rules = append(rules, types.PathPermission{Path: &locked, Subject: group, Permission: scenario.deny, Policy: types.PolicyReject})
					}
					if scenario.siblingGrant {
						rules = append(rules, types.PathPermission{Path: &sibling, Subject: group, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept})
					}
					if scenario.userOverride {
						rules = append(rules, types.PathPermission{Path: &locked, Subject: types.UserSubject("alice"), Permission: scenario.deny, Policy: types.PolicyAccept})
					}
					pm := utils.NewPermMap(rules).Filter(types.Principal{User: types.User{
						Username: "alice", Groups: []types.Group{{Name: "users"}},
					}})
					backend := &permissionOperationDrive{}
					e := op.run(drive.NewPermissionWrapperDrive(backend, pm))
					wantAllowed := scenario.deny&op.require == 0 || scenario.userOverride
					if wantAllowed {
						if e != nil || !backend.called {
							t.Fatalf("allowed operation: error=%v backend called=%v", e, backend.called)
						}
					} else if !err.IsNotAllowedError(e) || backend.called {
						t.Fatalf("denied operation: error=%v backend called=%v", e, backend.called)
					}
				})
			}
		})
	}
}

// Only destructive operations are implemented so an unexpected call fails the test.
type permissionOperationDrive struct {
	types.IDrive
	called bool
}

func (d *permissionOperationDrive) Delete(types.TaskCtx, string) error {
	d.called = true
	return nil
}

func (d *permissionOperationDrive) Move(_ types.TaskCtx, _ types.IEntry, to string, _ bool) (types.IEntry, error) {
	d.called = true
	return permissionOperationEntry{path: to}, nil
}

func (d *permissionOperationDrive) Copy(_ types.TaskCtx, _ types.IEntry, to string, _ bool) (types.IEntry, error) {
	d.called = true
	return permissionOperationEntry{path: to}, nil
}

type permissionOperationEntry struct {
	types.IEntry
	path string
}

func (e permissionOperationEntry) Path() string { return e.path }
