package types

import "strings"

type User struct {
	Username string  `gorm:"column:username;primaryKey;not null;type:string;size:32" json:"username" binding:"required"`
	Password string  `gorm:"column:password;not null;type:string;size:64" json:"password"`
	Groups   []Group `gorm:"many2many:user_groups;joinForeignKey:group_name;foreignKey:username" json:"groups"`
}

type Group struct {
	Name string `gorm:"column:name;primaryKey;not null;type:string;size:32" json:"name" binding:"required"`
}

type UserGroup struct {
	Username  string `gorm:"column:username;primaryKey;not null;type:string;size:32" binding:"required"`
	GroupName string `gorm:"column:group_name;primaryKey;not null;type:string;size:32" binding:"required"`
}

type Drive struct {
	Name    string `gorm:"column:name;primaryKey;not null;type:string;size:255" json:"name" binding:"required"`
	Enabled bool   `gorm:"column:enabled;not null;type:bool" json:"enabled"`
	Type    string `gorm:"column:type;not null;type:string;size:32" json:"type" binding:"required"`
	Config  string `gorm:"column:config;not null;type:string;size:4096" json:"config"`
}

type PathMount struct {
	ID      uint    `gorm:"column:id;primaryKey;autoIncrement"`
	Path    *string `gorm:"column:path;not null;type:string;size:4096" json:"path"`
	Name    string  `gorm:"column:name;not null;type:string;size:255" json:"name"`
	MountAt string  `gorm:"column:mount_at;not null;type:string;size:4096" json:"mount_at"`
}

func (PathMount) TableName() string {
	return "path_mount"
}

type DriveData struct {
	Drive string `gorm:"column:drive;primaryKey;not null;type:string;size:255"`
	Key   string `gorm:"column:data_key;primaryKey;not null;type:string;size:255"`
	Value string `gorm:"column:data_value;not null;type:string;size:4096"`
}

func (DriveData) TableName() string {
	return "drive_data"
}

const (
	CacheEntry    uint8 = 1
	CacheChildren uint8 = 2
)

type DriveCache struct {
	Drive     string `gorm:"column:drive;primaryKey;not null;type:string;size:255"`
	Path      string `gorm:"column:path;primaryKey;not null;type:string;size:255"`
	Depth     *uint8 `gorm:"column:depth;primaryKey;not null"`
	Type      uint8  `gorm:"column:type;primaryKey;not null"`
	Value     string `gorm:"column:cache_value;not null;type:text"`
	ExpiresAt int64  `gorm:"column:expires_at;not null"`
}

func (DriveCache) TableName() string {
	return "drive_cache"
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
	ID      uint    `gorm:"column:id;primaryKey;autoIncrement"`
	Path    *string `gorm:"column:path;not null;type:string;size:4096" json:"path"`
	Subject string  `gorm:"column:subject;not null;type:string;size:34" json:"subject"`
	// Permission bits for the path which subject accessed: 1: read, 2: write
	Permission Permission `gorm:"column:permission;not null" json:"permission"`
	// Policy to apply to the permission when subject access this path: 0: REJECT, 1: ACCEPT
	Policy uint8 `gorm:"column:policy;not null" json:"policy"`
	Depth  uint8 `gorm:"column:depth;not null" json:"-"`
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
