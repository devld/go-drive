package server

import (
	"encoding/json"
	"fmt"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/storage"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
)

func SignatureAuth(signer *utils.Signer, userDAO *storage.UserDAO, skipOnEmptySignature bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.Query(common.SignatureQueryKey)
		if signature == "" && skipOnEmptySignature {
			c.Next()
			return
		}

		session := types.NewSession()
		var username string

		path := utils.CleanPath(c.Param("path"))

		if signature != "" {
			parts := strings.Split(signature, ".")
			signature = parts[0]
			if len(parts) > 1 {
				temp, e := utils.Base64URLDecode(parts[1])
				if e != nil {
					c.AbortWithError(http.StatusBadRequest, e)
					return
				}
				username = string(temp)
			}

			if !signer.Validate(path+username, signature) {
				_ = c.Error(err.NewBadRequestError("bad signature"))
				c.Abort()
				return
			}
		}

		if username != "" {
			user, e := userDAO.GetUser(username)
			if e != nil {
				_ = c.Error(err.NewBadRequestError("bad signature"))
				c.Abort()
				return
			}
			session.User = user
		}

		SetSession(c, session)
		c.Next()
	}
}

func MakeSignature(signer *utils.Signer, path, username string, ttl time.Duration) string {
	signature := signer.Sign(path+username, time.Now().Add(ttl))
	return signature + "." + utils.Base64URLEncode([]byte(username))
}

// TokenAuthWithPostParams get token from Header or FormData
func TokenAuthWithPostParams(tokenStore types.TokenStore) gin.HandlerFunc {
	return tokenAuth(tokenStore, func(c *gin.Context) string {
		t := c.PostForm(common.ParamAuth)
		if t != "" {
			return t
		}
		return c.GetHeader(common.HeaderAuth)
	})
}

func TokenAuth(tokenStore types.TokenStore) gin.HandlerFunc {
	return tokenAuth(tokenStore, func(c *gin.Context) string {
		return c.GetHeader(common.HeaderAuth)
	})
}

func tokenAuth(tokenStore types.TokenStore, getToken func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsAuthenticated(c) {
			c.Next()
			return
		}

		tokenKey := getToken(c)
		token, e := tokenStore.Validate(tokenKey)
		if e != nil {
			_ = c.Error(e)
			c.Abort()
			return
		}
		session := token.Value

		SetToken(c, token.Token)
		SetSession(c, session)

		c.Next()
	}
}

func BasicAuth(userAuth *UserAuth, realm string, allowAnonymous bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsAuthenticated(c) {
			c.Next()
			return
		}

		username, password, ok := c.Request.BasicAuth()
		session := types.NewSession()
		if ok {
			user, e := userAuth.AuthByUsernamePassword(username, password)
			if e != nil {
				if !err.IsUnauthorizedError(e) {
					_ = c.Error(e)
					c.Abort()
					return
				}
			}
			session.User = user
		}

		if session.IsAnonymous() && !allowAnonymous {
			c.Status(http.StatusUnauthorized)
			c.Header("WWW-Authenticate", fmt.Sprintf("Basic realm=\""+realm+"\""))
			c.Abort()
			return
		}

		SetSession(c, session)
		c.Next()
	}
}

func UserGroupRequired(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := GetSession(c)
		if session.HasUserGroup(group) {
			c.Next()
			return
		}
		_ = c.Error(err.NewPermissionDeniedError(i18n.T("api.auth.group_permission_required", group)))
		c.Abort()
	}
}

func AdminGroupRequired() gin.HandlerFunc {
	return UserGroupRequired(types.AdminUserGroup)
}

func ExecuteTaskStreaming(c *gin.Context, runner task.Runner, runnable task.Runnable, options ...task.Option) error {
	completeChan := make(chan struct{})
	streamReady := make(chan struct{})
	task, e := runner.Execute(func(ctx types.TaskCtx) (interface{}, error) {
		<-streamReady
		defer close(completeChan)
		return runnable(ctx)
	}, options...)
	if e != nil {
		return e
	}
	ms := GetMessageSource(c)
	taskJson, e := json.Marshal(TranslateV(c, ms, task))
	if e != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, e)
		return nil
	}
	c.Header("X-Accel-Buffering", "no") // for nginx to not buffer the response
	c.Header(common.ResponseHeaderKey, string(taskJson))
	streamReady <- struct{}{}

	select {
	case <-c.Request.Context().Done():
		runner.StopTask(task.Id)
	case <-completeChan:
	}
	return nil
}

func SetResult(c *gin.Context, result interface{}) {
	c.Set(keyResult, result)
}

func GetResult(c *gin.Context) (interface{}, bool) {
	return c.Get(keyResult)
}

func GetToken(c *gin.Context) string {
	return c.GetString(keyToken)
}

func SetToken(c *gin.Context, token string) {
	c.Set(keyToken, token)
}

func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(keySession)
	return exists
}

func SetMessageSource(c *gin.Context, messageSource i18n.MessageSource) {
	c.Set(keyMessageSource, messageSource)
}

func GetMessageSource(c *gin.Context) i18n.MessageSource {
	if ms, exists := c.Get(keyMessageSource); exists {
		return ms.(i18n.MessageSource)
	}
	return nil
}

func GetSession(c *gin.Context) types.Session {
	if s, exists := c.Get(keySession); exists {
		return s.(types.Session)
	}
	return types.NewSession()
}

func SetSession(c *gin.Context, session types.Session) {
	c.Set(keySession, session)
}

func UpdateSession(c *gin.Context, tokenStore types.TokenStore, update func(session *types.Session)) error {
	session := GetSession(c)
	update(&session)
	if token := GetToken(c); token != "" {
		_, e := tokenStore.Update(token, session)
		if e != nil {
			return e
		}
	}
	SetSession(c, session)
	return nil
}

func TranslateV(c *gin.Context, ms i18n.MessageSource, v interface{}) interface{} {
	lang := c.GetHeader("accept-language")
	i := strings.IndexByte(lang, ',')
	if i >= 0 {
		lang = lang[:i]
	}
	return i18n.TranslateV(lang, ms, v)
}
