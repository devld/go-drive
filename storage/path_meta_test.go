package storage

import (
	"testing"
)

func TestPathMetaDAO_Get_missingReturnsNilNil(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewPathMetaDAO(db, ch)
	pm, e := dao.Get("/nonexistent/path")
	if e != nil {
		t.Fatalf("Get: %v", e)
	}
	if pm != nil {
		t.Errorf("Get missing path should return nil meta, got %v", pm)
	}
}
