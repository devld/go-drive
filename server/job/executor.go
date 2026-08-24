package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/storage"
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
	runner := ch.Get(registry.KeyTaskRunner).(task.Runner)

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

	ch.Add(registry.KeyJobExecutor, executor)
	return executor, nil
}

func (je *JobExecutor) ReloadJobs() error {
	je.mu.Lock()
	defer je.mu.Unlock()

	jobs, e := je.jobDAO.GetJobs(false)
	if e != nil {
		logging.For("job").Errorf("job reload failed: %v", e)
		return e
	}

	for _, trigger := range je.triggers {
		trigger.Clear()
	}

	// Parse triggers and register them
	for _, job := range jobs {
		triggers, e := je.parseTriggers(job)
		if e != nil {
			logging.For("job").Warnf("error parsing triggers for job %d: %v", job.ID, e)
			continue
		}

		for _, trigger := range triggers {
			triggerInstance := je.triggers[trigger.Type]
			if triggerInstance == nil {
				continue
			}
			if e := triggerInstance.Register(job.ID, trigger.Config); e != nil {
				logging.For("job").Warnf("error registering trigger %s for job %d: %v", string(trigger.Type), job.ID, e)
				continue
			}
		}
	}
	logging.For("job").Debugf("jobs reloaded jobs=%d triggers=%d", len(jobs), len(je.triggers))

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
		logging.For("job").Warnf("job trigger failed job_id=%d reason=load_job: %v", jobID, e)
		return task.Task{}, e
	}

	created, e := je.runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return nil, je.ExecuteJobSync(ctx, job, event, nil)
	}, task.WithNameGroup(job.Description, "job/execution"))
	if e != nil {
		logging.For("job").Warnf("job trigger queue failed job_id=%d: %v", jobID, e)
		return task.Task{}, e
	}
	logging.For("job").Debugf("job triggered job_id=%d task_id=%s event=%s", jobID, created.Id, event.Type)
	return created, nil
}

func (je *JobExecutor) ExecuteJobSync(ctx context.Context, job types.Job, event TriggerEvent, onLog func(string)) error {
	started := time.Now()
	jobExecution, e := je.newJobExecution(job)
	if e != nil {
		logging.For("job").Errorf("job execution creation failed job_id=%d: %v", job.ID, e)
		return e
	}
	logger := newJobExecutionLogger(job.ID, jobExecution.ID, onLog)
	e = je.executeJob(ctx, job, jobExecution, logger, &event)
	logging.For("job").Debugf("job execution completed job_id=%d execution_id=%d duration=%s status=%s",
		job.ID, jobExecution.ID, time.Since(started), jobExecution.Status)
	return e
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
		logging.For("job").Errorf("job action not found job_id=%d action=%s", job.ID, job.Action)
		e = errors.New("job not found")
		return
	}

	params := make(types.SM, 0)
	e = json.Unmarshal([]byte(job.ActionParams), &params)
	if e != nil {
		logging.For("job").Errorf("job action params invalid job_id=%d: %v", job.ID, e)
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
	if e == nil {
		logging.For("job").Debugf("job execution started job_id=%d execution_id=%d", job.ID, jobExecution.ID)
	}
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
		logging.For("job").Errorf("failed to update job execution: %v", e)
	}
	duration := time.Duration(0)
	if item.CompletedAt >= item.StartedAt {
		duration = time.Duration(item.CompletedAt-item.StartedAt) * time.Millisecond
	}
	if e != nil {
		logging.For("job").Errorf("job execution failed job_id=%d execution_id=%d duration=%s: %v",
			item.JobId, item.ID, duration, e)
	} else {
		logging.For("job").Debugf("job execution succeeded job_id=%d execution_id=%d duration=%s",
			item.JobId, item.ID, duration)
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
		logging.For("job").Debugf("job execution canceled execution_id=%d job_id=%d", id, item.JobId)
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

func newJobExecutionLogger(jobID, executionID uint, onLog func(string)) *jobExecutionLogger {
	return &jobExecutionLogger{jobID: jobID, executionID: executionID, onLog: onLog}
}

type jobExecutionLogger struct {
	jobID       uint
	executionID uint
	onLog       func(string)
	logs        strings.Builder
	mu          sync.RWMutex
}

func (jel *jobExecutionLogger) Log(message string) {
	logging.For("job").Infof("[job_id=%d execution_id=%d] %s", jel.jobID, jel.executionID, message)
	if jel.onLog != nil {
		jel.onLog(message)
	}
	jel.mu.Lock()
	defer jel.mu.Unlock()
	jel.logs.WriteString(message)
	jel.logs.WriteRune('\n')
}

func (jel *jobExecutionLogger) String() string {
	jel.mu.RLock()
	defer jel.mu.RUnlock()
	return jel.logs.String()
}
