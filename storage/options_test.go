package storage

import (
	"testing"
)

func TestOptionsDAO_GetOrDefault_missingKeyReturnsDefault(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewOptionsDAO(db, ch)
	got, e := dao.GetOrDefault("nonexistent_key", "defaultVal")
	if e != nil {
		t.Fatalf("GetOrDefault: %v", e)
	}
	if got != "defaultVal" {
		t.Errorf("GetOrDefault(missing, defaultVal) = %q, want defaultVal", got)
	}
}
