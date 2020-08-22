package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common/types"
	"go-drive/storage"
)

const (
	keyTokenStore   = "tokenStore"
	keyDriveStorage = "driveStorage"
	keyUserStorage  = "userStorage"

	keyToken   = "token"
	keySession = "session"
	keyResult  = "apiResult"
)

func GetTokenStore(c *gin.Context) TokenStore {
	return c.MustGet(keyTokenStore).(TokenStore)
}

func GetDriveStorage(c *gin.Context) *storage.DriveStorage {
	return c.MustGet(keyDriveStorage).(*storage.DriveStorage)
}

func GetUserStorage(c *gin.Context) *storage.UserStorage {
	return c.MustGet(keyUserStorage).(*storage.UserStorage)
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
	return c.MustGet(keySession).(types.Session)
}

func SetSession(c *gin.Context, session types.Session) {
	c.Set(keySession, session)
}

func UpdateSessionUser(c *gin.Context, user types.User) error {
	session := GetSession(c)
	session.User = user
	_, e := GetTokenStore(c).Update(GetToken(c), session)
	return e
}
