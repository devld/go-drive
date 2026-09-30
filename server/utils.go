package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/auth"
	"go-drive/storage"
	"io"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var authLog = logging.For("auth")

const (
	mimeTypes = "aac:audio/aac;abw:application/x-abiword;arc:application/x-freearc;avif:image/avif;avi:video/x-msvideo;azw:application/vnd.amazon.ebook;bmp:image/bmp;bz:application/x-bzip;bz2:application/x-bzip2;cda:application/x-cdf;csh:application/x-csh;css:text/css;csv:text/csv;doc:application/msword;docx:application/vnd.openxmlformats-officedocument.wordprocessingml.document;eot:application/vnd.ms-fontobject;epub:application/epub+zip;gz:application/gzip;gif:image/gif;htm,html:text/html;ico:image/vnd.microsoft.icon;ics:text/calendar;jar:application/java-archive;jpeg,jpg:image/jpeg;js:text/javascript;json:application/json;jsonld:application/ld+json;mid,midi:audio/midi;mjs:text/javascript;mp3:audio/mpeg;mp4:video/mp4;mpeg:video/mpeg;mpkg:application/vnd.apple.installer+xml;odp:application/vnd.oasis.opendocument.presentation;ods:application/vnd.oasis.opendocument.spreadsheet;odt:application/vnd.oasis.opendocument.text;oga:audio/ogg;ogv:video/ogg;ogx:application/ogg;opus:audio/opus;otf:font/otf;png:image/png;pdf:application/pdf;php:application/x-httpd-php;ppt:application/vnd.ms-powerpoint;pptx:application/vnd.openxmlformats-officedocument.presentationml.presentation;rar:application/vnd.rar;rtf:application/rtf;sh:application/x-sh;svg:image/svg+xml;tar:application/x-tar;tif,tiff:image/tiff;ts:video/mp2t;ttf:font/ttf;txt:text/plain;vsd:application/vnd.visio;wav:audio/wav;weba:audio/webm;webm:video/webm;webp:image/webp;woff:font/woff;woff2:font/woff2;xhtml:application/xhtml+xml;xls:application/vnd.ms-excel;xlsx:application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;xml:application/xml;xul:application/vnd.mozilla.xul+xml;zip:application/zip;3gp:video/3gpp;3g2:video/3gpp2;7z:application/x-7z-compressed;apk:application/vnd.android.package-archive;ipa,exe:application/octet-stream;plist:application/x-plist"
)

func init() {
	for _, i := range strings.Split(mimeTypes, ";") {
		t := strings.Split(i, ":")
		for _, j := range strings.Split(t[0], ",") {
			mime.AddExtensionType("."+j, t[1])
		}
	}
}

const (
	keyToken         = "token"
	keySession       = "session"
	keyResult        = "apiResult"
	keyMessageSource = "messageSource"

	signaturePathUserSep = "\x00"
)

var errBadSignature = err.NewBadRequestError("bad signature")

func signatureAuth(signer *utils.Signer, userDAO *storage.UserDAO, signatureRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.Query(common.SignatureQueryKey)
		if signature == "" {
			if signatureRequired {
				authLog.Debugf("signature rejected reason=missing path=%s", logging.Sanitize(c.Request.URL.Path))
				_ = c.Error(errBadSignature)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		var username string

		path, e := getQueryPath(c, "path")
		if e != nil {
			authLog.Debugf("signature rejected reason=invalid_path path=%s: %v", logging.Sanitize(c.Request.URL.Path), e)
			_ = c.Error(e)
			c.Abort()
			return
		}

		parts := strings.Split(signature, ".")
		signature = parts[0]
		if len(parts) > 1 {
			temp, e := utils.Base64URLDecode(parts[1])
			if e != nil {
				authLog.Debugf("signature rejected reason=invalid_principal path=%s", logging.Sanitize(path))
				_ = c.Error(errBadSignature)
				c.Abort()
				return
			}
			username = string(temp)
		}

		if !signer.Validate(path+signaturePathUserSep+username, signature) {
			authLog.Debugf("signature rejected reason=invalid_signature path=%s user=%s",
				logging.Sanitize(path), logging.Sanitize(username))
			_ = c.Error(errBadSignature)
			c.Abort()
			return
		}

		// a valid signature already proves authorization for this path, even for
		// an anonymous access key (empty username)
		principal := types.Principal{AuthType: types.AuthTypeSignature}
		if username != "" {
			user, e := userDAO.GetUser(username)
			if e != nil {
				authLog.Debugf("signature rejected reason=unknown_user user=%s", logging.Sanitize(username))
				_ = c.Error(errBadSignature)
				c.Abort()
				return
			}
			principal.User = user
		}

		setPrincipal(c, principal)
		c.Next()
	}
}

func getQueryPath(c *gin.Context, key string) (string, error) {
	path, exists := c.GetQuery(key)
	if !exists {
		return "", err.NewBadRequestError("missing query parameter: " + key)
	}
	return utils.CleanPath(path), nil
}

func makeSignature(signer *utils.Signer, path, username string, ttl time.Duration) string {
	signature := signer.Sign(path+signaturePathUserSep+username, time.Now().Add(ttl))
	return signature + "." + utils.Base64URLEncode([]byte(username))
}

func tokenAuthMiddleware(tokenStore types.TokenStore) gin.HandlerFunc {
	return tokenAuth(tokenStore, func(c *gin.Context) string {
		return c.GetHeader(common.HeaderAuth)
	})
}

func tokenAuth(tokenStore types.TokenStore, getToken func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthenticated(c) {
			c.Next()
			return
		}

		tokenKey := getToken(c)
		if tokenKey == "" {
			// no token: browse as an anonymous session
			setPrincipal(c, types.Principal{})
			c.Next()
			return
		}
		token, e := tokenStore.Validate(tokenKey)
		if e != nil {
			authLog.Debugf("token rejected method=%s path=%s: %v", c.Request.Method,
				logging.Sanitize(c.Request.URL.Path), e)
			_ = c.Error(e)
			c.Abort()
			return
		}

		setToken(c, token.Token)
		setPrincipal(c, token.Value)

		c.Next()
	}
}

func basicAuth(userAuth *auth.UserAuth, realm string, allowAnonymous bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthenticated(c) {
			c.Next()
			return
		}

		username, password, ok := c.Request.BasicAuth()
		principal := types.Principal{}
		if ok {
			user, e := userAuth.AuthByUsernamePassword(username, password)
			if e != nil {
				if !err.IsNotAllowedError(e) {
					authLog.Errorf("basic authentication failed user=%s: %v", logging.Sanitize(username), e)
					_ = c.Error(e)
					c.Abort()
					return
				}
				// invalid credentials: fall back to an anonymous principal
				authLog.Debugf("basic authentication rejected user=%s", logging.Sanitize(username))
			} else {
				principal = types.Principal{User: user, AuthType: types.AuthTypeBasic}
				authLog.Debugf("basic authentication succeeded user=%s", logging.Sanitize(username))
			}
		}

		if principal.IsAnonymous() && !allowAnonymous {
			authLog.Debugf("basic authentication rejected reason=anonymous_not_allowed path=%s",
				logging.Sanitize(c.Request.URL.Path))
			c.Status(http.StatusUnauthorized)
			c.Header("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
			c.Abort()
			return
		}

		setPrincipal(c, principal)
		c.Next()
	}
}

func userGroupRequired(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := getPrincipal(c)
		if principal.HasUserGroup(group) {
			c.Next()
			return
		}
		_ = c.Error(err.NewNotAllowedMessageError(i18n.T("api.auth.group_permission_required", group)))
		authLog.Debugf("authorization rejected reason=missing_group group=%s path=%s",
			logging.Sanitize(group), logging.Sanitize(c.Request.URL.Path))
		c.Abort()
	}
}

func adminGroupRequired() gin.HandlerFunc {
	return userGroupRequired(types.AdminUserGroup)
}

type flushingWriter struct {
	writer gin.ResponseWriter
}

func (w flushingWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if n > 0 {
		w.writer.Flush()
	}
	return n, err
}

func executeTaskStreaming(c *gin.Context, runner task.Runner,
	runnable func(types.TaskCtx, io.Writer) (any, error), options ...task.Option) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	createdTask, e := runner.Execute(func(ctx types.TaskCtx) (any, error) {
		defer writer.Close()
		return runnable(ctx, writer)
	}, options...)
	if e != nil {
		_ = writer.Close()
		return e
	}
	taskJSON, e := json.Marshal(getTranslator(c).translateV(createdTask))
	if e != nil {
		stopStreamingTask(runner, createdTask.ID)
		_ = c.AbortWithError(http.StatusInternalServerError, e)
		return nil
	}
	c.Header("X-Accel-Buffering", "no") // for nginx to not buffer the response
	c.Header(common.ResponseHeaderKey, string(taskJSON))
	stopWatching := context.AfterFunc(c.Request.Context(), func() { _ = reader.Close() })
	defer stopWatching()

	_, copyErr := io.Copy(flushingWriter{c.Writer}, reader)
	if copyErr != nil || c.Request.Context().Err() != nil {
		stopStreamingTask(runner, createdTask.ID)
	}
	return nil
}

func stopStreamingTask(runner task.Runner, id string) {
	if _, err := runner.StopTask(id); err != nil && !errors.Is(err, task.ErrorNotFound) {
		logging.For("task").Warnf("failed to stop canceled streaming task id=%s: %v", id, err)
	}
}

func readRequestBodyToTempFile(c *gin.Context, tempDir string) (*utils.TempFile, int64, error) {
	size := utils.ToInt64(c.GetHeader("Content-Length"), -1)
	file, e := driveutil.CopyReaderToTempFile(task.NewTaskContext(c.Request.Context()), c.Request.Body, tempDir)
	if e != nil {
		return nil, -1, e
	}
	stat, e := file.Stat()
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, -1, e
	}
	actualSize := stat.Size()
	if size >= 0 {
		if size != actualSize {
			_ = file.Close()
			_ = os.Remove(file.Name())
			return nil, -1, err.NewBadRequestError(i18n.T("api.drive.invalid_file_size"))
		}
	} else {
		size = actualSize
	}
	return utils.NewTempFile(file), size, nil
}

func getRequestOrigin(c *gin.Context) string {
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	protocol := c.GetHeader("X-Forwarded-Proto")
	if protocol == "" {
		if c.Request.TLS != nil {
			protocol = "https"
		}
	}
	if protocol != "https" {
		protocol = "http"
	}
	return protocol + "://" + host
}

func setResult(c *gin.Context, result any) {
	c.Set(keyResult, result)
}

func getResult(c *gin.Context) (any, bool) {
	return c.Get(keyResult)
}

func getToken(c *gin.Context) string {
	return c.GetString(keyToken)
}

func setToken(c *gin.Context, token string) {
	c.Set(keyToken, token)
}

func isAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(keySession)
	return exists
}

func setMessageSource(c *gin.Context, messageSource i18n.MessageSource) {
	c.Set(keyMessageSource, messageSource)
}

func getMessageSource(c *gin.Context) i18n.MessageSource {
	if ms, exists := c.Get(keyMessageSource); exists {
		return ms.(i18n.MessageSource)
	}
	return nil
}

func getPrincipal(c *gin.Context) types.Principal {
	if s, exists := c.Get(keySession); exists {
		return s.(types.Principal)
	}
	return types.Principal{}
}

func setPrincipal(c *gin.Context, principal types.Principal) {
	c.Set(keySession, principal)
}

type translator struct {
	language      string
	messageSource i18n.MessageSource
}

func getTranslator(c *gin.Context) translator {
	language, _, _ := strings.Cut(c.GetHeader("Accept-Language"), ",")
	return translator{language: language, messageSource: getMessageSource(c)}
}

func (t translator) translateV(v any) any {
	return i18n.TranslateV(t.language, t.messageSource, v)
}

func (t translator) translateT(text string) string {
	return i18n.TranslateT(t.language, t.messageSource, text)
}

func writeJSON(c *gin.Context, code int, v any) {
	if c.Writer.Written() {
		return
	}
	result := getTranslator(c).translateV(v)
	c.JSON(code, result)
}

var pathSegmentPattern = regexp.MustCompile("^[^/\\\x00:*\"<>|]+$")

func checkPathSegment(name string, errI18nKey string) error {
	if name == "" || name == "." || name == ".." || !pathSegmentPattern.MatchString(name) {
		return err.NewBadRequestError(i18n.T(errI18nKey, name))
	}
	return nil
}
