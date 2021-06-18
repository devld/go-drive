package server

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/storage"
	"golang.org/x/crypto/bcrypt"
)

type UserAuth struct {
	userDAO *storage.UserDAO
}

func NewUserAuth(userDao *storage.UserDAO) *UserAuth {
	return &UserAuth{userDAO: userDao}
}

func (ua *UserAuth) AuthByUsernamePassword(username, password string) (types.User, error) {
	getUser, e := ua.userDAO.GetUser(username)
	if e != nil {
		if err.IsNotFoundError(e) {
			return types.User{},
				err.NewUnauthorizedError(i18n.T("api.auth.invalid_username_or_password"))
		} else {
			return types.User{}, e
		}
	}
	if e := bcrypt.CompareHashAndPassword([]byte(getUser.Password), []byte(password)); e != nil {
		return types.User{},
			err.NewUnauthorizedError(i18n.T("api.auth.invalid_username_or_password"))
	}
	return getUser, nil
}
