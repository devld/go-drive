package server

import (
	"go-drive/common"
)

type EntryJson struct {
	Path      string                 `json:"path"`
	Name      string                 `json:"name"`
	Type      common.EntryType       `json:"type"`
	Size      int64                  `json:"size"`
	Meta      map[string]interface{} `json:"meta"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
}

func NewEntryJson(e common.IEntry) *EntryJson {
	entryMeta := e.Meta()
	meta := make(map[string]interface{})
	meta["can_write"] = entryMeta.CanWrite()
	if entryMeta != nil {
		for k, v := range entryMeta.Props() {
			meta[k] = v
		}
	}
	return &EntryJson{
		Path:      e.Path(),
		Name:      e.Name(),
		Type:      e.Type(),
		Size:      e.Size(),
		Meta:      meta,
		CreatedAt: e.CreatedAt(),
		UpdatedAt: e.UpdatedAt(),
	}
}

type UploadConfig struct {
	Provider string      `json:"provider"`
	Config   interface{} `json:"config"`
}

func NewUploadConfig(c *common.DriveUploadConfig) *UploadConfig {
	return &UploadConfig{c.Provider, c.Config}
}
