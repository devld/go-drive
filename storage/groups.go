package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"go-drive/common"
	"go-drive/common/types"
)

type GroupStorage struct {
	db *DB
}

type GroupWithUsers struct {
	types.Group
	Users []types.User `json:"users"`
}

func NewGroupStorage(db *DB) (*GroupStorage, error) {
	return &GroupStorage{db}, nil
}

func (g *GroupStorage) ListGroup() ([]types.Group, error) {
	groups := make([]types.Group, 0)
	e := g.db.C().Find(&groups).Error
	return groups, e
}

func (g *GroupStorage) GetGroup(name string) (GroupWithUsers, error) {
	gus := GroupWithUsers{}
	group := types.Group{}
	e := g.db.C().First(&group, "name = ?", name).Error
	if gorm.IsRecordNotFoundError(e) {
		return gus, common.NewNotFoundMessageError(fmt.Sprintf("group '%s' not found", name))
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
	if users == nil || len(users) == 0 {
		return nil
	}
	for _, u := range users {
		if e := db.Create(&types.UserGroup{Username: u.Username, GroupName: name}).Error; e != nil {
			return e
		}
	}
	return nil
}

func (g *GroupStorage) AddGroup(group GroupWithUsers) (GroupWithUsers, error) {
	e := g.db.C().Where("name = ?", group.Name).Find(&types.Group{}).Error
	if e == nil {
		return GroupWithUsers{},
			common.NewNotAllowedMessageError(fmt.Sprintf("group '%s' exists", group.Name))
	}
	if !gorm.IsRecordNotFoundError(e) {
		return GroupWithUsers{}, e
	}
	e = g.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(group).Error; e != nil {
			return e
		}
		return saveUserGroup(group.Users, group.Name, tx)
	})
	return group, e
}

func (g *GroupStorage) UpdateGroup(name string, gus GroupWithUsers) error {
	if gus.Users == nil {
		return nil
	}
	return g.db.C().Transaction(func(tx *gorm.DB) error {
		group := types.Group{}
		e := tx.First(&group, "name = ?", name).Error
		if gorm.IsRecordNotFoundError(e) {
			return common.NewNotFoundMessageError(fmt.Sprintf("group '%s' not found", name))
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

func (g *GroupStorage) DeleteGroup(name string) error {
	return g.db.C().Transaction(func(tx *gorm.DB) error {
		s := tx.Delete(types.Group{}, "name = ?", name)
		if s.Error != nil {
			return s.Error
		}
		if s.RowsAffected != 1 {
			return common.NewNotFoundMessageError(fmt.Sprintf("group '%s' not found", name))
		}
		if e := tx.Where("group_name = ?", name).Delete(&types.UserGroup{}).Error; e != nil {
			return e
		}
		return tx.Where("subject = ?", types.GroupSubject(name)).Delete(&types.PathPermission{}).Error
	})
}
