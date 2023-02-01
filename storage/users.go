package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"

	cmap "github.com/orcaman/concurrent-map/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserDAO struct {
	db    *DB
	cache cmap.ConcurrentMap[string, types.User]
}

func NewUserDAO(db *DB, ch *registry.ComponentsHolder) *UserDAO {
	dao := &UserDAO{
		db:    db,
		cache: cmap.New[types.User](),
	}
	ch.Add("userDAO", dao)
	return dao
}

func (u *UserDAO) GetUser(username string) (types.User, error) {
	if cached, ok := u.cache.Get(username); ok {
		return cached, nil
	}

	user := types.User{}
	e := u.db.C().First(&user, "username = ?", username).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return user, err.NewNotFoundMessageError(i18n.T("storage.users.user_not_exists", username))
	}
	if e = u.db.C().Model(&user).Association("Groups").Find(&user.Groups); e != nil {
		return user, e
	}
	u.cache.Set(username, user)
	return user, e
}

func (u *UserDAO) AddUser(user types.User) (types.User, error) {
	e := u.db.C().Where("username = ?", user.Username).Take(&types.User{}).Error
	if e == nil {
		return types.User{},
			err.NewNotAllowedMessageError(i18n.T("storage.users.user_exists", user.Username))
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
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
	data := map[string]interface{}{
		"root_path": user.RootPath,
	}
	if user.Password != "" {
		encoded, e := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if e != nil {
			return e
		}
		data["password"] = string(encoded)
	}
	defer u.cache.Remove(username)
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
	u.cache.Remove(username)
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
