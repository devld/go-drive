package server

import (
	"time"

	"go-drive/common/types"
	"go-drive/server/auth"

	"github.com/gin-gonic/gin"
)

func InitAuthRoutes(r gin.IRouter, ua *auth.UserAuth,
	tokenStore types.TokenStore, failBan *FailBanGroup) error {

	ar := authRoute{ua, tokenStore}

	authGroup := r.Group("/auth", TokenAuth(tokenStore))
	{
		authGroup.POST("/:provider/start", ar.start)

		authGroup.POST(
			"/:provider/callback",
			failBan.LimiterByIP("/auth/callback", 5*time.Minute, 5),
			ar.callback,
		)

		authGroup.POST("/logout", ar.logout)
		authGroup.GET("/user", ar.getUser)
	}

	return nil
}

type authRoute struct {
	userAuth   *auth.UserAuth
	tokenStore types.TokenStore
}

func (a *authRoute) start(c *gin.Context) {
	provider := c.Param("provider")
	result, e := a.userAuth.Start(provider, c.Request, readAuthFormData(c))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, result)
}

func (a *authRoute) callback(c *gin.Context) {
	provider := c.Param("provider")
	user, e := a.userAuth.AuthenticateCallback(provider, c.Request, readAuthFormData(c))
	if e != nil {
		_ = c.Error(e)
		return
	}
	token, e := a.tokenStore.Create(types.Principal{User: user, AuthType: types.AuthTypeToken})
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, token)
}

// readAuthFormData reads the submitted credentials/parameters from the JSON body.
func readAuthFormData(c *gin.Context) types.SM {
	formData := types.SM{}
	_ = c.ShouldBindJSON(&formData)
	return formData
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
