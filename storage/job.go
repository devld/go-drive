package storage

import (
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/registry"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type JobDAO struct {
	db *DB
}

func NewJobDAO(db *DB, ch *registry.ComponentsHolder) *JobDAO {
	dao := &JobDAO{db}
	ch.Add("jobDAO", dao)
	return dao
}

func (s *JobDAO) GetJobs(includeDisabled bool) ([]types.Job, error) {
	jobs := make([]types.Job, 0)
	tx := s.db.C()
	if !includeDisabled {
		tx = tx.Where("`enabled` = ?", true)
	}
	return jobs, tx.Find(&jobs).Error
}

func (s *JobDAO) GetJob(id uint) (types.Job, error) {
	job := types.Job{}
	e := s.db.C().Where("`id` = ?", id).First(&job).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return job, err.NewNotFoundError()
	}
	return job, e
}

func (s *JobDAO) AddJob(j types.Job) (types.Job, error) {
	return j, s.db.C().Create(&j).Error
}

func (s *JobDAO) UpdateJob(id uint, j types.Job) error {
	j.ID = id
	return s.db.C().Save(j).Error
}

func (s *JobDAO) DeleteJob(id uint) error {
	return s.db.C().Transaction(func(tx *gorm.DB) error {
		if e := tx.Delete(&types.Job{}, "`id` = ?", id).Error; e != nil {
			return e
		}
		return tx.Delete(&types.JobExecution{},
			"`job_id` = ? and ( `status` = ? or `status` = ? )",
			id, types.JobExecutionSuccess, types.JobExecutionFailed,
		).Error
	})
}

func (s *JobDAO) GetJobExecutions(jobId uint) ([]types.JobExecution, error) {
	jes := make([]types.JobExecution, 0)
	tx := s.db.C().Order("`started_at` DESC")
	if jobId != 0 {
		tx = tx.Where("`job_id` = ?", jobId)
	}
	return jes, tx.Find(&jes).Error
}

func (s *JobDAO) AddJobExecution(je *types.JobExecution) error {
	return s.db.C().Create(je).Error
}

func (s *JobDAO) UpdateJobExecution(je *types.JobExecution) error {
	if je.ID == 0 {
		panic("JobExecution update without ID")
	}
	return s.db.C().Save(je).Error
}

func (s *JobDAO) DeleteJobExecution(id uint) error {
	return s.db.C().Delete(&types.JobExecution{}, "`id` = ?", id).Error
}

func (s *JobDAO) DeleteJobExecutions(jobId uint) error {
	return s.db.C().Delete(&types.JobExecution{},
		"`job_id` = ? and ( `status` = ? or `status` = ? )",
		jobId, types.JobExecutionSuccess, types.JobExecutionFailed,
	).Error
}

func (s *JobDAO) UpdateAllRunningJobExecutionsToFailed() error {
	return s.db.C().Model(&types.JobExecution{}).
		Where("`status` = ?", types.JobExecutionRunning).
		Update("status", types.JobExecutionFailed).Error
}
