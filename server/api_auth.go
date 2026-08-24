package server

import (
	"go-drive/common/logging"
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
		logging.For("auth").Debugf("authentication start failed provider=%s: %v", logging.Sanitize(provider), e)
		_ = c.Error(e)
		return
	}
	logging.For("auth").Debugf("authentication start provider=%s", logging.Sanitize(provider))
	SetResult(c, result)
}

func (a *authRoute) callback(c *gin.Context) {
	provider := c.Param("provider")
	user, e := a.userAuth.AuthenticateCallback(provider, c.Request, readAuthFormData(c))
	if e != nil {
		logging.For("auth").Debugf("authentication failed provider=%s: %v", logging.Sanitize(provider), e)
		_ = c.Error(e)
		return
	}
	token, e := a.tokenStore.Create(types.Principal{User: user, AuthType: types.AuthTypeToken})
	if e != nil {
		logging.For("auth").Errorf("session creation failed provider=%s user=%s: %v",
			logging.Sanitize(provider), logging.Sanitize(user.Username), e)
		_ = c.Error(e)
		return
	}
	logging.For("auth").Debugf("authentication succeeded provider=%s user=%s",
		logging.Sanitize(provider), logging.Sanitize(user.Username))
	SetResult(c, token)
}

// readAuthFormData reads the submitted credentials/parameters from the JSON body.
func readAuthFormData(c *gin.Context) types.SM {
	formData := types.SM{}
	if e := c.ShouldBindJSON(&formData); e != nil {
		logging.For("auth").Debugf("authentication form parse failed provider=%s: %v",
			logging.Sanitize(c.Param("provider")), e)
	}
	return formData
}

func (a *authRoute) logout(c *gin.Context) {
	if token := GetToken(c); token != "" {
		if e := a.tokenStore.Revoke(token); e != nil {
			logging.For("auth").Warnf("session revoke failed: %v", e)
		} else {
			logging.For("auth").Debugf("session revoked")
		}
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
