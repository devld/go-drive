package storage

import (
	"testing"

	"go-drive/common/types"
)

func TestPathMountDAO_SaveMountsUpdatesExistingRecord(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	if e := db.C().Where("1 = 1").Delete(&types.PathMount{}).Error; e != nil {
		t.Fatalf("clear mounts: %v", e)
	}
	dao := NewPathMountDAO(db, ch)
	parent := "drive/parent"
	if e := dao.SaveMounts([]types.PathMount{{Path: &parent, Name: "mounted", MountAt: "source/one"}}, true); e != nil {
		t.Fatalf("create mount: %v", e)
	}
	if e := dao.SaveMounts([]types.PathMount{{Path: &parent, Name: "mounted", MountAt: "source/two"}}, true); e != nil {
		t.Fatalf("update mount: %v", e)
	}

	var mounts []types.PathMount
	if e := db.C().Find(&mounts).Error; e != nil {
		t.Fatalf("find mounts: %v", e)
	}
	if len(mounts) != 1 {
		t.Fatalf("mount count=%d, want 1", len(mounts))
	}
	if mounts[0].MountAt != "source/two" {
		t.Fatalf("mount target=%q, want source/two", mounts[0].MountAt)
	}
}

func TestPathMountDAO_SaveMountsDoesNotDuplicateExistingPath(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	if e := db.C().Where("1 = 1").Delete(&types.PathMount{}).Error; e != nil {
		t.Fatalf("clear mounts: %v", e)
	}
	dao := NewPathMountDAO(db, ch)
	parent := "drive/parent"
	e := dao.SaveMounts([]types.PathMount{
		{Path: &parent, Name: "mounted", MountAt: "source/one"},
		{Path: &parent, Name: "mounted", MountAt: "source/two"},
	}, false)
	if e != nil {
		t.Fatalf("SaveMounts: %v", e)
	}
	var count int64
	if e := db.C().Model(&types.PathMount{}).Count(&count).Error; e != nil {
		t.Fatalf("count mounts: %v", e)
	}
	if count != 1 {
		t.Fatalf("mount count=%d, want 1", count)
	}
}
