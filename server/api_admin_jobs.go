package server

import (
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/server/job"
	"go-drive/storage"
	"io"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type jobsRoute struct {
	ch          *registry.ComponentsHolder
	runner      task.Runner
	jobExecutor *job.JobExecutor
	jobDAO      *storage.JobDAO
}

func (jr *jobsRoute) getJobsDefinitions(c *gin.Context) {
	SetResult(c, types.M{
		"triggers": job.GetTriggerDefs(),
		"actions":  job.GetActionDefs(),
	})
}

func (jr *jobsRoute) getJobs(c *gin.Context) {
	jobs, e := jr.jobDAO.GetJobs(true)
	if e != nil {
		_ = c.Error(e)
		return
	}
	items := make([]jobItem, 0, len(jobs))
	for _, job := range jobs {
		triggersInfo, e := jr.jobExecutor.GetJobTriggersInfo(job.ID)
		if e != nil {
			_ = c.Error(e)
			return
		}
		items = append(items, jobItem{
			Job:          job,
			TriggersInfo: triggersInfo,
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

	if e := jr.jobExecutor.ValidateTriggers(job.Triggers); e != nil {
		_ = c.Error(e)
		return
	}
	addJob, e := jr.jobDAO.AddJob(job)
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

	if e := jr.jobExecutor.ValidateTriggers(job.Triggers); e != nil {
		_ = c.Error(e)
		return
	}
	e := jr.jobDAO.UpdateJob(id, job)
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
	e := jr.jobDAO.DeleteJob(id)
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
	result, e := jr.jobDAO.GetJobExecutions(uint(jobId))
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
	jobObj, e := jr.jobDAO.GetJob(uint(jobId))
	if e != nil {
		_ = c.Error(e)
		return
	}

	w := c.Writer
	e = ExecuteTaskStreaming(c, jr.runner,
		func(ctx types.TaskCtx) (any, error) {
			e := jr.jobExecutor.ExecuteJobSync(ctx, jobObj, job.TriggerEvent{}, func(s string) {
				_, _ = w.Write([]byte(s + "\n"))
				w.Flush()
			})
			if e != nil {
				w.Write([]byte(e.Error()))
			}
			return nil, e
		},
		task.WithNameGroup(jobObj.Description, "job/execution"),
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
	e := jr.jobDAO.DeleteJobExecution(id)
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
	e := jr.jobDAO.DeleteJobExecutions(id)
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
			e := job.ExecuteJobCode(c.Request.Context(), code, nil, jr.ch, func(s string) {
				_, _ = w.Write([]byte(s + "\n"))
				w.Flush()
			})
			if e != nil {
				w.Write([]byte("ERROR: " + e.Error()))
			}
			return nil, e
		},
		task.WithNameGroup(taskName, "job/script-eval"),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}
}

type jobItem struct {
	types.Job
	TriggersInfo map[job.JobTriggerType][]types.SM `json:"triggersInfo"`
}
