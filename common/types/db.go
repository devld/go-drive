package types

import "strings"

type User struct {
	Username string  `gorm:"COLUMN:username;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32" json:"username" binding:"required"`
	Password string  `gorm:"COLUMN:password;NOT NULL;TYPE:VARCHAR;SIZE:64" json:"password"`
	Groups   []Group `gorm:"MANY2MANY:user_groups;ASSOCIATION_JOINTABLE_FOREIGNKEY:group_name;JOINTABLE_FOREIGNKEY:username" json:"groups"`
}

type Group struct {
	Name string `gorm:"COLUMN:name;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32" json:"name" binding:"required"`
}

type UserGroup struct {
	Username  string `gorm:"COLUMN:username;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32" binding:"required"`
	GroupName string `gorm:"COLUMN:group_name;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32" binding:"required"`
}

type Drive struct {
	Name   string `gorm:"COLUMN:name;PRIMARY_KEY;TYPE:VARCHAR;SIZE:255" json:"name" binding:"required"`
	Type   string `gorm:"COLUMN:type;NOT NULL;TYPE:VARCHAR;SIZE:32" json:"type" binding:"required"`
	Config string `gorm:"COLUMN:config;NOT NULL;TYPE:VARCHAR;SIZE:4096" json:"config"`
}

type PathMount struct {
	Path    *string `gorm:"COLUMN:path;PRIMARY_KEY;TYPE:VARCHAR;SIZE:4096" json:"path"`
	Name    string  `gorm:"COLUMN:name;PRIMARY_KEY;NOT NULL;TYPE:VARCHAR;SIZE:255" json:"name"`
	MountAt string  `gorm:"COLUMN:mount_at;NOT NULL;TYPE:VARCHAR;SIZE:4096" json:"mount_at"`
}

func (PathMount) TableName() string {
	return "path_mount"
}

type Permission uint8

func (p Permission) CanRead() bool {
	return p&PermissionRead == PermissionRead
}

func (p Permission) CanWrite() bool {
	return p&PermissionWrite == PermissionWrite
}

const (
	PermissionEmpty     Permission = 0
	PermissionRead      Permission = 1 << 0
	PermissionWrite     Permission = 1 << 1
	PermissionReadWrite            = PermissionRead | PermissionWrite
)

const (
	PolicyReject uint8 = 0
	PolicyAccept uint8 = 1
)

type PathPermission struct {
	Path    *string `gorm:"COLUMN:path;PRIMARY_KEY;TYPE:VARCHAR;SIZE:4096" json:"path"`
	Subject string  `gorm:"COLUMN:subject;PRIMARY_KEY;NOT NULL;TYPE:VARCHAR;SIZE:34" json:"subject"`
	// Permission bits for the path which subject accessed: 1: read, 2: write
	Permission Permission `gorm:"COLUMN:permission;NOT NULL;TYPE:INTEGER" json:"permission"`
	// Policy to apply to the permission when subject access this path: 0: REJECT, 1: ACCEPT
	Policy uint8 `gorm:"COLUMN:policy;NOT NULL;TYPE:INTEGER" json:"policy"`
	Depth  uint8 `gorm:"COLUMN:depth;NOT NULL;TYPE:INTEGER" json:"-"`
}

func UserSubject(username string) string {
	return "u:" + username
}

func GroupSubject(name string) string {
	return "g:" + name
}

const AnySubject = "ANY"

func (p PathPermission) IsForAnonymous() bool {
	return p.Subject == AnySubject
}

func (p PathPermission) IsForUser() bool {
	return strings.HasPrefix(p.Subject, "u:")
}

func (p PathPermission) IsForGroup() bool {
	return strings.HasPrefix(p.Subject, "g:")
}

func (p PathPermission) IsAccept() bool {
	return p.Policy == PolicyAccept
}

func (p PathPermission) IsReject() bool {
	return p.Policy == PolicyReject
}
