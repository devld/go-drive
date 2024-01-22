package types

import "strings"

type Option struct {
	ID    uint   `gorm:"column:id;primaryKey;autoIncrement"`
	Key   string `gorm:"column:key;type:string;size:64;not null;unique;uniqueIndex"`
	Value string `gorm:"column:value;type:string;size:4096"`
}

type User struct {
	Username string  `gorm:"column:username;primaryKey;not null;type:string;size:32" json:"username" binding:"required"`
	Password string  `gorm:"column:password;not null;type:string;size:64" json:"password,omitempty"`
	RootPath string  `gorm:"column:root_path;type:string;size:4096" json:"rootPath,omitempty"`
	Groups   []Group `gorm:"many2many:user_groups;joinForeignKey:username;foreignKey:username" json:"groups"`
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
	MountAt string  `gorm:"column:mount_at;not null;type:string;size:4096" json:"mountAt"`
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

func (p Permission) Readable() bool {
	return p&PermissionRead == PermissionRead
}

func (p Permission) Writable() bool {
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
	ID      uint    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Path    *string `gorm:"column:path;not null;type:string;size:4096" json:"path"`
	Subject string  `gorm:"column:subject;not null;type:string;size:34" json:"subject"`
	// Permission bits for the path which subject accessed: 1: read, 2: write
	Permission Permission `gorm:"column:permission;not null" json:"permission"`
	// Policy to apply to the permission when subject access this path: 0: REJECT, 1: ACCEPT
	Policy uint8 `gorm:"column:policy;not null" json:"policy"`
}

type PathMeta struct {
	Path          *string `gorm:"column:path;primaryKey;not null;type:string;size:4096" json:"path"`
	Password      string  `gorm:"column:password;type:string;size:64" json:"password"`
	DefaultSort   string  `gorm:"column:default_sort;type:string;size:32" json:"defaultSort"`
	DefaultMode   string  `gorm:"column:default_mode;type:string;size:32" json:"defaultMode"`
	HiddenPattern string  `gorm:"column:hidden_pattern;type:string;size:4096" json:"hiddenPattern"`

	// Recursive Password|DefaultSort|DefaultMode|HiddenPattern
	Recursive uint32 `gorm:"column:recursive;not null" json:"recursive"`
}

type FileBucket struct {
	Name        string `gorm:"column:name;primaryKey;not null;type:string;size:255" json:"name" binding:"required"`
	TargetPath  string `gorm:"column:target_path;not null;type:string;size:4096" json:"targetPath" binding:"required"`
	KeyTemplate string `gorm:"column:key_template;type:string;size:4096" json:"keyTemplate"`
	CustomKey   bool   `gorm:"column:custom_key;not null;type:bool" json:"customKey"`
	// SecretToken is the auto-generated upload token for this bucket
	SecretToken string `gorm:"column:secret_token;type:string;size:32" json:"secretToken" binding:"required"`
	URLTemplate string `gorm:"column:url_template;not null;type:string;size:4096" json:"urlTemplate"`
	// AllowedTypes is a comma separated list of allowed mime types or file extensions, e.g. "image/png,image/jpeg,.png,.jpg"
	AllowedTypes string `gorm:"column:allowed_types;type:string;size:4096" json:"allowedTypes"`
	// MaxSize is the maximum allowed size with unit, 0 for unlimited
	MaxSize string `gorm:"column:max_size;not null;type:string" json:"maxSize"`
}

type Job struct {
	ID          uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Description string `gorm:"column:description;not null;type:text" json:"description"`
	Job         string `gorm:"column:job;not null;type:string;size:64" json:"job"`
	Params      string `gorm:"column:params;not null;type:text" json:"params"`
	Schedule    string `gorm:"column:schedule;not null;type:string;size:64" json:"schedule"`
	Enabled     bool   `gorm:"column:enabled;not null;type:bool" json:"enabled"`
}

const (
	JobExecutionRunning = "running"
	JobExecutionSuccess = "success"
	JobExecutionFailed  = "failed"
)

type JobExecution struct {
	ID          uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	JobId       uint   `gorm:"column:job_id;not null;type:uint" json:"jobId"`
	StartedAt   uint64 `gorm:"column:started_at;type:uint" json:"startedAt"`
	CompletedAt uint64 `gorm:"column:completed_at;type:uint" json:"completedAt"`
	Status      string `gorm:"column:status;not null;type:string" json:"status"`
	Logs        string `gorm:"column:logs;type:string" json:"logs"`
	ErrorMsg    string `gorm:"column:error_msg;type:text" json:"errorMsg"`
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
