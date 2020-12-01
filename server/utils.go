package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"net/http"
	"strings"
	"time"
)

const (
	keyToken   = "token"
	keySession = "session"
	keyResult  = "apiResult"

	signatureQueryKey = "_k"
)

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

func getSignPayload(req *http.Request, path string) string {
	return req.Host + "." + path + "." + utils.GetRealIP(req)
}

func checkSignature(signer *utils.Signer, req *http.Request, path string) bool {
	return signer.Validate(getSignPayload(req, path), req.URL.Query().Get(signatureQueryKey))
}

func signPathRequest(signer *utils.Signer, req *http.Request, path string, notAfter time.Time) string {
	return signer.Sign(getSignPayload(req, path), notAfter)
}

func TranslateV(c *gin.Context, ms i18n.MessageSource, v interface{}) interface{} {
	lang := c.GetHeader("accept-language")
	i := strings.IndexByte(lang, ',')
	if i >= 0 {
		lang = lang[:i]
	}
	return i18n.TranslateV(lang, ms, v)
}
