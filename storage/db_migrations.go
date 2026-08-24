package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-drive/common/driveutil"
	"go-drive/common/logging"
	"go-drive/common/types"
	"strings"
	"time"

	"gorm.io/gorm"
)

// A negative version runs on every startup and does not advance the stored
// schema version. Positive versions run once when they are greater than the
// stored version.
type dbMigration struct {
	version int
	name    string
	run     func(*gorm.DB) error
}

var dbMigrations = []dbMigration{
	{version: -1, name: "legacy_job_schema", run: migrateLegacyJobSchema},
	{version: -1, name: "auto_migrate", run: autoMigrateModels},
	{version: -1, name: "init_db_data", run: tryInitDbData},
	{version: 1, name: "job_schedule_to_triggers", run: migrateJobScheduleToTriggers},
	{version: 2, name: "script_drive_configs", run: migrateScriptDriveConfigs},
	{version: 3, name: "oauth_drive_data", run: migrateOAuthDriveData},
	{version: 4, name: "drop_legacy_job_columns", run: migrateLegacyJobSchema},
}

func migrateAll(db *gorm.DB) error {
	return runDBMigrations(db, dbMigrations)
}

func runDBMigrations(db *gorm.DB, migrations []dbMigration) error {
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	if e := ensureSchemaMigrationTable(sqlDB); e != nil {
		return e
	}
	current, e := currentMigrationVersion(sqlDB)
	if e != nil {
		return e
	}
	logging.For("db-mgrt").Debugf("database migrations started current=%d", current)
	for _, migration := range migrations {
		started := time.Now()
		if migration.version < 0 {
			logging.For("db-mgrt").Debugf("migration started version=%d name=%s", migration.version, migration.name)
			if e := migration.run(db); e != nil {
				logging.For("db-mgrt").Errorf("migration failed version=%d name=%s duration=%s: %v", migration.version, migration.name, time.Since(started), e)
				return fmt.Errorf("migration %s: %w", migration.name, e)
			}
			logging.For("db-mgrt").Debugf("migration completed version=%d name=%s duration=%s", migration.version, migration.name, time.Since(started))
			continue
		}
		if migration.version <= current {
			continue
		}
		logging.For("db-mgrt").Infof("applying database migration %d (%s)", migration.version, migration.name)
		if e := migration.run(db); e != nil {
			logging.For("db-mgrt").Errorf("migration failed version=%d name=%s duration=%s: %v", migration.version, migration.name, time.Since(started), e)
			return fmt.Errorf("migration %d (%s): %w", migration.version, migration.name, e)
		}
		if e := setMigrationVersion(sqlDB, migration.version); e != nil {
			logging.For("db-mgrt").Errorf("migration version update failed version=%d name=%s duration=%s: %v", migration.version, migration.name, time.Since(started), e)
			return fmt.Errorf("record migration %d (%s): %w", migration.version, migration.name, e)
		}
		current = migration.version
		logging.For("db-mgrt").Debugf("migration completed version=%d name=%s duration=%s", migration.version, migration.name, time.Since(started))
	}
	logging.For("db-mgrt").Debugf("database migrations completed version=%d", current)
	return nil
}

func autoMigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&types.User{},
		&types.Group{},
		&types.UserGroup{},
		&types.Drive{},
		&types.PathPermission{},
		&types.PathMount{},
		&types.DriveData{},
		&types.DriveCache{},
		&types.Option{},
		&types.Job{},
		&types.JobExecution{},
		&types.PathMeta{},
		&types.FileBucket{},
		&types.Session{},
	)
}

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
		logging.For("db-mgrt").Debugf("database seed data already exists users=%d", n)
		return nil
	}
	e := db.Transaction(func(tx *gorm.DB) error {
		for _, initSQL := range initSQL {
			if e := tx.Exec(initSQL).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if e == nil {
		logging.For("db-mgrt").Infof("initialized database seed data statements=%d", len(initSQL))
	}
	return e
}

func migrateJobScheduleToTriggers(db *gorm.DB) error {
	return migrateLegacyJobSchema(db)
}

const (
	jobNeedMigrationSentinel = "_need_migration_"
	jobsMigrateTempTable     = "jobs__migrate"
)

var (
	legacyJobColumns = []string{"schedule", "job", "params"}
	jobTargetColumns = []string{"triggers", "action", "action_params"}
)

// migrateLegacyJobSchema copies Schedule/Job/Params into Triggers/Action/ActionParams
// when those legacy columns still exist, then drops them and any leftover
// `_need_migration_` column defaults. New databases never create the old columns.
func migrateLegacyJobSchema(db *gorm.DB) error {
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	dialect := db.Dialector.Name()
	exists, e := hasTable(sqlDB, dialect, "jobs")
	if e != nil || !exists {
		return e
	}
	if e := prepareLegacyJobTable(db); e != nil {
		return e
	}
	columns, e := listTableColumns(sqlDB, dialect, "jobs")
	if e != nil {
		return e
	}
	if e := copyLegacyJobValues(sqlDB, columns); e != nil {
		return e
	}
	if dialect == "sqlite" {
		if needsSQLiteJobsRebuild(columns) {
			return rebuildSQLiteJobsTable(db)
		}
		return nil
	}
	if e := dropLegacyJobColumns(sqlDB, columns); e != nil {
		return e
	}
	return dropMySQLJobNeedMigrationDefaults(sqlDB, columns)
}

func prepareLegacyJobTable(db *gorm.DB) error {
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	dialect := db.Dialector.Name()
	exists, e := hasTable(sqlDB, dialect, "jobs")
	if e != nil || !exists {
		return e
	}
	columns, e := listTableColumns(sqlDB, dialect, "jobs")
	if e != nil {
		return e
	}
	defs := map[string]string{
		"triggers":      "TEXT NOT NULL DEFAULT ''",
		"action":        "VARCHAR(64) NOT NULL DEFAULT ''",
		"action_params": "TEXT NOT NULL DEFAULT ''",
	}
	for _, name := range jobTargetColumns {
		if _, ok := columns[name]; ok {
			continue
		}
		if _, e := sqlDB.Exec("ALTER TABLE `jobs` ADD COLUMN `" + name + "` " + defs[name]); e != nil {
			return fmt.Errorf("add jobs.%s: %w", name, e)
		}
	}
	return nil
}

func copyLegacyJobValues(sqlDB *sql.DB, columns map[string]tableColumn) error {
	if !hasAnyColumn(columns, legacyJobColumns...) {
		return clearJobNeedMigrationValues(sqlDB, columns)
	}

	selectCols := []string{"`id`"}
	for _, name := range append(append([]string{}, legacyJobColumns...), jobTargetColumns...) {
		if _, ok := columns[name]; ok {
			selectCols = append(selectCols, "`"+name+"`")
		}
	}

	rows, e := sqlDB.Query("SELECT " + strings.Join(selectCols, ", ") + " FROM `jobs`")
	if e != nil {
		return e
	}

	type jobUpdate struct {
		id      string
		updates map[string]string
	}
	pending := make([]jobUpdate, 0)
	for rows.Next() {
		raw := make([]sql.NullString, len(selectCols))
		dest := make([]any, len(selectCols))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if e := rows.Scan(dest...); e != nil {
			_ = rows.Close()
			return e
		}
		values := make(map[string]string, len(selectCols))
		for i, name := range selectCols {
			key := strings.Trim(name, "`")
			if raw[i].Valid {
				values[key] = raw[i].String
			}
		}
		updates := make(map[string]string)
		if triggers, ok := migratedJobTriggers(values); ok {
			updates["triggers"] = triggers
		}
		if action, ok := migratedJobAction(values); ok {
			updates["action"] = action
		}
		if params, ok := migratedJobParams(values); ok {
			updates["action_params"] = params
		}
		if len(updates) == 0 {
			continue
		}
		pending = append(pending, jobUpdate{id: values["id"], updates: updates})
	}
	e = rows.Err()
	_ = rows.Close()
	if e != nil {
		return e
	}

	migrated := 0
	for _, item := range pending {
		if e := updateJobColumns(sqlDB, item.id, item.updates); e != nil {
			return fmt.Errorf("migrate job %s: %w", item.id, e)
		}
		migrated++
	}
	if migrated > 0 {
		logging.For("db-mgrt").Infof("migrated %d jobs from Schedule/Job/Params to Triggers/Action/ActionParams", migrated)
	}
	return nil
}

func clearJobNeedMigrationValues(sqlDB *sql.DB, columns map[string]tableColumn) error {
	if !hasAnyColumn(columns, jobTargetColumns...) {
		return nil
	}
	_, e := sqlDB.Exec(
		"UPDATE `jobs` SET `triggers` = CASE WHEN `triggers` = ? THEN '' ELSE `triggers` END, "+
			"`action` = CASE WHEN `action` = ? THEN '' ELSE `action` END, "+
			"`action_params` = CASE WHEN `action_params` = ? THEN '' ELSE `action_params` END",
		jobNeedMigrationSentinel, jobNeedMigrationSentinel, jobNeedMigrationSentinel,
	)
	return e
}

func migratedJobTriggers(values map[string]string) (string, bool) {
	if !needsJobFieldMigration(values["triggers"]) {
		return "", false
	}
	schedule := values["schedule"]
	if schedule == "" {
		if values["triggers"] == jobNeedMigrationSentinel {
			return "", true
		}
		return "", false
	}
	triggersJSON, e := json.Marshal([]map[string]any{{
		"type":   "cron",
		"config": map[string]any{"schedule": schedule},
	}})
	if e != nil {
		logging.For("db-mgrt").Warnf("error marshaling triggers for job %s: %v", values["id"], e)
		return "", false
	}
	return string(triggersJSON), true
}

func migratedJobAction(values map[string]string) (string, bool) {
	if !needsJobFieldMigration(values["action"]) {
		return "", false
	}
	if values["job"] != "" {
		return values["job"], true
	}
	if values["action"] == jobNeedMigrationSentinel {
		return "", true
	}
	return "", false
}

func migratedJobParams(values map[string]string) (string, bool) {
	if !needsJobFieldMigration(values["action_params"]) {
		return "", false
	}
	if values["params"] != "" {
		return values["params"], true
	}
	if values["action_params"] == jobNeedMigrationSentinel {
		return "", true
	}
	return "", false
}

func needsJobFieldMigration(value string) bool {
	return value == "" || value == jobNeedMigrationSentinel
}

func updateJobColumns(sqlDB *sql.DB, id string, updates map[string]string) error {
	sets := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+1)
	for _, name := range jobTargetColumns {
		value, ok := updates[name]
		if !ok {
			continue
		}
		sets = append(sets, "`"+name+"` = ?")
		args = append(args, value)
	}
	args = append(args, id)
	_, e := sqlDB.Exec("UPDATE `jobs` SET "+strings.Join(sets, ", ")+" WHERE `id` = ?", args...)
	return e
}

func dropLegacyJobColumns(sqlDB *sql.DB, columns map[string]tableColumn) error {
	for _, name := range legacyJobColumns {
		if _, ok := columns[name]; !ok {
			continue
		}
		if _, e := sqlDB.Exec("ALTER TABLE `jobs` DROP COLUMN `" + name + "`"); e != nil {
			return e
		}
	}
	return nil
}

func dropMySQLJobNeedMigrationDefaults(sqlDB *sql.DB, columns map[string]tableColumn) error {
	for _, name := range jobTargetColumns {
		info, ok := columns[name]
		if !ok || !isNeedMigrationDefault(info.defVal) {
			continue
		}
		if _, e := sqlDB.Exec("ALTER TABLE `jobs` ALTER `" + name + "` DROP DEFAULT"); e != nil {
			return e
		}
	}
	return nil
}

func needsSQLiteJobsRebuild(columns map[string]tableColumn) bool {
	if hasAnyColumn(columns, legacyJobColumns...) {
		return true
	}
	for _, name := range jobTargetColumns {
		if info, ok := columns[name]; ok && isNeedMigrationDefault(info.defVal) {
			return true
		}
	}
	return false
}

func isNeedMigrationDefault(value sql.NullString) bool {
	if !value.Valid {
		return false
	}
	return strings.Trim(value.String, "'\"") == jobNeedMigrationSentinel
}

type jobsMigrateTable struct {
	ID           uint   `gorm:"column:id;primaryKey;autoIncrement"`
	Description  string `gorm:"column:description;not null;type:text"`
	Triggers     string `gorm:"column:triggers;not null;type:text"`
	Action       string `gorm:"column:action;not null;type:string;size:64"`
	ActionParams string `gorm:"column:action_params;not null;type:text;size:512"`
	Enabled      bool   `gorm:"column:enabled;not null;type:bool"`
}

func (jobsMigrateTable) TableName() string {
	return jobsMigrateTempTable
}

func rebuildSQLiteJobsTable(db *gorm.DB) error {
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	if _, e := sqlDB.Exec("DROP TABLE IF EXISTS `" + jobsMigrateTempTable + "`"); e != nil {
		return e
	}
	if e := db.AutoMigrate(&jobsMigrateTable{}); e != nil {
		return e
	}
	if _, e := sqlDB.Exec("INSERT INTO `" + jobsMigrateTempTable +
		"` (`id`, `description`, `triggers`, `action`, `action_params`, `enabled`) " +
		"SELECT `id`, `description`, `triggers`, `action`, `action_params`, `enabled` FROM `jobs`"); e != nil {
		return e
	}
	if _, e := sqlDB.Exec("DROP TABLE `jobs`"); e != nil {
		return e
	}
	if _, e := sqlDB.Exec("ALTER TABLE `" + jobsMigrateTempTable + "` RENAME TO `jobs`"); e != nil {
		return e
	}
	var lastID int64
	if e := sqlDB.QueryRow("SELECT IFNULL(MAX(`id`), 0) FROM `jobs`").Scan(&lastID); e != nil {
		return e
	}
	if _, e := sqlDB.Exec("DELETE FROM `sqlite_sequence` WHERE `name` IN ('jobs', '" + jobsMigrateTempTable + "')"); e != nil {
		return e
	}
	_, e = sqlDB.Exec("INSERT INTO `sqlite_sequence`(`name`, `seq`) VALUES ('jobs', ?)", lastID)
	return e
}

var scriptDriveConfigFields = map[string][]string{
	"dropbox": {"client_id", "client_secret", "cache_ttl"},
	"qiniu":   {"bucket", "ak", "sk", "uploadURL", "downloadBaseURL", "cache_ttl"},
	"github":  {"owner", "repo", "branch", "token", "cache_ttl"},
}

func migrateScriptDriveConfigs(db *gorm.DB) error {
	var drives []types.Drive
	if e := db.Where("`type` = ?", "script").Find(&drives).Error; e != nil {
		return e
	}

	migrated := 0
	for _, drive := range drives {
		config := make(types.SM)
		if e := json.Unmarshal([]byte(drive.Config), &config); e != nil {
			logging.For("db-mgrt").Warnf("cannot migrate script drive %q: invalid config: %v", drive.Name, e)
			continue
		}
		if config == nil {
			config = make(types.SM)
		}

		data, e := loadScriptDriveData(db, drive.Name)
		if e != nil {
			return fmt.Errorf("load data for script drive %q: %w", drive.Name, e)
		}
		if _, ok := config["script"]; !ok {
			if _, ok := config["pool"]; !ok {
				if _, ok := config["__script"]; !ok {
					if _, ok := data["_script"]; !ok {
						continue
					}
				}
			}
		}

		scriptName := strings.TrimSuffix(config["__script"], ".js")
		if scriptName == "" {
			scriptName = strings.TrimSuffix(config["script"], ".js")
		}
		if scriptName == "" {
			scriptName = strings.TrimSuffix(data["_script"], ".js")
		}
		fields, knownScript := scriptDriveConfigFields[scriptName]
		if !knownScript {
			logging.For("db-mgrt").Warnf("cannot migrate script drive %q: unknown script %q", drive.Name, scriptName)
			continue
		}

		migratedConfig := make(types.SM, len(config)+len(fields)+2)
		for key, value := range config {
			migratedConfig[key] = value
		}
		changed := false
		if drive.Type != "script/"+scriptName {
			changed = true
		}
		if pool, ok := migratedConfig["pool"]; ok {
			if migratedConfig["__pool"] != pool {
				migratedConfig["__pool"] = pool
				changed = true
			}
		}

		clearKeys := make([]string, 0, len(fields)+1)
		for _, field := range fields {
			value, ok := data[field]
			if !ok {
				continue
			}
			if current, exists := migratedConfig[field]; !exists || current == "" {
				migratedConfig[field] = value
				changed = true
			}
			clearKeys = append(clearKeys, field)
		}
		if _, ok := data["_script"]; ok {
			clearKeys = append(clearKeys, "_script")
		}
		if _, ok := migratedConfig["__script"]; ok {
			delete(migratedConfig, "__script")
			changed = true
		}
		if _, ok := migratedConfig["script"]; ok {
			delete(migratedConfig, "script")
			changed = true
		}
		if _, ok := migratedConfig["pool"]; ok {
			delete(migratedConfig, "pool")
			changed = true
		}
		if !changed && len(clearKeys) == 0 {
			continue
		}

		encodedConfig, e := json.Marshal(migratedConfig)
		if e != nil {
			return fmt.Errorf("marshal config for script drive %q: %w", drive.Name, e)
		}
		if e := migrateScriptDriveRecord(db, drive.Name, "script/"+scriptName, string(encodedConfig), clearKeys); e != nil {
			return fmt.Errorf("migrate script drive %q: %w", drive.Name, e)
		}
		migrated++
	}

	if migrated > 0 {
		logging.For("db-mgrt").Infof("migrated %d Script Drive configuration(s)", migrated)
	}
	return nil
}

func loadScriptDriveData(db *gorm.DB, name string) (types.SM, error) {
	var rows []types.DriveData
	if e := db.Where("`drive` = ?", name).Find(&rows).Error; e != nil {
		return nil, e
	}
	data := make(types.SM, len(rows))
	for _, row := range rows {
		data[row.Key] = row.Value
	}
	return data, nil
}

func migrateScriptDriveRecord(db *gorm.DB, name, driveType, config string, clearKeys []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&types.Drive{}).
			Where("`name` = ?", name).
			Updates(map[string]any{"type": driveType, "config": config}).Error; e != nil {
			return e
		}
		for _, key := range clearKeys {
			if e := tx.Where("`drive` = ? AND `data_key` = ?", name, key).
				Delete(&types.DriveData{}).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

var oauthDataKeyMigrations = map[string]string{
	"token":         driveutil.DsKeyToken,
	"token_type":    driveutil.DsKeyTokenType,
	"expires_at":    driveutil.DsKeyExpiresAt,
	"refresh_token": driveutil.DsKeyRefreshToken,
	"state":         driveutil.DsKeyState,
}

func migrateOAuthDriveData(db *gorm.DB) error {
	var rows []types.DriveData
	if e := db.Find(&rows).Error; e != nil {
		return e
	}

	migrated := 0
	for _, row := range rows {
		newKey, ok := oauthDataKeyMigrations[row.Key]
		if !ok {
			continue
		}
		if e := migrateOAuthDriveDataRecord(db, row, newKey); e != nil {
			return fmt.Errorf("migrate OAuth data for drive %q key %q: %w", row.Drive, row.Key, e)
		}
		migrated++
	}

	if migrated > 0 {
		logging.For("db-mgrt").Infof("migrated %d OAuth data field(s) to the reserved namespace", migrated)
	}
	return nil
}

func migrateOAuthDriveDataRecord(db *gorm.DB, row types.DriveData, newKey string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var existing types.DriveData
		e := tx.Where("`drive` = ? AND `data_key` = ?", row.Drive, newKey).First(&existing).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if errors.Is(e, gorm.ErrRecordNotFound) && row.Value != "" {
			if e := tx.Create(&types.DriveData{
				Drive: row.Drive,
				Key:   newKey,
				Value: row.Value,
			}).Error; e != nil {
				return e
			}
		}
		return tx.Where("`drive` = ? AND `data_key` = ?", row.Drive, row.Key).
			Delete(&types.DriveData{}).Error
	})
}

// ---------------------------------------------------------------------------
// Reusable helpers
// ---------------------------------------------------------------------------

// schema_migration stores a single integer version. It is created and updated
// with database/sql so it does not depend on GORM models or AutoMigrate.
const schemaMigrationTable = "`schema_migration`"

func ensureSchemaMigrationTable(sqlDB *sql.DB) error {
	if _, e := sqlDB.Exec("CREATE TABLE IF NOT EXISTS " + schemaMigrationTable + " (`version` INTEGER NOT NULL)"); e != nil {
		return e
	}
	var n int
	if e := sqlDB.QueryRow("SELECT COUNT(*) FROM " + schemaMigrationTable).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	_, e := sqlDB.Exec("INSERT INTO " + schemaMigrationTable + " (`version`) VALUES (0)")
	return e
}

func currentMigrationVersion(sqlDB *sql.DB) (int, error) {
	var version int
	e := sqlDB.QueryRow("SELECT `version` FROM " + schemaMigrationTable + " LIMIT 1").Scan(&version)
	if errors.Is(e, sql.ErrNoRows) {
		return 0, nil
	}
	return version, e
}

func setMigrationVersion(sqlDB *sql.DB, version int) error {
	result, e := sqlDB.Exec("UPDATE "+schemaMigrationTable+" SET `version` = ?", version)
	if e != nil {
		return e
	}
	affected, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if affected > 0 {
		return nil
	}
	_, e = sqlDB.Exec("INSERT INTO "+schemaMigrationTable+" (`version`) VALUES (?)", version)
	return e
}

type tableColumn struct {
	name    string
	defVal  sql.NullString
	notNull bool
}

func hasTable(sqlDB *sql.DB, dialect, table string) (bool, error) {
	var name string
	var e error
	switch dialect {
	case "sqlite":
		e = sqlDB.QueryRow("SELECT `name` FROM `sqlite_master` WHERE `type` = 'table' AND `name` = ?", table).Scan(&name)
	case "mysql":
		e = sqlDB.QueryRow(
			"SELECT `TABLE_NAME` FROM `information_schema`.`TABLES` WHERE `TABLE_SCHEMA` = DATABASE() AND `TABLE_NAME` = ?",
			table,
		).Scan(&name)
	default:
		return false, fmt.Errorf("unsupported database dialect %q", dialect)
	}
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	return e == nil, e
}

func listTableColumns(sqlDB *sql.DB, dialect, table string) (map[string]tableColumn, error) {
	columns := make(map[string]tableColumn)
	switch dialect {
	case "sqlite":
		rows, e := sqlDB.Query("PRAGMA table_info(`" + table + "`)")
		if e != nil {
			return nil, e
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var cid, notNull, pk int
			var name, colType string
			var dflt sql.NullString
			if e := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); e != nil {
				return nil, e
			}
			columns[strings.ToLower(name)] = tableColumn{name: name, defVal: dflt, notNull: notNull == 1}
		}
		return columns, rows.Err()
	case "mysql":
		rows, e := sqlDB.Query(
			"SELECT `COLUMN_NAME`, `COLUMN_DEFAULT`, `IS_NULLABLE` FROM `information_schema`.`COLUMNS` "+
				"WHERE `TABLE_SCHEMA` = DATABASE() AND `TABLE_NAME` = ?",
			table,
		)
		if e != nil {
			return nil, e
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var name, nullable string
			var dflt sql.NullString
			if e := rows.Scan(&name, &dflt, &nullable); e != nil {
				return nil, e
			}
			columns[strings.ToLower(name)] = tableColumn{name: name, defVal: dflt, notNull: nullable == "NO"}
		}
		return columns, rows.Err()
	default:
		return nil, fmt.Errorf("unsupported database dialect %q", dialect)
	}
}

func hasAnyColumn(columns map[string]tableColumn, names ...string) bool {
	for _, name := range names {
		if _, ok := columns[name]; ok {
			return true
		}
	}
	return false
}
