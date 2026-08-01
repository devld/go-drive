package server

import (
	"fmt"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/storage"
	"mime"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	headerWOPIOverride          = "X-WOPI-Override"
	headerWOPILock              = "X-WOPI-Lock"
	headerWOPIOldLock           = "X-WOPI-OldLock"
	headerWOPILockFailureReason = "X-WOPI-LockFailureReason"
	headerWOPIItemVersion       = "X-WOPI-ItemVersion"
)

type wopiRoute struct {
	service *wopiService
}

type wopiSessionResponse struct {
	ActionURL      string `json:"actionUrl"`
	AccessToken    string `json:"accessToken"`
	AccessTokenTTL int64  `json:"accessTokenTtl"`
	Mode           string `json:"mode"`
	UserID         string `json:"userId"`
	OwnerID        string `json:"ownerId"`
}

func InitWOPIRoutes(router gin.IRouter, config common.Config, access *drive.Access,
	tokenStore types.TokenStore, userDAO *storage.UserDAO,
	ch *registry.ComponentsHolder) error {
	service, e := newWOPIService(config, access, userDAO, ch)
	if e != nil {
		return e
	}
	route := &wopiRoute{service: service}

	wopi := router.Group("/wopi")
	wopi.POST("/session/*path", TokenAuth(tokenStore), route.createSession)
	wopi.GET("/files/:id", route.checkFileInfo)
	wopi.GET("/files/:id/contents", route.getFile)
	wopi.POST("/files/:id/contents", route.putFile)
	wopi.POST("/files/:id", route.fileOperation)
	return nil
}

func (r *wopiRoute) createSession(c *gin.Context) {
	principal := GetPrincipal(c)
	if principal.IsAnonymous() {
		c.AbortWithStatusJSON(http.StatusUnauthorized, types.M{"message": "authentication required"})
		return
	}
	origin, e := validateWOPIOrigin(c.GetHeader("Origin"), c.Request.Host)
	if e != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, types.M{"message": e.Error()})
		return
	}
	path := utils.CleanPath(c.Param("path"))
	d, e := r.service.access.GetDrive(principal)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	entry, e := d.Get(c.Request.Context(), path)
	if e != nil || !entry.Type().IsFile() || !entry.Meta().Readable {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	discovery, e := r.service.discovery.get(c.Request.Context())
	if e != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, types.M{"message": e.Error()})
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(pathpkg.Ext(entry.Name()), "."))
	action, ok := discovery.action(ext, entry.Meta().Writable)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, types.M{"message": "no WOPI action for this file type"})
		return
	}
	session, token, e := r.service.createSession(
		principal.User.Username, path, origin, canonicalWOPIResourceKey(entry), action.Name == "edit", time.Time{},
	)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	actionURL, e := makeWOPIActionURL(action.URLSrc, r.service.fileURL(origin, session.id))
	if e != nil {
		r.service.deleteSession(session.id)
		c.AbortWithStatusJSON(http.StatusBadGateway, types.M{"message": e.Error()})
		return
	}
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "-1")
	c.JSON(http.StatusOK, wopiSessionResponse{
		ActionURL:      actionURL,
		AccessToken:    token,
		AccessTokenTTL: session.expiresAt.UnixMilli(),
		Mode:           action.Name,
		UserID:         principal.User.Username,
		OwnerID:        principal.User.Username,
	})
}

func (r *wopiRoute) authenticated(c *gin.Context) (wopiSession, types.IDrive, types.IEntry, bool) {
	session, ok := r.service.validateSession(c.Param("id"), c.Query("access_token"))
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return wopiSession{}, nil, nil, false
	}
	principal, e := r.service.principal(session)
	if e != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return wopiSession{}, nil, nil, false
	}
	d, e := r.service.access.GetDrive(principal)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return wopiSession{}, nil, nil, false
	}
	entry, e := d.Get(c.Request.Context(), session.path)
	if e != nil || !entry.Type().IsFile() || !entry.Meta().Readable {
		c.AbortWithStatus(http.StatusNotFound)
		return wopiSession{}, nil, nil, false
	}
	return session, d, entry, true
}

func (r *wopiRoute) checkFileInfo(c *gin.Context) {
	session, _, entry, ok := r.authenticated(c)
	if !ok {
		return
	}
	discovery, e := r.service.discovery.get(c.Request.Context())
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(pathpkg.Ext(entry.Name()), "."))
	_, editable := discovery.actions[ext]["edit"]
	canWrite := session.writable && entry.Meta().Writable && editable
	username := session.username
	c.Header("Cache-Control", "no-cache, no-store")
	c.JSON(http.StatusOK, types.M{
		"BaseFileName":               entry.Name(),
		"LastModifiedTime":           wopiLastModifiedTime(entry),
		"OwnerId":                    username,
		"Size":                       entry.Size(),
		"UserId":                     username,
		"UserFriendlyName":           username,
		"Version":                    wopiVersion(entry),
		"ReadOnly":                   !canWrite,
		"UserCanWrite":               canWrite,
		"UserCanNotWriteRelative":    !canWrite,
		"SupportsUpdate":             canWrite,
		"SupportsLocks":              canWrite,
		"SupportsGetLock":            canWrite,
		"SupportsExtendedLockLength": true,
	})
}

func (r *wopiRoute) getFile(c *gin.Context) {
	_, _, entry, ok := r.authenticated(c)
	if !ok {
		return
	}
	maxExpected := int64(1<<31 - 1)
	if raw := c.GetHeader("X-WOPI-MaxExpectedSize"); raw != "" {
		parsed, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || parsed < 0 {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		maxExpected = parsed
	}
	if entry.Size() > maxExpected {
		c.AbortWithStatus(http.StatusPreconditionFailed)
		return
	}
	reader, e := entry.GetReader(c.Request.Context(), -1, -1)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer func() { _ = reader.Close() }()
	contentType := mime.TypeByExtension(pathpkg.Ext(entry.Name()))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header(headerWOPIItemVersion, wopiVersion(entry))
	c.DataFromReader(http.StatusOK, entry.Size(), contentType, reader, nil)
}

func (r *wopiRoute) fileOperation(c *gin.Context) {
	session, d, entry, ok := r.authenticated(c)
	if !ok {
		return
	}
	switch strings.ToUpper(c.GetHeader(headerWOPIOverride)) {
	case "LOCK":
		if !session.writable || !entry.Meta().Writable {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		r.lock(c, session, entry)
	case "REFRESH_LOCK":
		if !session.writable || !entry.Meta().Writable {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		r.refreshLock(c, session)
	case "UNLOCK":
		if !session.writable || !entry.Meta().Writable {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		r.unlock(c, session, entry)
	case "GET_LOCK":
		r.getLock(c, session)
	case "PUT_RELATIVE":
		r.putRelativeFile(c, session, d, entry)
	default:
		c.AbortWithStatus(http.StatusNotImplemented)
	}
}

func validWOPILock(value string) bool {
	if value == "" || len(value) > 1024 || !utf8.ValidString(value) {
		return false
	}
	for _, ch := range value {
		if ch > 127 {
			return false
		}
	}
	return true
}

func writeWOPILockMismatch(c *gin.Context, current, reason string) {
	c.Header(headerWOPILock, current)
	if reason != "" {
		c.Header(headerWOPILockFailureReason, reason)
	}
	c.AbortWithStatus(http.StatusConflict)
}

func (r *wopiRoute) lock(c *gin.Context, session wopiSession, entry types.IEntry) {
	value := c.GetHeader(headerWOPILock)
	oldValue := c.GetHeader(headerWOPIOldLock)
	if !validWOPILock(value) || (oldValue != "" && !validWOPILock(oldValue)) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	r.service.resourceLock.Lock(session.resourceKey)
	defer r.service.resourceLock.UnLock(session.resourceKey)
	current, exists := r.service.currentLock(session.resourceKey, time.Now())
	if oldValue != "" {
		if !exists || current.value != oldValue {
			writeWOPILockMismatch(c, current.value, "lock mismatch")
			return
		}
		current.value = value
		current.expiresAt = time.Now().Add(wopiLockTTL)
		r.service.setLock(session.resourceKey, current)
		c.Header(headerWOPIItemVersion, wopiVersion(entry))
		c.Status(http.StatusOK)
		return
	}
	if exists && current.value != value {
		writeWOPILockMismatch(c, current.value, "lock mismatch")
		return
	}
	if !exists {
		current = wopiLock{value: value, version: wopiVersion(entry)}
	}
	current.expiresAt = time.Now().Add(wopiLockTTL)
	r.service.setLock(session.resourceKey, current)
	c.Header(headerWOPIItemVersion, wopiVersion(entry))
	c.Status(http.StatusOK)
}

func (r *wopiRoute) refreshLock(c *gin.Context, session wopiSession) {
	value := c.GetHeader(headerWOPILock)
	if !validWOPILock(value) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	r.service.resourceLock.Lock(session.resourceKey)
	defer r.service.resourceLock.UnLock(session.resourceKey)
	current, exists := r.service.currentLock(session.resourceKey, time.Now())
	if !exists || current.value != value {
		writeWOPILockMismatch(c, current.value, "lock mismatch")
		return
	}
	current.expiresAt = time.Now().Add(wopiLockTTL)
	r.service.setLock(session.resourceKey, current)
	c.Status(http.StatusOK)
}

func (r *wopiRoute) unlock(c *gin.Context, session wopiSession, entry types.IEntry) {
	value := c.GetHeader(headerWOPILock)
	if !validWOPILock(value) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	r.service.resourceLock.Lock(session.resourceKey)
	defer r.service.resourceLock.UnLock(session.resourceKey)
	current, exists := r.service.currentLock(session.resourceKey, time.Now())
	if !exists || current.value != value {
		writeWOPILockMismatch(c, current.value, "lock mismatch")
		return
	}
	r.service.deleteLock(session.resourceKey)
	c.Header(headerWOPIItemVersion, wopiVersion(entry))
	c.Status(http.StatusOK)
}

func (r *wopiRoute) getLock(c *gin.Context, session wopiSession) {
	r.service.resourceLock.Lock(session.resourceKey)
	defer r.service.resourceLock.UnLock(session.resourceKey)
	current, _ := r.service.currentLock(session.resourceKey, time.Now())
	c.Header(headerWOPILock, current.value)
	c.Status(http.StatusOK)
}

func (r *wopiRoute) putFile(c *gin.Context) {
	session, d, entry, ok := r.authenticated(c)
	if !ok {
		return
	}
	if !session.writable || !entry.Meta().Writable {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if !strings.EqualFold(c.GetHeader(headerWOPIOverride), "PUT") {
		c.AbortWithStatus(http.StatusNotImplemented)
		return
	}
	tempFile, size, e := ReadRequestBodyToTempFile(c, r.service.config.TempDir)
	if e != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if !validateWOPIBodySize(c, size) {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	r.service.resourceLock.Lock(session.resourceKey)
	defer r.service.resourceLock.UnLock(session.resourceKey)
	entry, e = d.Get(c.Request.Context(), session.path)
	if e != nil || !entry.Type().IsFile() || !entry.Meta().Readable {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if !entry.Meta().Writable {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	current, locked := r.service.currentLock(session.resourceKey, time.Now())
	requestedLock := c.GetHeader(headerWOPILock)
	if !locked {
		if entry.Size() != 0 {
			writeWOPILockMismatch(c, "", "file is not locked")
			return
		}
	} else if requestedLock == "" || current.value != requestedLock {
		writeWOPILockMismatch(c, current.value, "lock mismatch")
		return
	} else if canonicalWOPIResourceKey(entry) != session.resourceKey || current.version != wopiVersion(entry) {
		writeWOPILockMismatch(c, current.value, "file changed outside WOPI")
		return
	}

	saved, e := d.Save(task.NewContextWrapper(c.Request.Context()), session.path, size, true, tempFile)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	version := wopiVersion(saved)
	if locked {
		current.version = version
		current.expiresAt = time.Now().Add(wopiLockTTL)
		r.service.setLock(session.resourceKey, current)
	}
	c.Header(headerWOPIItemVersion, version)
	c.JSON(http.StatusOK, types.M{"LastModifiedTime": wopiLastModifiedTime(saved)})
}

func wopiLastModifiedTime(entry types.IEntry) string {
	return time.UnixMilli(entry.ModTime()).UTC().Format(time.RFC3339Nano)
}

func validateWOPIBodySize(c *gin.Context, size int64) bool {
	raw := c.GetHeader("X-WOPI-Size")
	if raw == "" {
		return true
	}
	expected, e := strconv.ParseInt(raw, 10, 64)
	if e != nil || expected < 0 || expected != size {
		c.AbortWithStatus(http.StatusBadRequest)
		return false
	}
	return true
}

func sanitizeWOPISuggestedName(name string) string {
	name = strings.Map(func(ch rune) rune {
		if ch < 32 || strings.ContainsRune("/\\\\\x00:*\"<>|", ch) {
			return '_'
		}
		return ch
	}, name)
	name = strings.Trim(name, " .")
	if name == "" {
		return "document"
	}
	return name
}

func (r *wopiRoute) putRelativeFile(c *gin.Context, session wopiSession,
	d types.IDrive, source types.IEntry) {
	if !session.writable || !source.Meta().Writable {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	relativeTarget := c.GetHeader("X-WOPI-RelativeTarget")
	suggestedTarget := c.GetHeader("X-WOPI-SuggestedTarget")
	if (relativeTarget == "") == (suggestedTarget == "") {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	name := relativeTarget
	if suggestedTarget != "" {
		name = suggestedTarget
		if strings.HasPrefix(name, ".") {
			base := strings.TrimSuffix(source.Name(), pathpkg.Ext(source.Name()))
			name = base + name
		}
		name = sanitizeWOPISuggestedName(name)
	}
	targetPath, e := r.service.relativePath(session, name)
	if e != nil {
		c.Header("X-WOPI-InvalidFileNameError", e.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if suggestedTarget != "" {
		targetPath, e = driveutil.FindNonExistsEntryName(c.Request.Context(), d, targetPath)
		if e != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
	overwrite := strings.EqualFold(c.GetHeader("X-WOPI-OverwriteRelativeTarget"), "true")
	if suggestedTarget != "" {
		overwrite = false
	}

	var targetLockKey string
	if existing, getError := d.Get(c.Request.Context(), targetPath); getError == nil {
		targetLockKey = canonicalWOPIResourceKey(existing)
		r.service.resourceLock.Lock(targetLockKey)
		defer r.service.resourceLock.UnLock(targetLockKey)
		if current, locked := r.service.currentLock(targetLockKey, time.Now()); locked {
			writeWOPILockMismatch(c, current.value, "relative target is locked")
			return
		}
		if !overwrite {
			writeWOPILockMismatch(c, "", "relative target exists")
			return
		}
	}
	tempFile, size, e := ReadRequestBodyToTempFile(c, r.service.config.TempDir)
	if e != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if !validateWOPIBodySize(c, size) {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	saved, e := d.Save(task.NewContextWrapper(c.Request.Context()), targetPath, size, overwrite, tempFile)
	if e != nil {
		if suggestedTarget != "" {
			c.AbortWithStatus(http.StatusInternalServerError)
		} else {
			writeWOPILockMismatch(c, "", "relative target could not be saved")
		}
		return
	}
	newSession, token, e := r.service.createSession(
		session.username, targetPath, session.origin, canonicalWOPIResourceKey(saved), session.writable, session.expiresAt,
	)
	if e != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	wopiSrc := r.service.fileURL(session.origin, newSession.id)
	query := url.Values{
		"access_token":     []string{token},
		"access_token_ttl": []string{strconv.FormatInt(newSession.expiresAt.UnixMilli(), 10)},
	}
	c.JSON(http.StatusOK, types.M{
		"Name": saved.Name(),
		"Url":  fmt.Sprintf("%s?%s", wopiSrc, query.Encode()),
	})
}
