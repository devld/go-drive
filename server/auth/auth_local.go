package auth

import (
	"net/http"

	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"

	"golang.org/x/crypto/bcrypt"
)

type localAuthProvider struct {
	userDAO *storage.UserDAO
}

func newLocalAuthProvider(ch *registry.ComponentsHolder) *localAuthProvider {
	userDAO := ch.Get("userDAO").(*storage.UserDAO)
	return &localAuthProvider{userDAO: userDAO}
}

func (p *localAuthProvider) EntryPoint(r *http.Request) (*AuthForm, error) {
	return &AuthForm{
		Provider:    "local",
		DisplayName: "Login",
		Type:        AuthTypeForm,
		Form: []types.FormItem{
			{Field: "username", Label: "Username", Type: "text", Required: true},
			{Field: "password", Label: "Password", Type: "password", Required: true},
		},
	}, nil
}

func (p *localAuthProvider) Callback(r *http.Request, formData types.SM) (types.User, error) {
	username := formData["username"]
	password := formData["password"]
	if username == "" || password == "" {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}
	getUser, e := p.userDAO.GetUser(username)
	if e != nil {
		if err.IsNotFoundError(e) {
			return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
		}
		return types.User{}, e
	}
	if e := bcrypt.CompareHashAndPassword([]byte(getUser.Password), []byte(password)); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}
	return getUser, nil
}
