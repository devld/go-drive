package server

import (
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/scheduled"
	"go-drive/storage"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type jobsRoute struct {
	ch           *registry.ComponentsHolder
	runner       task.Runner
	jobExecutor  *scheduled.JobExecutor
	scheduledDAO *storage.ScheduledDAO
}

func (jr *jobsRoute) getJobsDefinitions(c *gin.Context) {
	SetResult(c, scheduled.GetJobs())
}

func (jr *jobsRoute) getJobs(c *gin.Context) {
	jobs, e := jr.scheduledDAO.GetJobs(true)
	if e != nil {
		_ = c.Error(e)
		return
	}
	items := make([]jobItem, 0, len(jobs))
	for _, job := range jobs {
		var nextRun *time.Time
		if j := jr.jobExecutor.GetJob(job.ID); j != nil {
			t := j.NextRun()
			nextRun = &t
		}
		items = append(items, jobItem{
			Job:     job,
			NextRun: nextRun,
		})
	}
	SetResult(c, items)
}

func (jr *jobsRoute) createJob(c *gin.Context) {
	job := types.Job{}
	if e := c.Bind(&job); e != nil {
		_ = c.Error(e)
		return
	}
	if e := jr.jobExecutor.ValidateSchedule(job.Schedule); e != nil {
		_ = c.Error(e)
		return
	}
	addJob, e := jr.scheduledDAO.AddJob(job)
	if e != nil {
		_ = c.Error(e)
		return
	}
	e = jr.jobExecutor.ReloadJobs()
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, addJob)
}

func (jr *jobsRoute) updateJob(c *gin.Context) {
	job := types.Job{}
	if e := c.Bind(&job); e != nil {
		_ = c.Error(e)
		return
	}
	id := utils.ToUInt(c.Param("id"), 0)
	if id == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	if e := jr.jobExecutor.ValidateSchedule(job.Schedule); e != nil {
		_ = c.Error(e)
		return
	}
	e := jr.scheduledDAO.UpdateJob(id, job)
	if e != nil {
		_ = c.Error(e)
		return
	}
	e = jr.jobExecutor.ReloadJobs()
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) deleteJob(c *gin.Context) {
	id := utils.ToUInt(c.Param("id"), 0)
	if id == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	e := jr.scheduledDAO.DeleteJob(id)
	if e != nil {
		_ = c.Error(e)
		return
	}
	e = jr.jobExecutor.ReloadJobs()
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) getAllExecutions(c *gin.Context) {
	jobId := utils.ToInt(c.Query("jobId"), -1)
	if jobId < 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	result, e := jr.scheduledDAO.GetJobExecutions(uint(jobId))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, result)
}

func (jr *jobsRoute) executeJob(c *gin.Context) {
	jobId := utils.ToInt(c.Query("jobId"), -1)
	if jobId < 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	job, e := jr.scheduledDAO.GetJob(uint(jobId))
	if e != nil {
		_ = c.Error(e)
		return
	}

	w := c.Writer
	e = ExecuteTaskStreaming(c, jr.runner,
		func(ctx types.TaskCtx) (interface{}, error) {
			e := jr.jobExecutor.TriggerExecution(ctx, job, func(s string) {
				_, _ = w.Write([]byte(s + "\n"))
				w.Flush()
			})
			if e != nil {
				w.Write([]byte(e.Error()))
			}
			return nil, e
		},
		task.WithNameGroup(job.Description, "scheduled/execution"),
	)

	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) cancelJobExecution(c *gin.Context) {
	id := utils.ToUInt(c.Param("id"), 0)
	if id == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	e := jr.jobExecutor.CancelJobExecution(id)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) deleteJobExecution(c *gin.Context) {
	id := utils.ToUInt(c.Param("id"), 0)
	if id == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	if jr.jobExecutor.IsJobExecutionRunning(id) {
		_ = c.Error(err.NewNotAllowedError())
		return
	}
	e := jr.scheduledDAO.DeleteJobExecution(id)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) deleteJobExecutionsByJobId(c *gin.Context) {
	id := utils.ToUInt(c.Query("jobId"), 0)
	if id == 0 {
		_ = c.Error(err.NewBadRequestError(""))
		return
	}
	if jr.jobExecutor.IsJobExecutionRunning(id) {
		_ = c.Error(err.NewNotAllowedError())
		return
	}
	e := jr.scheduledDAO.DeleteJobExecutions(id)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

func (jr *jobsRoute) scriptEval(c *gin.Context) {
	code, e := io.ReadAll(c.Request.Body)
	if e != nil {
		_ = c.Error(e)
		return
	}
	w := c.Writer
	taskName := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(string(code)), " ")
	if len(taskName) > 20 {
		taskName = taskName[:20]
	}
	e = ExecuteTaskStreaming(c, jr.runner,
		func(ctx types.TaskCtx) (interface{}, error) {
			e := scheduled.ExecuteJobCode(c.Request.Context(), code, jr.ch, func(s string) {
				_, _ = w.Write([]byte(s + "\n"))
				w.Flush()
			})
			if e != nil {
				w.Write([]byte("ERROR: " + e.Error()))
			}
			return nil, e
		},
		task.WithNameGroup(taskName, "scheduled/script-eval"),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

type jobItem struct {
	types.Job
	NextRun *time.Time `json:"nextRun"`
}
