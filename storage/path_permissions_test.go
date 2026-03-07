package storage

import (
	"go-drive/common/types"
	"testing"
)

func TestPathPermissionDAO_SavePathPermissions_replacesByPath(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewPathPermissionDAO(db, ch)
	path := "test/path"
	p1 := types.PathPermission{Subject: "u:alice", Permission: types.PermissionRead, Policy: types.PolicyAccept}
	if e := dao.SavePathPermissions(path, []types.PathPermission{p1}); e != nil {
		t.Fatalf("first Save: %v", e)
	}
	p2 := types.PathPermission{Subject: "u:bob", Permission: types.PermissionReadWrite, Policy: types.PolicyAccept}
	if e := dao.SavePathPermissions(path, []types.PathPermission{p2}); e != nil {
		t.Fatalf("second Save: %v", e)
	}
	pps, e := dao.GetByPath(path)
	if e != nil {
		t.Fatalf("GetByPath: %v", e)
	}
	if len(pps) != 1 || pps[0].Subject != "u:bob" {
		t.Errorf("SavePathPermissions should replace: got %v", pps)
	}
}
