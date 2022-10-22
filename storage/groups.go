package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type GroupDAO struct {
	db *DB
}

type GroupWithUsers struct {
	types.Group
	Users []types.User `json:"users"`
}

func NewGroupDAO(db *DB, ch *registry.ComponentsHolder) *GroupDAO {
	dao := &GroupDAO{db}
	ch.Add("groupDAO", dao)
	return dao
}

func (g *GroupDAO) ListGroup() ([]types.Group, error) {
	groups := make([]types.Group, 0)
	e := g.db.C().Find(&groups).Error
	return groups, e
}

func (g *GroupDAO) GetGroup(name string) (GroupWithUsers, error) {
	gus := GroupWithUsers{}
	group := types.Group{}
	e := g.db.C().First(&group, "name = ?", name).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return gus, err.NewNotFoundMessageError(i18n.T("storage.groups.group_not_exists", name))
	}
	if e != nil {
		return gus, e
	}
	gus.Group = group

	users := make([]types.User, 0)
	ugs := make([]types.UserGroup, 0)
	e = g.db.C().Find(&ugs, "group_name = ?", name).Error
	if e != nil {
		return gus, e
	}
	usernames := make([]string, len(ugs))
	for i, ug := range ugs {
		usernames[i] = ug.Username
	}
	e = g.db.C().Find(&users, "username IN (?)", usernames).Error
	gus.Users = users

	return gus, e
}

func saveUserGroup(users []types.User, name string, db *gorm.DB) error {
	if len(users) == 0 {
		return nil
	}
	for _, u := range users {
		if e := db.Create(&types.UserGroup{Username: u.Username, GroupName: name}).Error; e != nil {
			return e
		}
	}
	return nil
}

func (g *GroupDAO) AddGroup(group GroupWithUsers) (GroupWithUsers, error) {
	e := g.db.C().Where("name = ?", group.Name).Take(&types.Group{}).Error
	if e == nil {
		return GroupWithUsers{},
			err.NewNotAllowedMessageError(i18n.T("storage.groups.group_exists", group.Name))
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return GroupWithUsers{}, e
	}
	e = g.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&group.Group).Error; e != nil {
			return e
		}
		return saveUserGroup(group.Users, group.Name, tx)
	})
	return group, e
}

func (g *GroupDAO) UpdateGroup(name string, gus GroupWithUsers) error {
	if gus.Users == nil {
		return nil
	}
	return g.db.C().Transaction(func(tx *gorm.DB) error {
		group := types.Group{}
		e := tx.First(&group, "name = ?", name).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return err.NewNotFoundMessageError(i18n.T("storage.groups.group_not_exists", name))
		}
		if e != nil {
			return e
		}
		if e := tx.Delete(&types.UserGroup{}, "group_name = ?", name).Error; e != nil {
			return e
		}
		return saveUserGroup(gus.Users, group.Name, tx)
	})
}

func (g *GroupDAO) DeleteGroup(name string) error {
	return g.db.C().Transaction(func(tx *gorm.DB) error {
		s := tx.Delete(types.Group{}, "name = ?", name)
		if s.Error != nil {
			return s.Error
		}
		if s.RowsAffected != 1 {
			return err.NewNotFoundMessageError(i18n.T("storage.groups.group_not_exists", name))
		}
		if e := tx.Where("group_name = ?", name).Delete(&types.UserGroup{}).Error; e != nil {
			return e
		}
		return tx.Where("subject = ?", types.GroupSubject(name)).Delete(&types.PathPermission{}).Error
	})
}
