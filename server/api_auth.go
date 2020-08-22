package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"golang.org/x/crypto/bcrypt"
)

const (
	headerAuth = "Authorization"
)

func InitAuthRoutes(r gin.IRouter) {

	r.POST("/auth/init", initAuth)

	auth := r.Group("/auth", Auth())

	auth.POST("/login", login)
	auth.POST("/logout", logout)

}

func initAuth(c *gin.Context) {
	token, e := GetTokenStore(c).Create(types.Session{})
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, token)
}

func login(c *gin.Context) {
	user := types.User{}
	if e := c.Bind(&user); e != nil {
		_ = c.Error(e)
		return
	}
	getUser, e := GetUserStorage(c).GetUser(user.Username)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if e := bcrypt.CompareHashAndPassword([]byte(getUser.Password), []byte(user.Password)); e != nil {
		_ = c.Error(common.NewBadRequestError("invalid username or password"))
		return
	}
	e = UpdateSessionUser(c, getUser)
	if e != nil {
		_ = c.Error(e)
	}
}

func logout(c *gin.Context) {
	_ = UpdateSessionUser(c, types.User{})
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenKey := c.GetHeader(headerAuth)
		tokenRaw, e := GetTokenStore(c).Validate(tokenKey)
		if e != nil {
			_ = c.Error(e)
			c.Abort()
			return
		}
		token := tokenRaw.(Token)
		session := token.Value.(types.Session)

		SetToken(c, token.Token)
		SetSession(c, session)

		c.Next()
	}
}

func UserGroupRequired(group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := GetSession(c)
		if session.User.Groups != nil {
			for _, g := range session.User.Groups {
				if g.Name == group {
					c.Next()
					return
				}
			}
		}
		_ = c.Error(common.NewPermissionDeniedError(fmt.Sprintf("permission of group '%s' required", group)))
		c.Abort()
	}
}
