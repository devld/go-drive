package storage

import (
	"go-drive/common/registry"
	"go-drive/common/types"

	"gorm.io/gorm"
)

type ScheduledDAO struct {
	db *DB
}

func NewScheduledDAO(db *DB, ch *registry.ComponentsHolder) *ScheduledDAO {
	dao := &ScheduledDAO{db}
	ch.Add("scheduledDAO", dao)
	return dao
}

func (s *ScheduledDAO) GetJobs(includeDisabled bool) ([]types.Job, error) {
	jobs := make([]types.Job, 0)
	tx := s.db.C()
	if !includeDisabled {
		tx = tx.Where("`enabled` = ?", true)
	}
	return jobs, tx.Find(&jobs).Error
}

func (s *ScheduledDAO) AddJob(j types.Job) (types.Job, error) {
	return j, s.db.C().Create(&j).Error
}

func (s *ScheduledDAO) UpdateJob(id uint, j types.Job) error {
	j.ID = id
	return s.db.C().Save(j).Error
}

func (s *ScheduledDAO) DeleteJob(id uint) error {
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

func (s *ScheduledDAO) GetJobExecutions(jobId uint) ([]types.JobExecution, error) {
	jes := make([]types.JobExecution, 0)
	tx := s.db.C().Order("`started_at` DESC")
	if jobId != 0 {
		tx = tx.Where("`job_id` = ?", jobId)
	}
	return jes, tx.Find(&jes).Error
}

func (s *ScheduledDAO) AddJobExecution(je *types.JobExecution) error {
	return s.db.C().Create(je).Error
}

func (s *ScheduledDAO) UpdateJobExecution(je *types.JobExecution) error {
	if je.ID == 0 {
		panic("JobExecution update without ID")
	}
	return s.db.C().Save(je).Error
}

func (s *ScheduledDAO) DeleteJobExecution(id uint) error {
	return s.db.C().Delete(&types.JobExecution{}, "`id` = ?", id).Error
}

func (s *ScheduledDAO) DeleteJobExecutions(jobId uint) error {
	return s.db.C().Delete(&types.JobExecution{},
		"`job_id` = ? and ( `status` = ? or `status` = ? )",
		jobId, types.JobExecutionSuccess, types.JobExecutionFailed,
	).Error
}

func (s *ScheduledDAO) UpdateAllRunningJobExecutionsToFailed() error {
	return s.db.C().Model(&types.JobExecution{}).
		Where("`status` = ?", types.JobExecutionRunning).
		Update("`status`", types.JobExecutionFailed).Error
}
