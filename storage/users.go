package storage

import (
	"github.com/jinzhu/gorm"
	"go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"golang.org/x/crypto/bcrypt"
)

type UserDAO struct {
	db *DB
}

func NewUserDAO(db *DB) *UserDAO {
	return &UserDAO{db}
}

func (u *UserDAO) GetUser(username string) (types.User, error) {
	user := types.User{}
	e := u.db.C().First(&user, "username = ?", username).Related(&user.Groups, "groups").Error
	if gorm.IsRecordNotFoundError(e) {
		return user, err.NewNotFoundMessageError(i18n.T("storage.users.user_not_exists", username))
	}
	return user, e
}

func (u *UserDAO) AddUser(user types.User) (types.User, error) {
	e := u.db.C().Where("username = ?", user.Username).Find(&types.User{}).Error
	if e == nil {
		return types.User{},
			err.NewNotAllowedMessageError(i18n.T("storage.users.user_exists", user.Username))
	}
	if !gorm.IsRecordNotFoundError(e) {
		return types.User{}, e
	}
	encoded, e := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if e != nil {
		return types.User{}, e
	}
	user.Password = string(encoded)
	e = u.db.C().Create(&user).Error
	return user, e
}

func (u *UserDAO) UpdateUser(username string, user types.User) error {
	data := types.M{}
	if user.Password != "" {
		encoded, e := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if e != nil {
			return e
		}
		data["password"] = string(encoded)
	}

	return u.db.C().Transaction(func(tx *gorm.DB) error {
		if len(data) > 0 {
			s := tx.Model(&types.User{}).Where("username = ?", username).Updates(data)
			if s.Error != nil {
				return s.Error
			}
			if s.RowsAffected != 1 {
				return err.NewNotFoundMessageError(i18n.T("storage.users.user_not_exists", username))
			}
		}
		if user.Groups != nil {
			if e := tx.Where("username = ?", username).Delete(&types.UserGroup{}).Error; e != nil {
				return e
			}
			for _, g := range user.Groups {
				if e := tx.Create(&types.UserGroup{Username: username, GroupName: g.Name}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
}

func (u *UserDAO) DeleteUser(username string) error {
	return u.db.C().Transaction(func(tx *gorm.DB) error {
		s := tx.Delete(types.User{}, "username = ?", username)
		if s.Error != nil {
			return s.Error
		}
		if s.RowsAffected != 1 {
			return err.NewNotFoundMessageError(i18n.T("storage.users.user_not_exists", username))

		}
		if e := tx.Where("username = ?", username).Delete(&types.UserGroup{}).Error; e != nil {
			return e
		}
		return tx.Where("subject = ?", types.UserSubject(username)).Delete(&types.PathPermission{}).Error
	})
}

func (u *UserDAO) ListUser() ([]types.User, error) {
	users := make([]types.User, 0)
	e := u.db.C().Find(&users).Error
	return users, e
}
