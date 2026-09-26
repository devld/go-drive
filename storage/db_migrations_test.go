package storage

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"go-drive/common/driveutil"
	"go-drive/common/secretbox"
	"go-drive/common/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newIsolatedGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migrate.db")), &gorm.Config{
		Logger: logger.Discard,
	})
	if e != nil {
		t.Fatalf("open sqlite: %v", e)
	}
	t.Cleanup(func() { closeDB(db) })
	if e := db.AutoMigrate(&types.Drive{}, &types.DriveData{}, &types.Job{}); e != nil {
		t.Fatalf("AutoMigrate: %v", e)
	}
	return db
}

func TestRunDBMigrationsAppliesPendingVersionsOnce(t *testing.T) {
	db := newIsolatedGormDB(t)
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}

	applied := make([]int, 0, 2)
	migrations := []dbMigration{
		{version: 1, name: "first", run: func(*gorm.DB) error {
			applied = append(applied, 1)
			return nil
		}},
		{version: 2, name: "second", run: func(*gorm.DB) error {
			applied = append(applied, 2)
			return nil
		}},
	}
	if e := runDBMigrations(db, migrations); e != nil {
		t.Fatal(e)
	}
	version, e := currentMigrationVersion(sqlDB)
	if e != nil {
		t.Fatal(e)
	}
	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}
	if got, want := applied, []int{1, 2}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("applied = %#v, want %#v", applied, want)
	}

	if e := runDBMigrations(db, migrations); e != nil {
		t.Fatal(e)
	}
	if len(applied) != 2 {
		t.Fatalf("migrations ran again: %#v", applied)
	}
}

func TestRunDBMigrationsDoesNotAdvanceVersionOnFailure(t *testing.T) {
	db := newIsolatedGormDB(t)
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}

	wantErr := errors.New("boom")
	migrations := []dbMigration{
		{version: 1, name: "ok", run: func(*gorm.DB) error { return nil }},
		{version: 2, name: "fail", run: func(*gorm.DB) error { return wantErr }},
	}
	if e := runDBMigrations(db, migrations); !errors.Is(e, wantErr) {
		t.Fatalf("error = %v, want %v", e, wantErr)
	}
	version, e := currentMigrationVersion(sqlDB)
	if e != nil {
		t.Fatal(e)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1 after failed later migration", version)
	}
}

func TestRunDBMigrationsSkipsAlreadyAppliedOAuthRename(t *testing.T) {
	db := newIsolatedGormDB(t)
	if e := db.Create(&types.DriveData{Drive: "custom", Key: "token", Value: "pat"}).Error; e != nil {
		t.Fatal(e)
	}

	oauth := []dbMigration{{version: 1, name: "oauth_drive_data", run: migrateOAuthDriveData}}
	if e := runDBMigrations(db, oauth); e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&types.DriveData{Drive: "custom", Key: "token", Value: "later-pat"}).Error; e != nil {
		t.Fatal(e)
	}
	if e := runDBMigrations(db, oauth); e != nil {
		t.Fatal(e)
	}

	got := loadDriveData(t, db, "custom")
	if got["token"] != "later-pat" {
		t.Fatalf("later token was rewritten: %#v", got)
	}
	if got[driveutil.DsKeyToken] != "pat" {
		t.Fatalf("original oauth migration result lost: %#v", got)
	}
}

func TestMigrateOAuthDriveDataRenamesLegacyKeys(t *testing.T) {
	db := newIsolatedGormDB(t)
	if e := db.Create(&types.DriveData{
		Drive: "gdrive", Key: "token", Value: "access",
	}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&types.DriveData{
		Drive: "gdrive", Key: "refresh_token", Value: "refresh",
	}).Error; e != nil {
		t.Fatal(e)
	}

	if e := migrateOAuthDriveData(db); e != nil {
		t.Fatal(e)
	}

	got := loadDriveData(t, db, "gdrive")
	if _, ok := got["token"]; ok {
		t.Fatalf("legacy token key still present: %#v", got)
	}
	if got[driveutil.DsKeyToken] != "access" || got[driveutil.DsKeyRefreshToken] != "refresh" {
		t.Fatalf("oauth keys = %#v", got)
	}
}

func TestMigrateScriptDriveConfigsGitHub(t *testing.T) {
	db := newIsolatedGormDB(t)
	if e := db.Create(&types.Drive{
		Name:    "gh",
		Enabled: true,
		Type:    "script",
		Config:  mustJSON(t, types.SM{"script": "github", "pool": "4,2,1,1m"}),
	}).Error; e != nil {
		t.Fatal(e)
	}
	data := types.SM{
		"_script":   "github.js",
		"owner":     "octo",
		"repo":      "go-drive",
		"token":     "ghp_old",
		"cache_ttl": "5m",
	}
	for key, value := range data {
		if e := db.Create(&types.DriveData{Drive: "gh", Key: key, Value: value}).Error; e != nil {
			t.Fatal(e)
		}
	}

	if e := migrateScriptDriveConfigs(db); e != nil {
		t.Fatal(e)
	}
	if e := migrateOAuthDriveData(db); e != nil {
		t.Fatal(e)
	}

	var drive types.Drive
	if e := db.Where("`name` = ?", "gh").First(&drive).Error; e != nil {
		t.Fatal(e)
	}
	if drive.Type != "script/github" {
		t.Fatalf("type = %q", drive.Type)
	}
	config := types.SM{}
	if e := json.Unmarshal([]byte(drive.Config), &config); e != nil {
		t.Fatal(e)
	}
	if config["owner"] != "octo" || config["repo"] != "go-drive" || config["token"] != "ghp_old" {
		t.Fatalf("config = %#v", config)
	}
	if config["__pool"] != "4,2,1,1m" {
		t.Fatalf("pool = %#v", config)
	}
	if _, ok := config["script"]; ok {
		t.Fatalf("legacy script field kept: %#v", config)
	}
	got := loadDriveData(t, db, "gh")
	if _, ok := got["token"]; ok {
		t.Fatalf("github token left in drive data: %#v", got)
	}
	if _, ok := got[driveutil.DsKeyToken]; ok {
		t.Fatalf("github token moved into oauth namespace: %#v", got)
	}
	if _, ok := got["_script"]; ok {
		t.Fatalf("_script leftover: %#v", got)
	}
}

func TestMigrateScriptDriveConfigsUnknownScriptLeavesMarkers(t *testing.T) {
	db := newIsolatedGormDB(t)
	if e := db.Create(&types.Drive{
		Name:    "custom",
		Enabled: true,
		Type:    "script",
		Config:  mustJSON(t, types.SM{"script": "custom"}),
	}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&types.DriveData{Drive: "custom", Key: "token", Value: "pat"}).Error; e != nil {
		t.Fatal(e)
	}

	if e := migrateScriptDriveConfigs(db); e != nil {
		t.Fatal(e)
	}

	var drive types.Drive
	if e := db.Where("`name` = ?", "custom").First(&drive).Error; e != nil {
		t.Fatal(e)
	}
	if drive.Type != "script" {
		t.Fatalf("unknown script type = %q", drive.Type)
	}
	got := loadDriveData(t, db, "custom")
	if got["token"] != "pat" {
		t.Fatalf("unknown script data rewritten before oauth migration: %#v", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func loadDriveData(t *testing.T, db *gorm.DB, name string) types.SM {
	t.Helper()
	var rows []types.DriveData
	if e := db.Where("`drive` = ?", name).Find(&rows).Error; e != nil {
		t.Fatal(e)
	}
	got := make(types.SM, len(rows))
	for _, row := range rows {
		got[row.Key] = row.Value
	}
	return got
}

func TestMigrateLegacyJobSchemaCopiesAndDropsOldColumns(t *testing.T) {
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-jobs.db")), &gorm.Config{
		Logger: logger.Discard,
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { closeDB(db) })
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	if _, e := sqlDB.Exec(`
CREATE TABLE jobs (
	id integer PRIMARY KEY AUTOINCREMENT,
	description text NOT NULL,
	enabled numeric NOT NULL,
	schedule varchar(64),
	job varchar(64) NOT NULL,
	params text
)`); e != nil {
		t.Fatal(e)
	}
	if _, e := sqlDB.Exec(
		`INSERT INTO jobs (description, enabled, schedule, job, params) VALUES (?, ?, ?, ?, ?)`,
		"nightly", 1, "0 0 * * *", "script", `{"code":"1"}`,
	); e != nil {
		t.Fatal(e)
	}

	if e := migrateLegacyJobSchema(db); e != nil {
		t.Fatal(e)
	}

	columns, e := listTableColumns(sqlDB, "sqlite", "jobs")
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range legacyJobColumns {
		if _, ok := columns[name]; ok {
			t.Fatalf("legacy column %q still exists", name)
		}
	}
	for _, name := range jobTargetColumns {
		info, ok := columns[name]
		if !ok {
			t.Fatalf("missing column %q", name)
		}
		if isNeedMigrationDefault(info.defVal) {
			t.Fatalf("column %q still defaults to %q", name, info.defVal.String)
		}
	}

	var job types.Job
	if e := db.First(&job).Error; e != nil {
		t.Fatal(e)
	}
	if job.Action != "script" || job.ActionParams != `{"code":"1"}` {
		t.Fatalf("job action fields = %#v", job)
	}
	if !strings.Contains(job.Triggers, "0 0 * * *") {
		t.Fatalf("triggers = %q", job.Triggers)
	}

	if e := db.Create(&types.Job{
		Description:  "new",
		Triggers:     "[]",
		Action:       "script",
		ActionParams: "{}",
		Enabled:      true,
	}).Error; e != nil {
		t.Fatal(e)
	}
}

func TestMigrateLegacyJobSchemaIsNoopForCurrentSchema(t *testing.T) {
	db := newIsolatedGormDB(t)
	if e := db.Create(&types.Job{
		Description:  "current",
		Triggers:     "[]",
		Action:       "script",
		ActionParams: "{}",
		Enabled:      true,
	}).Error; e != nil {
		t.Fatal(e)
	}
	if e := migrateLegacyJobSchema(db); e != nil {
		t.Fatal(e)
	}
	var job types.Job
	if e := db.First(&job).Error; e != nil {
		t.Fatal(e)
	}
	if job.Triggers != "[]" || job.Action != "script" {
		t.Fatalf("current job rewritten: %#v", job)
	}
}

func TestRunDBMigrationsAlwaysRunsEveryStartup(t *testing.T) {
	db := newIsolatedGormDB(t)
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}

	alwaysRuns := 0
	migrations := []dbMigration{
		{version: -1, name: "always", run: func(*gorm.DB) error {
			alwaysRuns++
			return nil
		}},
		{version: 1, name: "once", run: func(*gorm.DB) error { return nil }},
	}
	if e := runDBMigrations(db, migrations); e != nil {
		t.Fatal(e)
	}
	if e := runDBMigrations(db, migrations); e != nil {
		t.Fatal(e)
	}
	if alwaysRuns != 2 {
		t.Fatalf("always-run count = %d, want 2", alwaysRuns)
	}
	version, e := currentMigrationVersion(sqlDB)
	if e != nil {
		t.Fatal(e)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

func TestMigrateEncryptDriveSecretsSealsExistingFields(t *testing.T) {
	db := newIsolatedGormDB(t)
	secrets, e := secretbox.New("migration-test")
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&types.Drive{
		Name: "files", Type: "s3", Enabled: true,
		Config: `{"id":"ak","secret":"sk","bucket":"b"}`,
	}).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&types.Drive{
		Name: "local", Type: "fs", Enabled: true, Config: `{"path":"photos"}`,
	}).Error; e != nil {
		t.Fatal(e)
	}
	rows := []types.DriveData{
		{Drive: "cloud", Key: driveutil.DsKeyToken, Value: "access"},
		{Drive: "cloud", Key: driveutil.DsKeyRefreshToken, Value: "refresh"},
		{Drive: "cloud", Key: "drive_id", Value: "root"},
	}
	if e = db.Create(&rows).Error; e != nil {
		t.Fatal(e)
	}

	if e = migrateEncryptDriveSecrets(db, secrets); e != nil {
		t.Fatal(e)
	}
	if e = migrateEncryptDriveSecrets(db, secrets); e != nil {
		t.Fatal(e)
	}

	var s3 types.Drive
	if e = db.Where("`name` = ?", "files").First(&s3).Error; e != nil {
		t.Fatal(e)
	}
	config := types.SM{}
	if e = json.Unmarshal([]byte(s3.Config), &config); e != nil {
		t.Fatal(e)
	}
	opened, e := secrets.Decrypt(config["secret"])
	if e != nil {
		t.Fatal(e)
	}
	config["secret"] = opened
	if config["id"] != "ak" || config["secret"] != "sk" || config["bucket"] != "b" {
		t.Fatalf("config = %#v", config)
	}
	if !secretbox.IsSealed(mustConfigField(t, s3.Config, "secret")) {
		t.Fatal("secret was not sealed")
	}
	if mustConfigField(t, s3.Config, "id") != "ak" {
		t.Fatal("id was sealed")
	}

	var local types.Drive
	if e = db.Where("`name` = ?", "local").First(&local).Error; e != nil {
		t.Fatal(e)
	}
	if local.Config != `{"path":"photos"}` {
		t.Fatalf("fs config changed: %s", local.Config)
	}

	var stored []types.DriveData
	if e = db.Where("`drive` = ?", "cloud").Find(&stored).Error; e != nil {
		t.Fatal(e)
	}
	got := types.SM{}
	for _, row := range stored {
		got[row.Key] = row.Value
	}
	if !secretbox.IsSealed(got[driveutil.DsKeyToken]) || !secretbox.IsSealed(got[driveutil.DsKeyRefreshToken]) {
		t.Fatalf("oauth values = %#v", got)
	}
	if got["drive_id"] != "root" {
		t.Fatalf("drive_id = %q", got["drive_id"])
	}
	token, e := secrets.Decrypt(got[driveutil.DsKeyToken])
	if e != nil {
		t.Fatal(e)
	}
	if token != "access" {
		t.Fatalf("token = %q", token)
	}
}

func mustConfigField(t *testing.T, raw, field string) string {
	t.Helper()
	values := types.SM{}
	if e := json.Unmarshal([]byte(raw), &values); e != nil {
		t.Fatal(e)
	}
	return values[field]
}
