package job

import (
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
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

// cronJobInfo contains information about a scheduled cron job
type cronJobInfo struct {
	job      types.Job
	schedule cron.Schedule
	stop     func()
	nextRun  time.Time
}

var _ IJobTriggerInstance = (*cronTrigger)(nil)

// cronTrigger handles cron-based job scheduling (used by Trigger definition)
type cronTrigger struct {
	executor *JobExecutor
	s        *gocron.Scheduler
}

func newCronTrigger(executor *JobExecutor) *cronTrigger {
	s := gocron.NewScheduler(time.Local)

	s.TagsUnique()
	s.WaitForScheduleAll()

	s.StartAsync()

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

	j, e := ct.s.Cron(schedule).Tag(strconv.FormatUint(uint64(jobID), 10)).Do(ct.triggerAction, jobID)

	if e == nil {
		j.SingletonMode()
	}

	return e
}

func (ct *cronTrigger) GetInfo(jobID uint) ([]types.SM, error) {
	jobs, e := ct.s.FindJobsByTag(strconv.FormatUint(uint64(jobID), 10))
	if e != nil {
		return nil, nil
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	stats := make([]types.SM, 0, len(jobs))
	for _, job := range jobs {
		stats = append(stats, types.SM{
			"nextRun": job.NextRun().Format(time.RFC3339),
		})
	}
	return stats, nil
}

func (ct *cronTrigger) triggerAction(jobID uint) {
	ct.executor.TriggerExecution(jobID, TriggerEvent{Type: JobTriggerTypeCron})
}

func (ct *cronTrigger) Clear() {
	ct.s.Clear()
}

func (ct *cronTrigger) Dispose() error {
	ct.s.Stop()
	ct.Clear()
	return nil
}
