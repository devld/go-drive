package job

import (
	err "go-drive/common/errors"
	"go-drive/common/event"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

func init() {
	t := i18n.TPrefix("jobs.trigger.entry.")

	RegisterTriggerDef(JobTriggerTypeEntry, JobTriggerDef{
		Name:        string(JobTriggerTypeEntry),
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "pathPattern", Label: t("path_pattern"), Description: t("path_pattern_desc"), Type: "text", Required: true},
			{Field: "eventTypes", Label: t("event_types"), Type: "checkboxes", Required: true,
				Options: &[]types.FormItemOption{
					{Name: t("event_entry_updated"), Value: string(EntryEventTypeUpdated)},
					{Name: t("event_entry_deleted"), Value: string(EntryEventTypeDeleted)},
				}},
		},
		Validate: validateEntryTriggerConfig,
		Factory: func(executor *JobExecutor, ch *registry.ComponentsHolder) IJobTriggerInstance {
			return newEntryEventTrigger(executor, ch.Get("eventBus").(event.Bus))
		},
	})
}

// entryEventTrigger handles entry event-based job triggering (created/updated/deleted)
type entryEventTrigger struct {
	executor *JobExecutor
	triggers []entryEventTriggerConfig
	mu       sync.RWMutex

	bus event.Bus
}

var _ IJobTriggerInstance = (*entryEventTrigger)(nil)

type entryEventTriggerConfig struct {
	jobID       uint
	pathPattern string
	eventTypes  map[EntryEventType]struct{}
}

func newEntryEventTrigger(executor *JobExecutor, bus event.Bus) *entryEventTrigger {
	trigger := &entryEventTrigger{
		executor: executor,
		triggers: make([]entryEventTriggerConfig, 0, 2),
		bus:      bus,
	}
	trigger.subscribeBusEvents()
	return trigger
}

func validateEntryTriggerConfig(config types.SM) error {
	if config["pathPattern"] == "" {
		return err.NewBadRequestError("path pattern is required")
	}
	_, e := doublestar.Match(config["pathPattern"], "/test")
	if e != nil {
		return err.NewBadRequestError("invalid path pattern: " + e.Error())
	}
	eventTypes, e := parseEventTypes(config["eventTypes"])
	if e != nil || len(eventTypes) == 0 {
		return err.NewBadRequestError("eventTypes must be a non-empty JSON array of \"updated\" and/or \"deleted\"")
	}
	return nil
}

func parseEventTypes(s string) (map[EntryEventType]struct{}, error) {
	if s == "" {
		return nil, err.NewBadRequestError("eventTypes is required")
	}
	result := make(map[EntryEventType]struct{}, 2)
	for _, v := range strings.Split(s, ",") {
		result[EntryEventType(strings.TrimSpace(v))] = struct{}{}
	}
	if len(result) == 0 {
		return nil, err.NewBadRequestError("eventTypes cannot be empty")
	}
	return result, nil
}

func parseConfig(config types.SM) (*entryEventTriggerConfig, error) {
	pathPattern := config["pathPattern"]
	if pathPattern == "" {
		return nil, err.NewBadRequestError("path pattern is required")
	}
	_, e := doublestar.Match(pathPattern, "test")
	if e != nil {
		return nil, err.NewBadRequestError("invalid path pattern: " + e.Error())
	}
	eventTypes, e := parseEventTypes(config["eventTypes"])
	if e != nil {
		return nil, e
	}
	return &entryEventTriggerConfig{pathPattern: pathPattern, eventTypes: eventTypes}, nil
}

func (eet *entryEventTrigger) Register(jobID uint, config types.SM) error {
	eet.mu.Lock()
	defer eet.mu.Unlock()

	triggerConfig, e := parseConfig(config)
	if e != nil {
		return e
	}
	triggerConfig.jobID = jobID
	eet.triggers = append(eet.triggers, *triggerConfig)
	return nil
}

func (eet *entryEventTrigger) GetInfo(jobID uint) ([]types.SM, error) {
	return nil, nil
}

func (eet *entryEventTrigger) Clear() {
	eet.mu.Lock()
	defer eet.mu.Unlock()
	eet.triggers = make([]entryEventTriggerConfig, 0, 2)
}

// checkAndTrigger checks if the event matches any registered triggers and triggers the job.
// entryEventType is "updated" or "deleted".
func (eet *entryEventTrigger) checkAndTrigger(path string, entryEventType EntryEventType, includeDescendants bool) {
	eet.mu.RLock()
	triggers := make([]entryEventTriggerConfig, len(eet.triggers))
	copy(triggers, eet.triggers)
	eet.mu.RUnlock()

	for _, config := range triggers {
		_, matched := config.eventTypes[entryEventType]
		if !matched {
			continue
		}
		pathMatched, e := doublestar.Match(config.pathPattern, path)
		if e != nil {
			log.Printf("error matching path pattern %s with path %s: %v", config.pathPattern, path, e)
			continue
		}
		if !pathMatched {
			continue
		}
		event := TriggerEvent{
			Type: JobTriggerTypeEntry,
			Data: types.SM{
				"path":               path,
				"eventType":          string(entryEventType),
				"includeDescendants": strconv.FormatBool(includeDescendants),
			},
		}
		eet.executor.TriggerExecution(config.jobID, event)
	}
}

func (eet *entryEventTrigger) onEntryUpdated(dc types.DriveListenerContext, path string, includeDescendants bool) {
	eet.checkAndTrigger(path, "updated", includeDescendants)
}

func (eet *entryEventTrigger) onEntryDeleted(dc types.DriveListenerContext, path string) {
	eet.checkAndTrigger(path, "deleted", false)
}

func (eet *entryEventTrigger) subscribeBusEvents() {
	eet.bus.Subscribe(event.EntryUpdated, eet.onEntryUpdated)
	eet.bus.Subscribe(event.EntryDeleted, eet.onEntryDeleted)
}

func (eet *entryEventTrigger) Dispose() error {
	eet.bus.Unsubscribe(event.EntryUpdated, eet.onEntryUpdated)
	eet.bus.Unsubscribe(event.EntryDeleted, eet.onEntryDeleted)
	return nil
}
