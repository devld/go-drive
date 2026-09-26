package storage

import (
	"encoding/json"
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/secretbox"
	"go-drive/common/types"
	"testing"
)

func TestDriveDAO_AddDrive_duplicateReturnsNotAllowed(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewDriveDAO(db, ch)
	d := types.Drive{Name: "d1", Type: "fs", Enabled: true, Config: "{}"}
	_, _ = dao.AddDrive(d, nil)
	_, e := dao.AddDrive(d, nil)
	if e == nil {
		t.Fatal("expected error when adding duplicate drive")
	}
	var notAllowed err.NotAllowedError
	if !errors.As(e, &notAllowed) {
		t.Errorf("expected NotAllowedError, got %T: %v", e, e)
	}
}

func TestDriveDAO_GetDrive_notFoundReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewDriveDAO(db, ch)
	_, e := dao.GetDrive("nonexistent")
	if e == nil {
		t.Fatal("expected error for nonexistent drive")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}

func TestDriveDAOSealsAndOpensConfig(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewDriveDAO(db, ch)
	form := []types.FormItem{
		{Field: "host", Type: "text"},
		{Field: "password", Type: "password"},
		{Field: "token", Type: "password"},
		{Field: "blank", Type: "password"},
	}
	saved, e := dao.AddDrive(types.Drive{
		Name: "ftp", Type: "ftp", Enabled: true,
		Config: `{"host":"ftp.example","password":"secret","blank":""}`,
	}, form)
	if e != nil {
		t.Fatal(e)
	}
	if !secretbox.IsSealed(mustDriveConfigField(t, saved.Config, "password")) {
		t.Fatal("password was not sealed")
	}
	if mustDriveConfigField(t, saved.Config, "host") != "ftp.example" {
		t.Fatal("host was sealed")
	}
	savedConfig := mustDriveConfig(t, saved.Config)
	if _, ok := savedConfig["token"]; ok {
		t.Fatal("missing token field was written")
	}
	if savedConfig["blank"] != "" {
		t.Fatalf("blank = %q", savedConfig["blank"])
	}

	got, e := dao.GetDrive("ftp")
	if e != nil {
		t.Fatal(e)
	}
	if mustDriveConfigField(t, got.Config, "password") != "secret" {
		t.Fatalf("opened password = %q", mustDriveConfigField(t, got.Config, "password"))
	}

	if e = dao.UpdateDrive("ftp", types.Drive{
		Type: "ftp", Enabled: true,
		Config: `{"host":"ftp.example","password":"next"}`,
	}, form); e != nil {
		t.Fatal(e)
	}
	got, e = dao.GetDrive("ftp")
	if e != nil {
		t.Fatal(e)
	}
	if mustDriveConfigField(t, got.Config, "password") != "next" {
		t.Fatalf("updated password = %q", mustDriveConfigField(t, got.Config, "password"))
	}
}

func mustDriveConfig(t *testing.T, raw string) types.SM {
	t.Helper()
	values := types.SM{}
	if e := json.Unmarshal([]byte(raw), &values); e != nil {
		t.Fatal(e)
	}
	return values
}

func mustDriveConfigField(t *testing.T, raw, field string) string {
	t.Helper()
	return mustDriveConfig(t, raw)[field]
}
