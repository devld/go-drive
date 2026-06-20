package job

import (
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"strconv"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/robfig/cron/v3"
)

func init() {
	t := i18n.TPrefix("jobs.trigger.cron.")
	RegisterTriggerDef(JobTriggerTypeCron, JobTriggerDef{
		Name:        string(JobTriggerTypeCron),
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "schedule", Label: t("schedule"), Description: t("schedule_desc"), Type: "text", Required: true},
		},
		Validate: validateCronConfig,
		Factory: func(executor *JobExecutor, ch *registry.ComponentsHolder) IJobTriggerInstance {
			return newCronTrigger(executor)
		},
	})
}

var _ IJobTriggerInstance = (*cronTrigger)(nil)

// cronTrigger handles cron-based job scheduling (used by Trigger definition)
type cronTrigger struct {
	executor *JobExecutor
	s        gocron.Scheduler
}

func newCronTrigger(executor *JobExecutor) *cronTrigger {
	s, e := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if e != nil {
		panic(e)
	}
	s.Start()

	return &cronTrigger{executor: executor, s: s}
}

func validateCronConfig(config types.SM) error {
	schedule := config["schedule"]
	if schedule == "" {
		return err.NewBadRequestError("cron schedule is required")
	}
	_, e := cron.ParseStandard(schedule)
	if e != nil {
		return err.NewNotAllowedMessageError("invalid cron schedule: " + e.Error())
	}
	return nil
}

func (ct *cronTrigger) Validate(config types.SM) error {
	schedule := config["schedule"]
	if schedule == "" {
		return err.NewBadRequestError("cron schedule is required")
	}
	_, e := cron.ParseStandard(schedule)
	if e != nil {
		return err.NewBadRequestError("invalid cron schedule: " + e.Error())
	}
	return nil
}

func (ct *cronTrigger) Register(jobID uint, config types.SM) error {
	schedule := config["schedule"]
	if schedule == "" {
		return err.NewBadRequestError("cron schedule is required")
	}

	tag := strconv.FormatUint(uint64(jobID), 10)
	for _, scheduledJob := range ct.s.Jobs() {
		for _, jobTag := range scheduledJob.Tags() {
			if jobTag == tag {
				return fmt.Errorf("cron job %d is already registered", jobID)
			}
		}
	}

	_, e := ct.s.NewJob(
		gocron.CronJob(schedule, false),
		gocron.NewTask(ct.triggerAction, jobID),
		gocron.WithTags(tag),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	return e
}

func (ct *cronTrigger) GetInfo(jobID uint) ([]types.SM, error) {
	tag := strconv.FormatUint(uint64(jobID), 10)
	stats := make([]types.SM, 0, 1)
	for _, scheduledJob := range ct.s.Jobs() {
		matched := false
		for _, jobTag := range scheduledJob.Tags() {
			if jobTag == tag {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		nextRun, e := scheduledJob.NextRun()
		if e != nil {
			return nil, e
		}
		stats = append(stats, types.SM{
			"nextRun": nextRun.Format(time.RFC3339),
		})
	}
	return stats, nil
}

func (ct *cronTrigger) triggerAction(jobID uint) {
	ct.executor.TriggerExecution(jobID, TriggerEvent{Type: JobTriggerTypeCron})
}

func (ct *cronTrigger) Clear() {
	for _, scheduledJob := range ct.s.Jobs() {
		_ = ct.s.RemoveJob(scheduledJob.ID())
	}
}

func (ct *cronTrigger) Dispose() error {
	return ct.s.Shutdown()
}
