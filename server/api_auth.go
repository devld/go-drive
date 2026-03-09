package server

import (
	"net/http"

	"go-drive/common/types"
	"go-drive/server/auth"
	"time"

	"github.com/gin-gonic/gin"
)

func InitAuthRoutes(r gin.IRouter, ua *auth.UserAuth,
	tokenStore types.TokenStore, failBan *FailBanGroup) error {

	ar := authRoute{ua, tokenStore}

	r.POST("/auth/init", ar.init)

	r.GET("/auth/start", ar.getStart)

	r.GET("/auth/callback/:provider", ar.callback)

	authGroup := r.Group("/auth", TokenAuth(tokenStore))
	{
		authGroup.POST(
			"/login",
			failBan.LimiterByIP("/login", 5*time.Minute, 5),
			ar.login,
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

func (a *authRoute) init(c *gin.Context) {
	token, e := a.tokenStore.Create(types.NewSession())
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, token)
}

func (a *authRoute) getStart(c *gin.Context) {
	forms := a.userAuth.GetForms(c.Request)
	SetResult(c, forms)
}

func (a *authRoute) callback(c *gin.Context) {
	provider := c.Param("provider")
	formData := types.SM{}
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			formData[k] = v[0]
		}
	}
	user, e := a.userAuth.AuthenticateCallback(provider, c.Request, formData)
	if e != nil {
		_ = c.Error(e)
		return
	}
	session := types.NewSession()
	session.User = user
	token, e := a.tokenStore.Create(session)
	if e != nil {
		_ = c.Error(e)
		return
	}
	c.Redirect(http.StatusFound, "/?token="+token.Token)
}

func (a *authRoute) login(c *gin.Context) {
	var formData types.SM
	if e := c.ShouldBindJSON(&formData); e != nil {
		_ = c.Error(e)
		return
	}
	user, e := a.userAuth.AuthenticateForm(c.Request, formData)
	if e != nil {
		_ = c.Error(e)
		return
	}
	e = UpdateSession(c, a.tokenStore, func(session *types.Session) { session.User = user })
	if e != nil {
		_ = c.Error(e)
	}
}

func (a *authRoute) logout(c *gin.Context) {
	_ = UpdateSession(c, a.tokenStore, func(session *types.Session) { session.User = types.User{} })
}

func (a *authRoute) getUser(c *gin.Context) {
	s := GetSession(c)
	if !s.IsAnonymous() {
		u := s.User
		u.Password = ""
		u.RootPath = ""
		SetResult(c, u)
	} else {
		SetResult(c, nil)
	}
}
