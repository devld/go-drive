package drive

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive/cache"
	"io"
	"math"
	"net/url"
	fsPath "path"
	"strings"
	"time"
)

type S3Drive struct {
	c             *s3.S3
	bucket        *string
	uploadProxy   bool
	downloadProxy bool
	cache         cache.DriveCache
}

// NewS3Drive creates a S3 compatible storage
// params:
//   - id: access key
//   - secret: secret key
//   - bucket: the bucket name
//   - path_style: force path style api
//   - region: service region
//   - endpoint: the api endpoint
//   - proxy_upload: whether it needs to be uploaded to server proxy
//   - proxy_download: whether it needs to be downloaded from server proxy
//   - cache_ttl: cache time to live
//   - max_cache: maximum number of caches, if less than or equal to 0, no cache
func NewS3Drive(config map[string]string) (types.IDrive, error) {
	id := config["id"]
	secret := config["secret"]
	bucket := config["bucket"]
	pathStyle := config["path_style"]
	region := config["region"]
	endpoint := config["endpoint"]
	proxyUpload := config["proxy_upload"]
	proxyDownload := config["proxy_download"]
	cacheTtl, e := time.ParseDuration(config["cache_ttl"])
	maxCache := common.ToInt(config["max_cache"], -1)
	if e != nil {
		cacheTtl = -1
	}

	sess, e := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(id, secret, ""),
		S3ForcePathStyle: aws.Bool(pathStyle != ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
	})
	if e != nil {
		return nil, e
	}
	client := s3.New(sess)
	d := &S3Drive{
		c:             client,
		bucket:        aws.String(bucket),
		uploadProxy:   proxyUpload != "",
		downloadProxy: proxyDownload != "",
	}
	if maxCache <= 0 {
		d.cache = cache.DummyCache()
	} else {
		d.cache = cache.NewMemCache(maxCache, cacheTtl, d.deserializeEntry)
	}
	return d, d.check()
}

func (s *S3Drive) check() error {
	_, e := s.c.HeadBucket(&s3.HeadBucketInput{
		Bucket: s.bucket,
	})
	if e != nil {
		if ae, ok := e.(awserr.Error); ok {
			switch ae.Code() {
			case s3.ErrCodeNoSuchBucket:
				return common.NewNotFoundMessageError(fmt.Sprintf("Bucket '%s' not found", *s.bucket))
			}
		}
	}
	return e
}

func (s *S3Drive) deserializeEntry(dat string) (types.IEntry, error) {
	temp := strings.SplitN(dat, ",", 4)
	if len(temp) != 4 {
		return nil, errors.New("")
	}
	isDir := temp[0] == "1"
	modTime := common.ToInt64(temp[1], -1)
	size := common.ToInt64(temp[2], -1)
	return &s3Entry{
		key:     temp[3],
		c:       s,
		size:    size,
		modTime: modTime,
		isDir:   isDir,
	}, nil
}

func (s *S3Drive) Meta() types.DriveMeta {
	return types.DriveMeta{CanWrite: true}
}

func (s *S3Drive) get(path string) (*s3Entry, error) {
	obj, e := s.c.HeadObject(&s3.HeadObjectInput{
		Bucket: s.bucket,
		Key:    aws.String(path),
	})
	if e != nil {
		if errCodeMatches(e, "NotFound") {
			if strings.HasSuffix(path, "/") {
				return nil, common.NewNotFoundError()
			}
			return s.get(path + "/")
		}
		return nil, e
	}
	if strings.HasSuffix(path, "/") {
		return s.newS3DirEntry(path, obj.LastModified), nil
	}
	return s.newS3ObjectEntry(path, obj.ContentLength, obj.LastModified), nil
}

func (s *S3Drive) Get(path string) (types.IEntry, error) {
	if common.IsRootPath(path) {
		return s.newS3DirEntry(path, nil), nil
	}
	cached, _ := s.cache.GetEntry(path)
	if cached != nil {
		return cached, nil
	}
	entry, e := s.get(path)
	if e != nil {
		return nil, e
	}
	_ = s.cache.PutEntry(entry)
	return entry, nil
}

func (s *S3Drive) Save(path string, reader io.Reader, ctx task.Context) (types.IEntry, error) {
	var readSeeker io.ReadSeeker
	if rs, ok := reader.(io.ReadSeeker); ok {
		readSeeker = rs
	} else {
		file, e := common.CopyReaderToTempFile(reader, ctx)
		if e != nil {
			return nil, e
		}
		readSeeker = file
	}
	_, e := s.c.PutObject(&s3.PutObjectInput{
		Bucket: s.bucket,
		Key:    aws.String(path),
		Body:   readSeeker,
	})
	if e != nil {
		return nil, e
	}
	_ = s.cache.Evict(common.PathParent(path))
	return s.Get(path)
}

func (s *S3Drive) MakeDir(path string) (types.IEntry, error) {
	path = path + "/"
	_, e := s.Get(path)
	if e == nil {
		return nil, common.NewNotAllowedMessageError("file exists")
	}
	if !common.IsNotFoundError(e) {
		return nil, e
	}
	_, e = s.c.PutObject(&s3.PutObjectInput{
		Bucket: s.bucket,
		Key:    aws.String(path),
	})
	if e != nil {
		return nil, e
	}
	_ = s.cache.Evict(common.PathParent(path))
	return s.newS3DirEntry(path, nil), nil
}

func (s *S3Drive) isSelf(e types.IEntry) bool {
	if fe, ok := e.(*s3Entry); ok {
		return fe.c == s
	}
	return false
}

func (s *S3Drive) Copy(from types.IEntry, to string, override bool, _ task.Context) (types.IEntry, error) {
	from = common.GetIEntry(from, s.isSelf)
	if from == nil || from.Type().IsDir() {
		return nil, common.NewUnsupportedError()
	}
	entry, _, e := s.copy(from.(*s3Entry), to, override)
	return entry, e
}

func (s *S3Drive) copy(from *s3Entry, to string, override bool) (*s3Entry, bool, error) {
	if !override {
		_, e := s.Get(to)
		if e == nil {
			modTime := common.Time(from.modTime)
			// skip
			return s.newS3ObjectEntry(to, &from.size, &modTime), true, nil
		}
		if !common.IsNotFoundError(e) {
			return nil, false, e
		}
	}
	obj, e := s.c.CopyObject(&s3.CopyObjectInput{
		Bucket:     s.bucket,
		Key:        aws.String(to),
		CopySource: aws.String(url.QueryEscape(*s.bucket + "/" + from.key)),
	})
	if e != nil {
		return nil, false, e
	}
	_ = s.cache.Evict(common.PathParent(to))
	return s.newS3ObjectEntry(to, &from.size, obj.CopyObjectResult.LastModified), false, nil
}

func (s *S3Drive) Move(from types.IEntry, to string, override bool, _ task.Context) (types.IEntry, error) {
	from = common.GetIEntry(from, s.isSelf)
	if from == nil || from.Type().IsDir() {
		return nil, common.NewUnsupportedMessageError("Move files/dirs across drives is not supported.")
	}
	fromEntry := from.(*s3Entry)
	entry, skip, e := s.copy(fromEntry, to, override)
	if e != nil {
		return nil, e
	}
	if !skip {
		e = s.Delete(fromEntry.key, task.DummyContext())
	}
	return entry, e
}

func (s *S3Drive) List(path string) ([]types.IEntry, error) {
	cached, _ := s.cache.GetChildren(path)
	if cached != nil {
		return cached, nil
	}
	s3Path := path
	if !common.IsRootPath(s3Path) {
		s3Path = s3Path + "/"
	}
	objs, e := s.c.ListObjects(&s3.ListObjectsInput{
		Bucket:    s.bucket,
		Prefix:    aws.String(s3Path),
		Delimiter: aws.String("/"),
	})
	if e != nil {
		return nil, e
	}
	entries := make([]types.IEntry, 0)
	pathSet := make(map[string]bool, 0)
	for _, o := range objs.Contents {
		if *o.Key == s3Path {
			// fake dir
			continue
		}
		entries = append(entries, s.newS3ObjectEntry(*o.Key, o.Size, o.LastModified))
		pathSet[*o.Key] = true
	}
	for _, p := range objs.CommonPrefixes {
		if _, ok := pathSet[(*p.Prefix)[:len(*p.Prefix)-1]]; ok {
			// skip dir with same name
			continue
		}
		entries = append(entries, s.newS3DirEntry(*p.Prefix, nil))
	}
	cacheable := make([]cache.CacheableEntry, len(entries))
	for i, e := range entries {
		cacheable[i] = e.(cache.CacheableEntry)
	}
	_ = s.cache.PutChildren(path, cacheable)
	return entries, nil
}

func (s *S3Drive) Delete(path string, ctx task.Context) error {
	entry, e := s.Get(path)
	if e != nil {
		return e
	}
	tree, e := common.BuildEntriesTree(entry, ctx)
	if e != nil {
		return e
	}
	entries := common.FlattenEntriesTree(tree)
	n := int(math.Ceil(float64(len(entries)) / 1000))
	deleted := 0
	for i := 0; i < n; i += 1 {
		batches := entries[i*1000 : int(math.Min(float64((i+1)*1000), float64(len(entries))))]
		deletes := make([]*s3.ObjectIdentifier, len(batches))
		for i, o := range batches {
			key := o.Path()
			if o.Type().IsDir() {
				key += "/"
			}
			deletes[i] = &s3.ObjectIdentifier{
				Key: aws.String(key),
			}
		}
		r, e := s.c.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: s.bucket,
			Delete: &s3.Delete{
				Objects: deletes,
				Quiet:   aws.Bool(true),
			},
		})
		if e != nil {
			return e
		}
		if r.Errors != nil && len(r.Errors) > 0 {
			return errors.New(fmt.Sprintf("%s: %s", *r.Errors[0].Key, *r.Errors[0].Code))
		}
		deleted += len(batches)
		ctx.Progress(int64(deleted))
	}
	_ = s.cache.Evict(common.PathParent(path))
	return nil
}

func (s *S3Drive) Upload(path string, size int64, _ bool,
	config map[string]string) (*types.DriveUploadConfig, error) {
	action := config["action"]
	uploadId := config["uploadId"]
	partsEtag := config["parts"]
	seq := common.ToInt64(config["seq"], -1)

	r := types.DriveUploadConfig{
		Provider: types.S3Provider,
		Config:   map[string]string{},
	}
	preSigned := ""

	var e error
	switch action {
	case "UploadPart":
		req, _ := s.c.UploadPartRequest(&s3.UploadPartInput{
			Bucket:     s.bucket,
			Key:        aws.String(path),
			PartNumber: aws.Int64(seq + 1),
			UploadId:   aws.String(uploadId),
		})
		preSigned, e = req.Presign(2 * time.Hour)
		break
	case "CompleteMultipartUpload":
		_, e := s.c.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
			Bucket:   s.bucket,
			Key:      aws.String(path),
			UploadId: aws.String(uploadId),
			MultipartUpload: &s3.CompletedMultipartUpload{
				Parts: buildCompleteUploadBody(partsEtag),
			},
		})
		_ = s.cache.Evict(common.PathParent(path))
		return nil, e
	case "AbortMultipartUpload":
		_, e := s.c.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   s.bucket,
			Key:      aws.String(path),
			UploadId: aws.String(uploadId),
		})
		return nil, e
	case "CompletePutObject":
		_ = s.cache.Evict(common.PathParent(path))
		return nil, nil
	default:
		if size <= 5*1024*1024 {
			if s.uploadProxy {
				return &types.DriveUploadConfig{Provider: types.LocalProvider}, nil
			}
			req, _ := s.c.PutObjectRequest(&s3.PutObjectInput{
				Bucket: s.bucket,
				Key:    aws.String(path),
			})
			preSigned, e = req.Presign(2 * time.Hour)
		} else {
			if s.uploadProxy {
				return &types.DriveUploadConfig{Provider: types.LocalChunkProvider}, nil
			}
			req, _ := s.c.CreateMultipartUploadRequest(&s3.CreateMultipartUploadInput{
				Bucket: s.bucket,
				Key:    aws.String(path),
			})
			preSigned, e = req.Presign(2 * time.Hour)
			r.Config["multipart"] = "1"
		}
	}
	if e != nil {
		return nil, e
	}
	if preSigned != "" {
		r.Config["url"] = preSigned
	}
	return &r, e
}

func buildCompleteUploadBody(etag string) []*s3.CompletedPart {
	temp := strings.Split(etag, ";")
	r := make([]*s3.CompletedPart, len(temp))
	for i, e := range temp {
		r[i] = &s3.CompletedPart{
			PartNumber: aws.Int64(int64(i + 1)),
			ETag:       aws.String(e),
		}
	}
	return r
}

func (s *S3Drive) Dispose() error {
	return nil
}

func (s *S3Drive) newS3DirEntry(path string, lastModified *time.Time) *s3Entry {
	var mtime int64 = -1
	if lastModified != nil {
		mtime = common.Millisecond(*lastModified)
	}
	path = common.CleanPath(path)
	return &s3Entry{
		isDir:   true,
		key:     path,
		modTime: mtime,
		c:       s,
	}
}

func (s *S3Drive) newS3ObjectEntry(path string, size *int64, lastModified *time.Time) *s3Entry {
	path = common.CleanPath(path)
	return &s3Entry{
		isDir:   false,
		key:     path,
		size:    *size,
		modTime: common.Millisecond(*lastModified),
		c:       s,
	}
}

type s3Entry struct {
	key     string
	c       *S3Drive
	size    int64
	modTime int64
	isDir   bool
}

func (s *s3Entry) Path() string {
	return s.key
}

func (s *s3Entry) Type() types.EntryType {
	if s.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (s *s3Entry) Size() int64 {
	if s.isDir {
		return -1
	}
	return s.size
}

func (s *s3Entry) Meta() types.EntryMeta {
	return types.EntryMeta{
		CanRead:  true,
		CanWrite: true,
	}
}

func (s *s3Entry) ModTime() int64 {
	if s.isDir {
		return -1
	}
	return s.modTime
}

func (s *s3Entry) Drive() types.IDrive {
	return s.c
}

func (s *s3Entry) Name() string {
	return fsPath.Base(s.key)
}

func (s *s3Entry) GetReader() (io.ReadCloser, error) {
	obj, e := s.c.c.GetObject(&s3.GetObjectInput{
		Bucket: s.c.bucket,
		Key:    aws.String(s.key),
	})
	if e != nil {
		return nil, e
	}
	return obj.Body, nil
}

func (s *s3Entry) GetURL() (string, bool, error) {
	req, _ := s.c.c.GetObjectRequest(&s3.GetObjectInput{
		Bucket: s.c.bucket,
		Key:    aws.String(s.key),
	})
	downloadUrl, e := req.Presign(8 * time.Hour)
	if e != nil {
		return "", false, e
	}
	return downloadUrl, s.c.downloadProxy, nil
}

func (s *s3Entry) Serialize() string {
	r := ""
	if s.isDir {
		r = "1"
	} else {
		r = "0"
	}
	return r + "," + fmt.Sprintf("%d,%d,%s", s.modTime, s.size, s.key)
}

func errCodeMatches(e error, code string) bool {
	if ae, ok := e.(awserr.Error); ok {
		return ae.Code() == code
	}
	return false
}
