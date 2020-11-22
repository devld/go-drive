package server

import (
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/storage"
	"net/http"
	"time"
)

const (
	keyToken   = "token"
	keySession = "session"
	keyResult  = "apiResult"

	signatureQueryKey = "_k"
)

func TokenStore() types.TokenStore {
	return common.R().Get("tokenStore").(types.TokenStore)
}

func Signer() *common.Signer {
	return common.R().Get("signer").(*common.Signer)
}

func RootDrive() *drive.RootDrive {
	return common.R().Get("rootDrive").(*drive.RootDrive)
}

func DriveDAO() *storage.DriveDAO {
	return common.R().Get("driveDAO").(*storage.DriveDAO)
}

func UserDAO() *storage.UserDAO {
	return common.R().Get("userDAO").(*storage.UserDAO)
}

func GroupDAO() *storage.GroupDAO {
	return common.R().Get("groupDAO").(*storage.GroupDAO)
}

func PermissionDAO() *storage.PathPermissionDAO {
	return common.R().Get("pathPermissionDAO").(*storage.PathPermissionDAO)
}

func PathMountDAO() *storage.PathMountDAO {
	return common.R().Get("pathMountDAO").(*storage.PathMountDAO)
}

func DriveCacheDAO() *storage.DriveCacheDAO {
	return common.R().Get("driveCacheDAO").(*storage.DriveCacheDAO)
}

func DriveDataDAO() *storage.DriveDataDAO {
	return common.R().Get("driveData").(*storage.DriveDataDAO)
}

func TaskRunner() task.Runner {
	return common.R().Get("taskRunner").(task.Runner)
}

func GetChunkUploader() *ChunkUploader {
	return common.R().Get("chunkUploader").(*ChunkUploader)
}

func GetThumbnail() *Thumbnail {
	return common.R().Get("thumbnail").(*Thumbnail)
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
	_, e := TokenStore().Update(GetToken(c), session)
	return e
}

func getSignPayload(req *http.Request, path string) string {
	return req.Host + "." + path + "." + common.GetRealIP(req)
}

func checkSignature(req *http.Request, path string) bool {
	return Signer().Validate(getSignPayload(req, path), req.URL.Query().Get(signatureQueryKey))
}

func signPathRequest(req *http.Request, path string, notAfter time.Time) string {
	return Signer().Sign(getSignPayload(req, path), notAfter)
}
