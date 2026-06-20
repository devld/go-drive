package server

import (
	"go-drive/common/types"
	"time"

	"github.com/gin-gonic/gin"
)

func InitAuthRoutes(r gin.IRouter, ua *UserAuth,
	tokenStore types.TokenStore, failBan *FailBanGroup) error {

	ar := authRoute{ua, tokenStore}

	auth := r.Group("/auth", TokenAuth(tokenStore))
	{
		auth.POST(
			"/login",
			failBan.LimiterByIP("/login", 5*time.Minute, 5),
			ar.login,
		)

		auth.POST("/logout", ar.logout)
		auth.GET("/user", ar.getUser)
	}

	return nil
}

type authRoute struct {
	userAuth   *UserAuth
	tokenStore types.TokenStore
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
	// rotate: drop any token carried into this request before issuing a new one
	if old := GetToken(c); old != "" {
		_ = a.tokenStore.Revoke(old)
	}
	principal := types.Principal{User: user, AuthType: types.AuthTypeToken}
	token, e := a.tokenStore.Create(principal)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, token)
}

func (a *authRoute) logout(c *gin.Context) {
	if token := GetToken(c); token != "" {
		_ = a.tokenStore.Revoke(token)
	}
}

func (a *authRoute) getUser(c *gin.Context) {
	principal := GetPrincipal(c)
	if !principal.IsAnonymous() {
		u := principal.User
		u.Password = ""
		u.RootPath = ""
		SetResult(c, u)
	} else {
		SetResult(c, nil)
	}
}
