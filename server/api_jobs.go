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

func InitJobsRoutes(
	r gin.IRouter,
	ch *registry.ComponentsHolder,
	runner task.Runner,
	tokenStore types.TokenStore,
	jobExecutor *scheduled.JobExecutor,
	scheduledDAO *storage.ScheduledDAO) error {

	r = r.Group("/admin/jobs", TokenAuth(tokenStore), AdminGroupRequired())

	// get all job definitions
	r.GET("/definitions", func(c *gin.Context) {
		SetResult(c, scheduled.GetJobs())
	})

	// get all created jobs
	r.GET("", func(c *gin.Context) {
		jobs, e := scheduledDAO.GetJobs(true)
		if e != nil {
			_ = c.Error(e)
			return
		}
		items := make([]jobItem, 0, len(jobs))
		for _, job := range jobs {
			var nextRun *time.Time
			if j := jobExecutor.GetJob(job.ID); j != nil {
				t := j.NextRun()
				nextRun = &t
			}
			items = append(items, jobItem{
				Job:     job,
				NextRun: nextRun,
			})
		}
		SetResult(c, items)
	})

	// create job
	r.POST("", func(c *gin.Context) {
		job := types.Job{}
		if e := c.Bind(&job); e != nil {
			_ = c.Error(e)
			return
		}
		if e := jobExecutor.ValidateSchedule(job.Schedule); e != nil {
			_ = c.Error(e)
			return
		}
		addJob, e := scheduledDAO.AddJob(job)
		if e != nil {
			_ = c.Error(e)
			return
		}
		e = jobExecutor.ReloadJobs()
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, addJob)
	})

	// update job
	r.PUT("/:id", func(c *gin.Context) {
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
		if e := jobExecutor.ValidateSchedule(job.Schedule); e != nil {
			_ = c.Error(e)
			return
		}
		e := scheduledDAO.UpdateJob(id, job)
		if e != nil {
			_ = c.Error(e)
			return
		}
		e = jobExecutor.ReloadJobs()
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete job
	r.DELETE("/:id", func(c *gin.Context) {
		id := utils.ToUInt(c.Param("id"), 0)
		if id == 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		e := scheduledDAO.DeleteJob(id)
		if e != nil {
			_ = c.Error(e)
			return
		}
		e = jobExecutor.ReloadJobs()
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// get all executions
	r.GET("/executions", func(c *gin.Context) {
		jobId := utils.ToInt(c.Query("jobId"), -1)
		if jobId < 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		result, e := scheduledDAO.GetJobExecutions(uint(jobId))
		if e != nil {
			_ = c.Error(e)
			return
		}
		SetResult(c, result)
	})

	// execute a job
	r.POST("/execution", func(c *gin.Context) {
		jobId := utils.ToInt(c.Query("jobId"), -1)
		if jobId < 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		job, e := scheduledDAO.GetJob(uint(jobId))
		if e != nil {
			_ = c.Error(e)
			return
		}

		w := c.Writer
		e = ExecuteTaskStreaming(c, runner,
			func(ctx types.TaskCtx) (interface{}, error) {
				e := jobExecutor.TriggerExecution(ctx, job, func(s string) {
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
	})

	// cancel job execution
	r.PUT("/execution/:id/cancel", func(c *gin.Context) {
		id := utils.ToUInt(c.Param("id"), 0)
		if id == 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		e := jobExecutor.CancelJobExecution(id)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete job execution
	r.DELETE("/execution/:id", func(c *gin.Context) {
		id := utils.ToUInt(c.Param("id"), 0)
		if id == 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		if jobExecutor.IsJobExecutionRunning(id) {
			_ = c.Error(err.NewNotAllowedError())
			return
		}
		e := scheduledDAO.DeleteJobExecution(id)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	// delete job executions by jobId
	r.DELETE("/execution", func(c *gin.Context) {
		id := utils.ToUInt(c.Query("jobId"), 0)
		if id == 0 {
			_ = c.Error(err.NewBadRequestError(""))
			return
		}
		if jobExecutor.IsJobExecutionRunning(id) {
			_ = c.Error(err.NewNotAllowedError())
			return
		}
		e := scheduledDAO.DeleteJobExecutions(id)
		if e != nil {
			_ = c.Error(e)
			return
		}
	})

	r.POST("/script-eval", func(c *gin.Context) {
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
		e = ExecuteTaskStreaming(c, runner,
			func(ctx types.TaskCtx) (interface{}, error) {
				e := scheduled.ExecuteJobCode(c.Request.Context(), code, ch, func(s string) {
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
	})

	return nil
}

type jobItem struct {
	types.Job
	NextRun *time.Time `json:"nextRun"`
}
