package storage

import (
	"go-drive/common/secretbox"
	"go-drive/common/types"
	"testing"
)

func TestDriveDataStoreLoadAllAndClearEmptyValues(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()

	store := NewDriveDataDAO(db, ch).GetDataStore("drive-1")
	if e := store.Save(types.SM{"first": "one", "second": "two"}); e != nil {
		t.Fatal(e)
	}

	got, e := store.Load("first", "second")
	if e != nil {
		t.Fatal(e)
	}
	if got["first"] != "one" || got["second"] != "two" {
		t.Fatalf("Load() = %#v", got)
	}

	if e := store.Save(types.SM{"first": ""}); e != nil {
		t.Fatal(e)
	}
	got, e = store.Load("first", "second")
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := got["first"]; ok {
		t.Fatalf("empty value was not cleared: %#v", got)
	}
	if got["second"] != "two" {
		t.Fatalf("unrelated value changed: %#v", got)
	}
}

func TestDriveDataStoreSaveEncryptedRoundTrip(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()

	store := NewDriveDataDAO(db, ch).GetDataStore("drive-1")
	if e := store.SaveEncrypted(types.SM{"token": "access", "note": ""}); e != nil {
		t.Fatal(e)
	}
	if e := store.Save(types.SM{"drive_id": "root"}); e != nil {
		t.Fatal(e)
	}

	var rows []types.DriveData
	if e := db.C().Where("`drive` = ?", "drive-1").Find(&rows).Error; e != nil {
		t.Fatal(e)
	}
	stored := types.SM{}
	for _, row := range rows {
		stored[row.Key] = row.Value
	}
	if !secretbox.IsSealed(stored["token"]) {
		t.Fatalf("token stored as %q", stored["token"])
	}
	if stored["drive_id"] != "root" {
		t.Fatalf("drive_id = %q", stored["drive_id"])
	}
	if _, ok := stored["note"]; ok {
		t.Fatalf("empty encrypted value was stored: %#v", stored)
	}

	got, e := store.Load("token", "drive_id")
	if e != nil {
		t.Fatal(e)
	}
	if got["token"] != "access" || got["drive_id"] != "root" {
		t.Fatalf("Load() = %#v", got)
	}
}
