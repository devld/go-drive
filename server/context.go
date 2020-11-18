package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/task"
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
	TokenStore    types.TokenStore
	RequestSigner *common.Signer

	RootDrive *drive.RootDrive

	DriveStorage *storage.DriveStorage
	UserStorage  *storage.UserStorage
	GroupStorage *storage.GroupStorage

	PermissionStorage *storage.PathPermissionStorage
	PathMountStorage  *storage.PathMountStorage
	DriveCacheStorage *storage.DriveCacheStorage
	DriveDataStorage  *storage.DriveDataStorage

	TaskRunner    task.Runner
	ChunkUploader *ChunkUploader
}

func getComponentsHolder(c *gin.Context) *ComponentsHolder {
	return c.MustGet(keyComponentsHolder).(*ComponentsHolder)
}

func GetTokenStore(c *gin.Context) types.TokenStore {
	return getComponentsHolder(c).TokenStore
}

func GetRequestSigner(c *gin.Context) *common.Signer {
	return getComponentsHolder(c).RequestSigner
}

func GetRootDrive(c *gin.Context) *drive.RootDrive {
	return getComponentsHolder(c).RootDrive
}

func GetDriveStorage(c *gin.Context) *storage.DriveStorage {
	return getComponentsHolder(c).DriveStorage
}

func GetUserStorage(c *gin.Context) *storage.UserStorage {
	return getComponentsHolder(c).UserStorage
}

func GetGroupStorage(c *gin.Context) *storage.GroupStorage {
	return getComponentsHolder(c).GroupStorage
}

func GetPermissionStorage(c *gin.Context) *storage.PathPermissionStorage {
	return getComponentsHolder(c).PermissionStorage
}

func GetPathMountStorage(c *gin.Context) *storage.PathMountStorage {
	return getComponentsHolder(c).PathMountStorage
}

func GetDriveCacheStorage(c *gin.Context) *storage.DriveCacheStorage {
	return getComponentsHolder(c).DriveCacheStorage
}

func GetDriveDataStorage(c *gin.Context) *storage.DriveDataStorage {
	return getComponentsHolder(c).DriveDataStorage
}

func GetTaskRunner(c *gin.Context) task.Runner {
	return getComponentsHolder(c).TaskRunner
}

func GetChunkUploader(c *gin.Context) *ChunkUploader {
	return getComponentsHolder(c).ChunkUploader
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
