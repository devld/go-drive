package server

import (
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"io"
	"log"
	"net/http"
	"net/url"
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
	FileBucketSecretTokenKey     = "t"
	FileBucketDefaultCacheMaxAge = "1d"
)

func InitFileBucketRoutes(
	router gin.IRouter,
	config common.Config,
	access *drive.Access,
	fileBucketDAO *storage.FileBucketDAO,
	messageSource i18n.MessageSource) error {

	fr := &fileBucketRoute{config, access, fileBucketDAO, messageSource}

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
	messageSource i18n.MessageSource
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
	if c.Query(FileBucketSecretTokenKey) != bucket.SecretToken {
		fr.abortWithMessage(c, http.StatusUnauthorized, "invalid token")
		return
	}

	var file io.ReadCloser
	var fileSize int64
	var fileMime *mimetype.MIME
	var filename string

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		form, e := c.MultipartForm()
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		if len(form.File["file"]) != 1 {
			fr.abortWithMessage(c, http.StatusBadRequest, "bad request")
			return
		}
		multipartFile := form.File["file"][0]
		formFile, e := multipartFile.Open()
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		defer func() { _ = file.Close() }()

		fileMime, e = mimetype.DetectReader(formFile)
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		_, e = formFile.Seek(0, io.SeekStart)
		if e != nil {
			fr.abortWithError(c, e)
			return
		}

		file = formFile
		fileSize = multipartFile.Size
		filename = multipartFile.Filename
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

		detectedType, e := mimetype.DetectReader(savedFile)
		if e != nil {
			fr.abortWithError(c, e)
			return
		}
		fileMime = detectedType
	}

	fileType := fileMime.String()
	fileExt := fileMime.Extension()
	if filename != "" && strings.HasSuffix(strings.ToLower(filename), fileExt) {
		filename = strings.TrimSuffix(filename, fileExt)
	}

	maxSize := types.SV(bucket.MaxSize).DataSize(0)
	if maxSize > 0 && fileSize > maxSize {
		fr.abortWithMessage(c, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	if !fr.checkAllowedTypes(fileType, fileExt, bucket.AllowedTypes) {
		fr.abortWithMessage(c, http.StatusForbidden, "file type not allowed")
		return
	}

	bucketDrive := c.MustGet("drive").(types.IDrive)
	path := c.Param("path")
	if path == "" || !bucket.CustomKey {
		path = fr.generateKey(bucket.KeyTemplate, keyTemplateValues{now: time.Now(), name: filename, ext: fileExt})
	}
	path = utils.CleanPath(path)

	savedEntry, e := bucketDrive.Save(task.NewTaskContext(c.Request.Context()), path, fileSize, false, file)
	if e != nil {
		fr.abortWithError(c, e)
		return
	}

	c.Header("X-File-Size", strconv.FormatInt(savedEntry.Size(), 10))
	c.Header("X-File-Mime", fileMime.String())
	c.Writer.WriteString(fr.generateURL(bucket.URLTemplate, urlTemplateValues{ctx: c, bucketName: bucket.Name, key: savedEntry.Path()}))
}

func (fr *fileBucketRoute) checkReferrers(referrer, allowedReferrers string) bool {
	if allowedReferrers == "" {
		return true
	}
	if referrer != "" {
		parsedReferrer, e := url.Parse(referrer)
		if e != nil {
			return false
		}
		referrer = parsedReferrer.Host
	}
	allowed := strings.Split(allowedReferrers, ",")
	for _, r := range allowed {
		r = strings.TrimSpace(r)
		if r == "" && referrer == "" {
			// allow empty referrers
			return true
		}
		if r == referrer || (strings.HasPrefix(r, "*.") && strings.HasSuffix(referrer, strings.TrimPrefix(r, "*"))) {
			return true
		}
	}
	return false

}

func (fr *fileBucketRoute) get(c *gin.Context) {
	bucket := c.MustGet("bucket").(types.FileBucket)
	bucketDrive := c.MustGet("drive").(types.IDrive)

	if !fr.checkReferrers(c.Request.Referer(), bucket.AllowedReferrers) {
		fr.abortWithMessage(c, http.StatusForbidden, "")
		return
	}

	path := utils.CleanPath(c.Param("path"))
	entry, e := bucketDrive.Get(c, path)
	if e != nil {
		fr.abortWithError(c, e)
		return
	}
	if !entry.Type().IsFile() {
		fr.abortWithMessage(c, http.StatusForbidden, "not found")
		return
	}
	cacheMaxAge := bucket.CacheMaxAge
	if cacheMaxAge == "" {
		cacheMaxAge = FileBucketDefaultCacheMaxAge
	}
	cacheControlMaxAge := int64(types.SV(cacheMaxAge).Duration(0)) / int64(time.Second)
	if cacheControlMaxAge > 0 {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheControlMaxAge))
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	if e := drive_util.DownloadIContent(c.Request.Context(), entry, c.Writer, c.Request, false); e != nil {
		fr.abortWithError(c, e)
		return
	}
}

func (fr *fileBucketRoute) abortWithError(c *gin.Context, e error) {
	if ge, ok := e.(err.Error); ok {
		fr.abortWithMessage(c, ge.Code(), TranslateV(c, fr.messageSource, ge.Error()).(string))
		return
	}
	log.Println("unknown error", e)
	fr.abortWithMessage(c, http.StatusInternalServerError, "internal server error")
}

func (fr *fileBucketRoute) abortWithMessage(c *gin.Context, status int, message string) {
	c.AbortWithStatus(status)
	c.Writer.WriteString(message)
}

func (fr *fileBucketRoute) generateURL(template string, values urlTemplateValues) string {
	if strings.TrimSpace(template) == "" {
		template = "{origin}/f/{bucket}/{key}"
	}
	template = strings.ReplaceAll(template, "{origin}", GetRequestOrigin(values.ctx)+fr.config.APIPath)
	template = strings.ReplaceAll(template, "{bucket}", url.PathEscape(values.bucketName))
	template = strings.ReplaceAll(template, "{key}", utils.URLEncodePath(values.key))
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
