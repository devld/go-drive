package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"net/http"
	"strings"
)

const (
	keyToken   = "token"
	keySession = "session"
	keyResult  = "apiResult"

	headerAuth = "Authorization"
)

func TokenAuth(tokenStore types.TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenKey := c.GetHeader(headerAuth)
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
		username, password, ok := c.Request.BasicAuth()
		session := types.Session{}
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
	return UserGroupRequired("admin")
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

func GetSession(c *gin.Context) types.Session {
	if s, exists := c.Get(keySession); exists {
		return s.(types.Session)
	}
	return types.Session{}
}

func SetSession(c *gin.Context, session types.Session) {
	c.Set(keySession, session)
}

func UpdateSessionUser(c *gin.Context, tokenStore types.TokenStore, user types.User) error {
	session := GetSession(c)
	session.User = user
	_, e := tokenStore.Update(GetToken(c), session)
	return e
}

func TranslateV(c *gin.Context, ms i18n.MessageSource, v interface{}) interface{} {
	lang := c.GetHeader("accept-language")
	i := strings.IndexByte(lang, ',')
	if i >= 0 {
		lang = lang[:i]
	}
	return i18n.TranslateV(lang, ms, v)
}
