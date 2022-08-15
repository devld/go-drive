package server

import (
	"go-drive/common/types"

	"github.com/gin-gonic/gin"
)

func InitAuthRoutes(r gin.IRouter, ua *UserAuth, tokenStore types.TokenStore) error {
	ar := authRoute{userAuth: ua, tokenStore: tokenStore}

	r.POST("/auth/init", ar.init)

	auth := r.Group("/auth", TokenAuth(tokenStore))
	{
		auth.POST("/login", ar.login)
		auth.POST("/logout", ar.logout)
		auth.GET("/user", ar.getUser)
	}

	return nil
}

type authRoute struct {
	userAuth   *UserAuth
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
	user, e := a.userAuth.AuthByUsernamePassword(user.Username, user.Password)
	if e != nil {
		_ = c.Error(e)
		return
	}
	e = UpdateSessionUser(c, a.tokenStore, user)
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
		u.RootPath = ""
		SetResult(c, u)
	}
}
