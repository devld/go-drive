package storage

import (
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
