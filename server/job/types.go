package job

import (
	"context"
	"go-drive/common/registry"
	"go-drive/common/types"
)

// JobTriggerDef represents a job trigger definition (similar to JobActionDef).
// Register/Unregister are handled in JobExecutor by trigger type.
type JobTriggerDef struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"displayName" i18n:""`
	Description string           `json:"description" i18n:""`
	ParamsForm  []types.FormItem `json:"paramsForm"`

	Validate func(config types.SM) error                                                    `json:"-"`
	Factory  func(executor *JobExecutor, ch *registry.ComponentsHolder) IJobTriggerInstance `json:"-"`
}

// JobActionDef represents a job action definition
type JobActionDef struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"displayName" i18n:""`
	Description string           `json:"description" i18n:""`
	ParamsForm  []types.FormItem `json:"paramsForm"`

	Do func(context.Context, types.SM, *registry.ComponentsHolder, func(string)) error `json:"-"`
}

type IJobTriggerInstance interface {
	types.IDisposable
	Register(jobID uint, config types.SM) error
	GetInfo(jobID uint) ([]types.SM, error)
	Clear()
}

// JobTriggerType represents the type of a job trigger
type JobTriggerType string

const (
	JobTriggerTypeCron  JobTriggerType = "cron"
	JobTriggerTypeEntry JobTriggerType = "entry"
)

// EntryEventType is the type of entry event (used in entry trigger's eventTypes)
type EntryEventType string

const (
	EntryEventTypeUpdated EntryEventType = "updated"
	EntryEventTypeDeleted EntryEventType = "deleted"
)

// ParsedJobTrigger represents a trigger configuration for a job
type ParsedJobTrigger struct {
	Type   JobTriggerType `json:"type"`   // cron, entry
	Config types.SM       `json:"config"` // Trigger-specific configuration
}

// TriggerEvent represents an event that triggered a job
type TriggerEvent struct {
	Type JobTriggerType `json:"type,omitempty"` // cron or entry
	Data types.SM       `json:"data,omitempty"` // Trigger-specific data (e.g. path, eventType for entry)
}
