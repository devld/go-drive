package db

type User struct {
	Username string  `gorm:"COLUMN:username;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32"`
	Password string  `gorm:"COLUMN:password;NOT NULL;TYPE:VARCHAR;SIZE:64"`
	Groups   []Group `gorm:"MANY2MANY:user_groups;ASSOCIATION_JOINTABLE_FOREIGNKEY:username;JOINTABLE_FOREIGNKEY:group_name"`
}

type Group struct {
	Name string `gorm:"COLUMN:name;PRIMARY_KEY;TYPE:VARCHAR;SIZE:32"`
}

type Drive struct {
	Name   string `gorm:"COLUMN:name;PRIMARY_KEY;TYPE:VARCHAR;SIZE:255"`
	Type   string `gorm:"COLUMN:type;NOT NULL;TYPE:VARCHAR;SIZE:32"`
	Config string `gorm:"COLUMN:config;NOT NULL;TYPE:VARCHAR;SIZE:4096"`
}
