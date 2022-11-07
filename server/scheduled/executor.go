package scheduled

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/robfig/cron/v3"
)

type JobExecutor struct {
	ch           *registry.ComponentsHolder
	s            *gocron.Scheduler
	scheduledDAO *storage.ScheduledDAO

	executions map[uint]*jobExecutionItem

	mu sync.Mutex
}

func NewJobExecutor(scheduledDAO *storage.ScheduledDAO, ch *registry.ComponentsHolder) (*JobExecutor, error) {
	executor := &JobExecutor{
		ch:           ch,
		s:            gocron.NewScheduler(time.Local),
		scheduledDAO: scheduledDAO,
		executions:   make(map[uint]*jobExecutionItem),
	}
	executor.s.TagsUnique()

	e := executor.ReloadJobs()
	if e != nil {
		return nil, e
	}

	executor.s.StartAsync()

	_ = scheduledDAO.UpdateAllRunningJobExecutionsToFailed()

	ch.Add("jobExecutor", executor)
	return executor, nil
}

func (je *JobExecutor) ReloadJobs() error {
	je.mu.Lock()
	defer je.mu.Unlock()

	jobs, e := je.scheduledDAO.GetJobs(false)
	if e != nil {
		return e
	}

	je.s.Clear()

	for _, job := range jobs {
		job, e := je.s.
			Cron(job.Schedule).
			Tag(strconv.FormatUint(uint64(job.ID), 10)).
			Do(je.jobExecutor, job)
		if e != nil {
			log.Printf("error creating job: %v", e)
			continue
		}
		job.SingletonMode()
	}

	return nil
}

func (je *JobExecutor) jobExecutor(job types.Job) {
	jobExecution := &types.JobExecution{
		JobId:     job.ID,
		StartedAt: uint64(time.Now().UnixMilli()),
		Status:    types.JobExecutionRunning,
	}
	e := je.scheduledDAO.AddJobExecution(jobExecution)
	if e != nil {
		log.Printf("failed to save job execution: %v", e)
		return
	}

	executionCtx, cancel := context.WithCancel(context.Background())
	logger := newJobExecutionLogger((jobExecution.ID))
	item := &jobExecutionItem{JobExecution: jobExecution, cancel: cancel, logger: logger}
	je.addJobExecution(item)

	defer func() {
		je.updateJobExecutionResult(item, e)
	}()

	jobDefinition := GetJob(job.Job)
	if jobDefinition == nil {
		e = errors.New("job not found")
		return
	}

	params := make(types.SM, 0)
	e = json.Unmarshal([]byte(job.Params), &params)
	if e != nil {
		e = fmt.Errorf("failed to parse params: %s", e.Error())
		return
	}

	e = jobDefinition.Do(executionCtx, params, je.ch, item.logger.Log)
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
	if e := je.scheduledDAO.UpdateJobExecution(item.JobExecution); e != nil {
		log.Printf("failed to update job execution: %v", e)
	}
	item.cancel()
	je.removeJobExecution(item.ID)
}

func (je *JobExecutor) ValidateSchedule(s string) error {
	_, e := cron.ParseStandard(s)
	if e != nil {
		return err.NewNotAllowedMessageError(e.Error())
	}
	return nil
}

func (je *JobExecutor) GetJob(id uint) *gocron.Job {
	jobs, e := je.s.FindJobsByTag(strconv.FormatUint(uint64(id), 10))
	if e != nil {
		return nil
	}
	return jobs[0]
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
	je.s.Stop()
	je.s.Clear()
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

func newJobExecutionLogger(jid uint) *jobExecutionLogger {
	return &jobExecutionLogger{jid: jid}
}

type jobExecutionLogger struct {
	jid  uint
	logs strings.Builder
	mu   sync.RWMutex
}

func (jel *jobExecutionLogger) Log(s string) {
	log.Printf("[JobExecutor] [%d] %s\n", jel.jid, s)
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
