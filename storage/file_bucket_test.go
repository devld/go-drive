package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"testing"
)

// GetBucket uses cache only; with a fresh DAO and no buckets in DB, any name returns NotFound.
func TestFileBucketDAO_GetBucket_notInCacheReturnsNotFound(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewFileBucketDAO(db, ch)
	_, e := dao.GetBucket("nonexistent_bucket")
	if e == nil {
		t.Fatal("expected error for bucket not in cache")
	}
	var notFound err.NotFoundError
	if !errors.As(e, &notFound) {
		t.Errorf("expected NotFoundError, got %T: %v", e, e)
	}
}

func TestFileBucketDAO_AddBucket_duplicateReturnsNotAllowed(t *testing.T) {
	db, ch, cleanup := newTestDB(t)
	defer cleanup()
	dao := NewFileBucketDAO(db, ch)
	b := types.FileBucket{
		Name: "b1", TargetPath: "/t", SecretToken: "tok", URLTemplate: "/u",
	}
	_, _ = dao.AddBucket(b)
	_, e := dao.AddBucket(b)
	if e == nil {
		t.Fatal("expected error when adding duplicate bucket")
	}
	var notAllowed err.NotAllowedError
	if !errors.As(e, &notAllowed) {
		t.Errorf("expected NotAllowedError, got %T: %v", e, e)
	}
}
