package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/common/utils"
	"log"

	"gorm.io/gorm"
)

type FileBucketDAO struct {
	db    *DB
	cache *utils.KVCache[types.FileBucket]
}

func NewFileBucketDAO(db *DB, ch *registry.ComponentsHolder) *FileBucketDAO {
	dao := &FileBucketDAO{db: db, cache: utils.NewKVCache[types.FileBucket](0)}
	ch.Add("fileBucketDAO", dao)
	dao.reloadBuckets()
	return dao
}

func (f *FileBucketDAO) reloadBuckets() {
	buckets, e := f.GetBuckets()
	if e != nil {
		log.Println("failed to reload buckets: ", e)
		return
	}
	for _, bucket := range buckets {
		f.cache.Set(bucket.Name, bucket, 0)
	}
}

func (f *FileBucketDAO) GetBuckets() ([]types.FileBucket, error) {
	var buckets []types.FileBucket
	e := f.db.C().Find(&buckets).Error
	return buckets, e
}

func (f *FileBucketDAO) GetBucket(name string) (types.FileBucket, error) {
	bucket, ok := f.cache.Get(name)
	if !ok {
		return types.FileBucket{}, err.NewNotFoundError()
	}
	return bucket, nil
}

func (f *FileBucketDAO) AddBucket(bucket types.FileBucket) (types.FileBucket, error) {
	e := f.db.C().Where("`name` = ?", bucket.Name).Take(&types.FileBucket{}).Error
	if e == nil {
		return types.FileBucket{},
			err.NewNotAllowedMessageError(i18n.T("storage.file_bucket.bucket_exists", bucket.Name))
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return types.FileBucket{}, e
	}
	e = f.db.C().Create(&bucket).Error
	if e == nil {
		f.cache.Set(bucket.Name, bucket, 0)
	}
	return bucket, e
}

func (f *FileBucketDAO) UpdateBucket(name string, bucket types.FileBucket) error {
	bucket.Name = name
	e := f.db.C().Save(bucket).Error
	if e == nil {
		f.cache.Set(bucket.Name, bucket, 0)
	}
	return e
}

func (f *FileBucketDAO) DeleteBucket(name string) error {
	e := f.db.C().Delete(&types.FileBucket{}, "`name` = ?", name).Error
	if e == nil {
		f.cache.Remove(name)
	}
	return e
}

func (f *FileBucketDAO) Dispose() error {
	return f.cache.Dispose()
}
