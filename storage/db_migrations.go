package storage

import (
	"encoding/json"
	"go-drive/common/types"
	"log"

	"gorm.io/gorm"
)

var initSQL = []string{
	"INSERT INTO `users`(`username`, `password`) VALUES ('admin', '$2y$10$Xqn8qV2D2KY2ceI5esM/JOiKTPKJFbkSzzuhce89BxygvCqnhyk3m')", // 123456
	"INSERT INTO `groups`(`name`) VALUES ('admin')",
	"INSERT INTO `user_groups`(`username`, `group_name`) VALUES ('admin', 'admin')",
	"INSERT INTO `path_permissions`(`path`, `subject`, `permission`, `policy`) VALUES ('', 'ANY', 1, 1)",
}

func tryInitDbData(db *gorm.DB) error {
	var n int64 = 0
	if e := db.Model(&types.User{}).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, initSQL := range initSQL {
			if e := tx.Exec(initSQL).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

// migrateJobScheduleToTriggers migrates deprecated Schedule/Job/Params to Triggers/Action/ActionParams
func migrateJobScheduleToTriggers(db *gorm.DB) error {
	var jobs []types.Job
	if e := db.Find(&jobs).Error; e != nil {
		return e
	}

	migrated := 0
	for _, job := range jobs {
		updates := make(map[string]interface{})
		needUpdate := false

		// Schedule -> Triggers (cron)
		if job.Triggers == "_need_migration_" && job.DeprecatedSchedule != "" {
			trigger := map[string]any{
				"type":   "cron",
				"config": map[string]any{"schedule": job.DeprecatedSchedule},
			}
			triggers := []map[string]any{trigger}
			triggersJSON, e := json.Marshal(triggers)
			if e != nil {
				log.Printf("error marshaling triggers for job %d: %v", job.ID, e)
				continue
			}
			updates["triggers"] = string(triggersJSON)
			needUpdate = true
		}

		// Job -> Action
		if job.Action == "_need_migration_" && job.DeprecatedJob != "" {
			updates["action"] = job.DeprecatedJob
			needUpdate = true
		}

		// Params -> ActionParams
		if job.ActionParams == "_need_migration_" && job.DeprecatedParams != "" {
			updates["action_params"] = job.DeprecatedParams
			needUpdate = true
		}

		if !needUpdate {
			continue
		}

		if e := db.Model(&types.Job{}).Where("id = ?", job.ID).Updates(updates).Error; e != nil {
			log.Printf("error migrating job %d: %v", job.ID, e)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("migrated %d jobs from Schedule/Job/Params to Triggers/Action/ActionParams", migrated)
	}

	return nil
}
