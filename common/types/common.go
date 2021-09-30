package types

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"
)

type M map[string]interface{}
type SM map[string]string

type TaskCtx interface {
	context.Context
	Progress(loaded int64, abs bool)
	Total(total int64, abs bool)
}

type IDisposable interface {
	Dispose() error
}

type IStatistics interface {
	// Status returns the name, status of this component
	Status() (string, SM, error)
}

// ISysConfig provides some configuration for the client
type ISysConfig interface {
	// SysConfig returns the config name, config map
	SysConfig() (string, M, error)
}

type FormItemOption struct {
	Name     string `json:"name" i18n:""`
	Title    string `json:"title" i18n:""`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type FormItem struct {
	Label        string           `json:"label" i18n:""`
	Type         string           `json:"type"`
	Field        string           `json:"field"`
	Required     bool             `json:"required"`
	Description  string           `json:"description" i18n:""`
	Disabled     bool             `json:"disabled"`
	Options      []FormItemOption `json:"options"`
	DefaultValue string           `json:"default_value"`
	// Secret is the replacement text when sending the value to client.
	// The raw value will be sent if Secret is empty.
	// FormItem with type 'password' will always be replaced.
	// This value cannot be used with i18n.
	Secret string `json:"-"`
}

func (c SM) GetInt(key string, defVal int) int {
	v, e := strconv.Atoi(c[key])
	if e != nil {
		return defVal
	}
	return v
}

func (c SM) GetUint(key string, defVal uint) uint {
	v := c.GetInt(key, -1)
	if v < 0 || uint(v) > uint(math.MaxUint) {
		return defVal
	}
	return uint(v)
}

func (c SM) GetInt64(key string, defVal int64) int64 {
	v, e := strconv.ParseInt(c[key], 10, 64)
	if e != nil {
		return defVal
	}
	return v
}

func (c SM) GetUint64(key string, defVal uint64) uint64 {
	v := c.GetInt64(key, -1)
	if v < 0 || uint64(v) > uint64(math.MaxUint64) {
		return defVal
	}
	return uint64(v)
}

func (c SM) GetDuration(key string, defVal time.Duration) time.Duration {
	dur, e := time.ParseDuration(c[key])
	if e != nil {
		dur = defVal
	}
	return dur
}

func (c SM) GetUnixTime(key string, defVal *time.Time) time.Time {
	if defVal == nil {
		defVal = &time.Time{}
	}
	t := c.GetInt64(key, -1)
	if t == -1 {
		return *defVal
	}
	return time.Unix(t, 0)
}

func (c SM) GetBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(c[key]))
	return c[key] != "" && v != "false"
}
