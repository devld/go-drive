package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/storage"
)

const (
	keyComponentsHolder = "componentsHolder"

	keyToken   = "token"
	keySession = "session"
	keyResult  = "apiResult"
)

type ComponentsHolder struct {
	TokenStore        TokenStore
	RequestSigner     *common.Signer
	RootDrive         *drive.RootDrive
	DriveStorage      *storage.DriveStorage
	UserStorage       *storage.UserStorage
	GroupStorage      *storage.GroupStorage
	PermissionStorage *storage.PathPermissionStorage
}

func GetComponentsHolder(c *gin.Context) *ComponentsHolder {
	return c.MustGet(keyComponentsHolder).(*ComponentsHolder)
}

func GetTokenStore(c *gin.Context) TokenStore {
	return GetComponentsHolder(c).TokenStore
}

func GetRequestSigner(c *gin.Context) *common.Signer {
	return GetComponentsHolder(c).RequestSigner
}

func GetRootDrive(c *gin.Context) *drive.RootDrive {
	return GetComponentsHolder(c).RootDrive
}

func GetDriveStorage(c *gin.Context) *storage.DriveStorage {
	return GetComponentsHolder(c).DriveStorage
}

func GetUserStorage(c *gin.Context) *storage.UserStorage {
	return GetComponentsHolder(c).UserStorage
}

func GetGroupStorage(c *gin.Context) *storage.GroupStorage {
	return GetComponentsHolder(c).GroupStorage
}

func GetPermissionStorage(c *gin.Context) *storage.PathPermissionStorage {
	return GetComponentsHolder(c).PermissionStorage
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

func UpdateSessionUser(c *gin.Context, user types.User) error {
	session := GetSession(c)
	session.User = user
	_, e := GetTokenStore(c).Update(GetToken(c), session)
	return e
}
