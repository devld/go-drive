package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/storage"
	"log"
	"strings"
	"sync"
	"time"
)

type JobExecutor struct {
	ch     *registry.ComponentsHolder
	runner task.Runner
	jobDAO *storage.JobDAO

	triggers   map[JobTriggerType]IJobTriggerInstance
	executions map[uint]*jobExecutionItem

	mu sync.Mutex
}

func NewJobExecutor(jobDAO *storage.JobDAO, ch *registry.ComponentsHolder) (*JobExecutor, error) {
	runner := ch.Get("taskRunner").(task.Runner)

	executor := &JobExecutor{
		ch:         ch,
		runner:     runner,
		jobDAO:     jobDAO,
		executions: make(map[uint]*jobExecutionItem),
		triggers:   make(map[JobTriggerType]IJobTriggerInstance),
	}

	for _, triggerDef := range GetTriggerDefs() {
		executor.triggers[JobTriggerType(triggerDef.Name)] = triggerDef.Factory(executor, ch)
	}

	e := executor.ReloadJobs()
	if e != nil {
		return nil, e
	}

	_ = jobDAO.UpdateAllRunningJobExecutionsToFailed()

	ch.Add("jobExecutor", executor)
	return executor, nil
}

func (je *JobExecutor) ReloadJobs() error {
	je.mu.Lock()
	defer je.mu.Unlock()

	jobs, e := je.jobDAO.GetJobs(false)
	if e != nil {
		return e
	}

	for _, trigger := range je.triggers {
		trigger.Clear()
	}

	// Parse triggers and register them
	for _, job := range jobs {
		triggers, e := je.parseTriggers(job)
		if e != nil {
			log.Printf("error parsing triggers for job %d: %v", job.ID, e)
			continue
		}

		for _, trigger := range triggers {
			triggerInstance := je.triggers[trigger.Type]
			if triggerInstance == nil {
				continue
			}
			if e := triggerInstance.Register(job.ID, trigger.Config); e != nil {
				log.Printf("error registering trigger %s for job %d: %v", string(trigger.Type), job.ID, e)
				continue
			}
		}
	}

	return nil
}

func (je *JobExecutor) parseTriggers(job types.Job) ([]ParsedJobTrigger, error) {
	if job.Triggers == "" {
		return nil, fmt.Errorf("no triggers found for job %d", job.ID)
	}
	var triggers []ParsedJobTrigger
	if e := json.Unmarshal([]byte(job.Triggers), &triggers); e != nil {
		return nil, fmt.Errorf("failed to parse triggers for job %d: %w", job.ID, e)
	}
	return triggers, nil
}

// TriggerExecutionWithEvent runs the job using task.Runner with event information and returns the task
func (je *JobExecutor) TriggerExecution(jobID uint, event TriggerEvent) (task.Task, error) {
	job, e := je.jobDAO.GetJob(jobID)
	if e != nil {
		return task.Task{}, e
	}

	return je.runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, je.ExecuteJobSync(ctx, job, event, nil)
	}, task.WithNameGroup(job.Description, "job/execution"))
}

func (je *JobExecutor) ExecuteJobSync(ctx context.Context, job types.Job, event TriggerEvent, onLog func(string)) error {
	jobExecution, e := je.newJobExecution(job)
	if e != nil {
		return e
	}
	logger := newJobExecutionLogger(jobExecution.ID, onLog)
	return je.executeJob(ctx, job, jobExecution, logger, &event)
}

func (je *JobExecutor) executeJob(ctx context.Context, job types.Job,
	jobExecution *types.JobExecution, logger *jobExecutionLogger, event *TriggerEvent) (e error) {
	executionCtx, cancel := context.WithCancel(ctx)
	item := &jobExecutionItem{JobExecution: jobExecution, cancel: cancel, logger: logger}
	je.addJobExecution(item)

	defer func() {
		je.updateJobExecutionResult(item, e)
	}()

	actionDef := GetActionDef(job.Action)
	if actionDef == nil {
		e = errors.New("job not found")
		return
	}

	params := make(types.SM, 0)
	e = json.Unmarshal([]byte(job.ActionParams), &params)
	if e != nil {
		e = fmt.Errorf("failed to parse params: %s", e.Error())
		return
	}

	// Merge event information into params if provided
	if event != nil {
		eventBytes, e := json.Marshal(event)
		if e == nil {
			params["$event"] = string(eventBytes)
		}
	}

	e = actionDef.Do(executionCtx, params, je.ch, item.logger.Log)
	return
}

func (je *JobExecutor) newJobExecution(job types.Job) (*types.JobExecution, error) {
	jobExecution := &types.JobExecution{
		JobId:     job.ID,
		StartedAt: uint64(time.Now().UnixMilli()),
		Status:    types.JobExecutionRunning,
	}
	e := je.jobDAO.AddJobExecution(jobExecution)
	return jobExecution, e
}

func (je *JobExecutor) updateJobExecutionResult(item *jobExecutionItem, e error) {
	item.CompletedAt = uint64(time.Now().UnixMilli())
	if e != nil {
		item.Status = types.JobExecutionFailed
		item.ErrorMsg = e.Error()
	} else {
		item.Status = types.JobExecutionSuccess
	}
	item.JobExecution.Logs = item.logger.String()
	if e := je.jobDAO.UpdateJobExecution(item.JobExecution); e != nil {
		log.Printf("failed to update job execution: %v", e)
	}
	item.cancel()
	je.removeJobExecution(item.ID)
}

// ValidateTriggers validates all triggers in a job
func (je *JobExecutor) ValidateTriggers(triggersJSON string) error {
	if triggersJSON == "" {
		return err.NewBadRequestError("triggers are required")
	}

	var triggers []ParsedJobTrigger
	if e := json.Unmarshal([]byte(triggersJSON), &triggers); e != nil {
		return err.NewBadRequestError("invalid triggers format: " + e.Error())
	}

	if len(triggers) == 0 {
		return err.NewBadRequestError("at least one trigger is required")
	}

	for _, trigger := range triggers {
		triggerDef := GetTriggerDef(trigger.Type)
		if triggerDef == nil {
			return err.NewBadRequestError("unknown trigger type: " + string(trigger.Type))
		}
		if e := triggerDef.Validate(trigger.Config); e != nil {
			return e
		}
	}

	return nil
}

func (je *JobExecutor) GetJobTriggersInfo(jobID uint) (map[JobTriggerType][]types.SM, error) {
	statsMap := make(map[JobTriggerType][]types.SM, len(je.triggers))
	for triggerType, trigger := range je.triggers {
		stats, e := trigger.GetInfo(jobID)
		if e != nil {
			return nil, e
		}
		if len(stats) == 0 {
			continue
		}
		statsMap[triggerType] = stats
	}
	return statsMap, nil
}

func (je *JobExecutor) CancelJobExecution(id uint) error {
	item := je.executions[id]
	if item != nil {
		item.cancel()
	}
	return nil
}

func (je *JobExecutor) IsJobExecutionRunning(id uint) bool {
	item := je.executions[id]
	if item == nil {
		return false
	}
	return item.Status == types.JobExecutionRunning
}

func (je *JobExecutor) addJobExecution(exec *jobExecutionItem) {
	je.mu.Lock()
	defer je.mu.Unlock()
	je.executions[exec.ID] = exec
}

func (je *JobExecutor) removeJobExecution(id uint) {
	je.mu.Lock()
	defer je.mu.Unlock()
	delete(je.executions, id)
}

func (je *JobExecutor) Dispose() error {
	je.mu.Lock()
	defer je.mu.Unlock()

	for _, trigger := range je.triggers {
		trigger.Dispose()
	}

	for _, exec := range je.executions {
		exec.cancel()
		je.updateJobExecutionResult(exec, errors.New("aborted"))
	}
	return nil
}

type jobExecutionItem struct {
	*types.JobExecution
	cancel func()
	logger *jobExecutionLogger
}

func newJobExecutionLogger(jid uint, onLog func(string)) *jobExecutionLogger {
	return &jobExecutionLogger{jid: jid, onLog: onLog}
}

type jobExecutionLogger struct {
	jid   uint
	onLog func(string)
	logs  strings.Builder
	mu    sync.RWMutex
}

func (jel *jobExecutionLogger) Log(s string) {
	log.Printf("[JobExecutor] [%d] %s\n", jel.jid, s)
	if jel.onLog != nil {
		jel.onLog(s)
	}
	jel.mu.Lock()
	defer jel.mu.Unlock()
	jel.logs.WriteString(s)
	jel.logs.WriteRune('\n')
}

func (jel *jobExecutionLogger) String() string {
	jel.mu.RLock()
	defer jel.mu.RUnlock()
	return jel.logs.String()
}
