package server

import (
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"io"
	"log"
	"net/http"
	"os"
	path2 "path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
)

const (
	fileBucketSecretTokenKey = "t"
)

func InitFileBucketRoutes(
	router gin.IRouter,
	config common.Config,
	access *drive.Access,
	fileBucketDAO *storage.FileBucketDAO) error {

	fr := &fileBucketRoute{config, access, fileBucketDAO}

	r := router.Group("/f/:name", fr._getBucketDrive)

	// upload file
	r.POST("", fr.upload)
	r.POST("/*path", fr.upload)

	// get file
	r.HEAD("/*path", fr.get)
	r.GET("/*path", fr.get)

	return nil
}

type fileBucketRoute struct {
	config        common.Config
	access        *drive.Access
	fileBucketDAO *storage.FileBucketDAO
}

func (fr *fileBucketRoute) _getBucketDrive(c *gin.Context) {
	bucketName := c.Param("name")
	bucket, e := fr.fileBucketDAO.GetBucket(bucketName)
	if e != nil {
		fr.abortWithError(c, e)
		return
	}
	bucketDrive := fr.access.GetRootDrive(nil)
	chroot := drive.NewChroot(bucket.TargetPath, nil)
	bucketDrive = drive.NewChrootWrapper(bucketDrive, chroot)
	c.Set("bucket", bucket)
	c.Set("drive", bucketDrive)
	c.Next()
}

func (fr *fileBucketRoute) checkAllowedTypes(mimeType, fileExt, allowedTypes string) bool {
	allowedTypes = strings.ReplaceAll(allowedTypes, " ", "")
	if allowedTypes == "" {
		return true
	}
	allowed := strings.Split(allowedTypes, ",")
	for _, t := range allowed {
		t = strings.TrimSpace(t)
		if t == mimeType || t == fileExt {
			return true
		}
		if strings.HasSuffix(t, "/*") {
			if strings.HasPrefix(mimeType, strings.TrimSuffix(t, "*")) {
				return true
			}
		}
	}
	return false
}

func (fr *fileBucketRoute) upload(c *gin.Context) {
	bucket := c.MustGet("bucket").(types.FileBucket)
	if c.Query(fileBucketSecretTokenKey) != bucket.SecretToken {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var file io.ReadCloser
	var fileSize int64
	var fileType string
	var filename string
	var fileExt string

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		form, e := c.MultipartForm()
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		if len(form.File["file"]) != 1 {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		multipartFile := form.File["file"][0]
		file, e = multipartFile.Open()
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		defer func() { _ = file.Close() }()
		fileSize = multipartFile.Size
		fileType = multipartFile.Header.Get("Content-Type")
		fileExt = path2.Ext(multipartFile.Filename)
		filename = strings.TrimSuffix(multipartFile.Filename, fileExt)
	} else {
		savedFile, size, e := ReadRequestBodyToTempFile(c, fr.config.TempDir)
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		defer func() {
			_ = savedFile.Close()
			_ = os.Remove(savedFile.Name())
		}()
		file = savedFile
		fileSize = size

		var fileMime *mimetype.MIME
		fileType = c.GetHeader("Content-Type")
		if fileType == "" {
			detectedType, e := mimetype.DetectReader(savedFile)
			if e != nil {
				fr.abortWithError(c, e)
				return
			}
			fileMime = detectedType
			fileType = detectedType.String()
		} else {
			fileMime = mimetype.Lookup(fileType)
		}
		if fileMime != nil {
			fileExt = fileMime.Extension()
		}
	}

	maxSize := types.SV(bucket.MaxSize).DataSize(0)
	if maxSize > 0 && fileSize > maxSize {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	if !fr.checkAllowedTypes(fileType, fileExt, bucket.AllowedTypes) {
		c.AbortWithStatus(http.StatusNotAcceptable)
		return
	}

	bucketDrive := c.MustGet("drive").(types.IDrive)
	path := utils.CleanPath(c.Param("path"))
	if path == "" || !bucket.CustomKey {
		path = fr.generateKey(bucket.KeyTemplate, keyTemplateValues{now: time.Now(), name: filename, ext: fileExt})
	}

	savedEntry, e := bucketDrive.Save(task.NewTaskContext(c.Request.Context()), path, fileSize, false, file)
	if e != nil {
		fr.abortWithError(c, e)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"url":      fr.generateURL(bucket.URLTemplate, urlTemplateValues{ctx: c, bucketName: bucket.Name, key: savedEntry.Path()}),
		"size":     savedEntry.Size(),
		"mimeType": fileType,
	})
}

func (fr *fileBucketRoute) get(c *gin.Context) {
	bucketDrive := c.MustGet("drive").(types.IDrive)
	path := utils.CleanPath(c.Param("path"))
	entry, e := bucketDrive.Get(c, path)
	if e != nil {
		fr.abortWithError(c, e)
		return
	}
	if !entry.Type().IsFile() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if e := drive_util.DownloadIContent(c.Request.Context(), entry, c.Writer, c.Request, false); e != nil {
		fr.abortWithError(c, e)
		return
	}
}

func (fr *fileBucketRoute) abortWithError(c *gin.Context, e error) {
	if ge, ok := e.(err.Error); ok {
		c.AbortWithStatus(ge.Code())
		return
	}
	log.Println("unknown error", e)
	c.AbortWithError(http.StatusInternalServerError, e)
}

func (fr *fileBucketRoute) generateURL(template string, values urlTemplateValues) string {
	if strings.TrimSpace(template) == "" {
		template = "{origin}/f/{bucket}/{key}"
	}
	template = strings.ReplaceAll(template, "{origin}", GetRequestOrigin(values.ctx)+fr.config.APIPath)
	template = strings.ReplaceAll(template, "{bucket}", values.bucketName)
	template = strings.ReplaceAll(template, "{key}", values.key)
	template = anyVariableRegexp.ReplaceAllString(template, "")
	return template
}

func (fr *fileBucketRoute) generateKey(template string, values keyTemplateValues) string {
	if strings.TrimSpace(template) == "" {
		template = "{year}{month}{date}/{name}-{rand}{ext}"
	}
	for _, v := range keyTemplateVariables {
		template = strings.ReplaceAll(template, "{"+v.Name+"}", v.Value(values))
	}
	template = anyVariableRegexp.ReplaceAllString(template, "")
	return template
}

type urlTemplateValues struct {
	ctx        *gin.Context
	bucketName string
	key        string
}

type keyTemplateValues struct {
	now  time.Time
	name string
	ext  string
}

var anyVariableRegexp = regexp.MustCompile("{[^}]*}")

var keyTemplateVariables = []struct {
	Name  string
	Value func(keyTemplateValues) string
}{
	{"year", func(v keyTemplateValues) string { return v.now.Format("2006") }},
	{"month", func(v keyTemplateValues) string { return v.now.Format("01") }},
	{"date", func(v keyTemplateValues) string { return v.now.Format("02") }},
	{"hour", func(v keyTemplateValues) string { return v.now.Format("15") }},
	{"minute", func(v keyTemplateValues) string { return v.now.Format("04") }},
	{"second", func(v keyTemplateValues) string { return v.now.Format("05") }},
	{"millisecond", func(v keyTemplateValues) string { return v.now.Format("000") }},
	{"timestamp", func(v keyTemplateValues) string { return strconv.FormatInt(v.now.UnixMilli(), 10) }},
	{"rand", func(v keyTemplateValues) string { return utils.RandString(16) }},
	{"name", func(v keyTemplateValues) string {
		if v.name == "" {
			return utils.RandString(8)
		}
		return v.name
	}},
	{"ext", func(v keyTemplateValues) string { return path2.Ext(v.ext) }},
}
