package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-drive/common"
	"go-drive/common/types"
	"go-drive/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	headerAuth = "Authorization"
)

func InitAuthRoutes(r gin.IRouter, tokenStore types.TokenStore, userDAO *storage.UserDAO) {
	ar := authRoute{
		userDAO:    userDAO,
		tokenStore: tokenStore,
	}

	r.POST("/auth/init", ar.init)

	auth := r.Group("/auth", Auth(tokenStore))
	{
		auth.POST("/login", ar.login)
		auth.POST("/logout", ar.logout)
		auth.GET("/user", ar.getUser)
	}
}

type authRoute struct {
	userDAO    *storage.UserDAO
	tokenStore types.TokenStore
}

func (a *authRoute) init(c *gin.Context) {
	token, e := a.tokenStore.Create(types.Session{})
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, token)
}

func (a *authRoute) login(c *gin.Context) {
	user := types.User{}
	if e := c.Bind(&user); e != nil {
		_ = c.Error(e)
		return
	}
	getUser, e := a.userDAO.GetUser(user.Username)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if e := bcrypt.CompareHashAndPassword([]byte(getUser.Password), []byte(user.Password)); e != nil {
		_ = c.Error(common.NewBadRequestError("invalid username or password"))
		return
	}
	e = UpdateSessionUser(c, a.tokenStore, getUser)
	if e != nil {
		_ = c.Error(e)
	}
}

func (a *authRoute) logout(c *gin.Context) {
	_ = UpdateSessionUser(c, a.tokenStore, types.User{})
}

func (a *authRoute) getUser(c *gin.Context) {
	s := GetSession(c)
	if !s.IsAnonymous() {
		u := s.User
		u.Password = ""
		SetResult(c, u)
	}
}

func Auth(tokenStore types.TokenStore) gin.HandlerFunc {
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
