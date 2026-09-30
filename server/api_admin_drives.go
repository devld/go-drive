package server

import (
	"encoding/json"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/drive"
	"go-drive/drive/script"
	"go-drive/storage"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type drivesRoute struct {
	driveRegistry *driveutil.DriveRegistry
	driveDAO      *storage.DriveDAO
	driveDataDAO  *storage.DriveDataDAO
	rootDrive     *drive.RootDrive
	runner        task.Runner
	reloadSubmit  sync.Mutex
	reloadState   driveReloadState
}

const driveReloadTaskGroup = "admin/drive-reload"

type driveReloadState struct {
	version         atomic.Uint64 // latest saved drive configuration version
	reloadedVersion atomic.Uint64 // latest version applied by a successful reload
}

func (s *driveReloadState) markDirty() {
	s.version.Add(1)
}

func (s *driveReloadState) needsReload() bool {
	return s.version.Load() != s.reloadedVersion.Load()
}

func (s *driveReloadState) markReloaded(version uint64) {
	for current := s.reloadedVersion.Load(); version > current; current = s.reloadedVersion.Load() {
		if s.reloadedVersion.CompareAndSwap(current, version) {
			return
		}
	}
}

func (dr *drivesRoute) getDriveFactories(c *gin.Context) {
	setResult(c, dr.driveRegistry.GetRegisteredDrives())
}

func (dr *drivesRoute) getDrives(c *gin.Context) {
	drives, e := dr.driveDAO.GetDrives()
	if e != nil {
		_ = c.Error(e)
		return
	}
	for i, d := range drives {
		f := dr.driveRegistry.GetDrive(d.Type)
		if f == nil {
			continue
		}
		drives[i].Config = escapeDriveConfigSecrets(f.ConfigForm, d.Config)
	}
	setResult(c, drives)
}

func (dr *drivesRoute) getDriveReloadStatus(c *gin.Context) {
	activeTask, e := dr.getActiveDriveReloadTask()
	if e != nil {
		_ = c.Error(e)
		return
	}
	result := types.M{
		"needsReload": dr.reloadState.needsReload(),
		"reloading":   activeTask != nil,
	}
	if activeTask != nil {
		result["task"] = activeTask
	}
	setResult(c, result)
}

// getActiveDriveReloadTask returns the pending or running reload task, if any.
func (dr *drivesRoute) getActiveDriveReloadTask() (*task.Task, error) {
	tasks, e := dr.runner.GetTasks(driveReloadTaskGroup)
	if e != nil {
		return nil, e
	}
	for i := range tasks {
		if !tasks[i].Finished() {
			return &tasks[i], nil
		}
	}
	return nil, nil
}

func (dr *drivesRoute) createDrive(c *gin.Context) {
	d := types.Drive{}
	if e := c.Bind(&d); e != nil {
		_ = c.Error(e)
		return
	}
	if e := checkPathSegment(d.Name, "api.admin.invalid_drive_name"); e != nil {
		_ = c.Error(e)
		return
	}
	f := dr.driveRegistry.GetDrive(d.Type)
	var form []types.FormItem
	if f != nil {
		form = f.ConfigForm
	}
	saved, e := dr.driveDAO.AddDrive(d, form)
	if e != nil {
		_ = c.Error(e)
		return
	}
	dr.reloadState.markDirty()
	if f != nil {
		saved.Config = escapeDriveConfigSecrets(f.ConfigForm, saved.Config)
	}
	setResult(c, saved)
}

func (dr *drivesRoute) updateDrive(c *gin.Context) {
	name := c.Param("name")
	d := types.Drive{}
	if e := c.Bind(&d); e != nil {
		_ = c.Error(e)
		return
	}
	f := dr.driveRegistry.GetDrive(d.Type)
	if f == nil {
		_ = c.Error(err.NewNotAllowedMessageError(i18n.T("api.admin.unknown_drive_type", d.Type)))
		return
	}
	savedDrive, e := dr.driveDAO.GetDrive(name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	d.Config = unescapeDriveConfigSecrets(f.ConfigForm, savedDrive.Config, d.Config)
	e = dr.driveDAO.UpdateDrive(name, d, f.ConfigForm)
	if e != nil {
		_ = c.Error(e)
		return
	}
	dr.reloadState.markDirty()
	_ = dr.rootDrive.ClearDriveCache(name)
}

func (dr *drivesRoute) deleteDrive(c *gin.Context) {
	name := c.Param("name")
	e := dr.driveDAO.DeleteDrive(name)
	_ = dr.rootDrive.ClearDriveCache(name)
	_ = dr.driveDataDAO.Remove(name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	dr.reloadState.markDirty()
}

func (dr *drivesRoute) getDriveInitConfig(c *gin.Context) {
	name := c.Param("name")
	data, e := dr.rootDrive.DriveInitConfig(c.Request.Context(), name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	escapeDriveInitConfigSecrets(data)
	setResult(c, data)
}

func (dr *drivesRoute) doDriveInit(c *gin.Context) {
	name := c.Param("name")
	data := make(types.SM, 0)
	if e := c.Bind(&data); e != nil {
		_ = c.Error(e)
		return
	}
	if e := restoreDriveInitSecrets(data, dr.driveDataDAO.GetDataStore(name)); e != nil {
		_ = c.Error(e)
		return
	}
	if e := dr.rootDrive.DriveInit(c.Request.Context(), name, data); e != nil {
		_ = c.Error(e)
		return
	}
	dr.reloadState.markDirty()
}

func (dr *drivesRoute) reloadDrives(c *gin.Context) {
	dr.reloadSubmit.Lock()
	defer dr.reloadSubmit.Unlock()

	activeTask, e := dr.getActiveDriveReloadTask()
	if e != nil {
		_ = c.Error(e)
		return
	}
	if activeTask != nil {
		setResult(c, *activeTask)
		return
	}

	reloadTask, e := dr.runner.ExecuteAndWait(c.Request.Context(), func(ctx types.TaskCtx) (any, error) {
		reloadVersion := dr.reloadState.version.Load()
		if e := dr.rootDrive.ReloadDrive(ctx, false); e != nil {
			return nil, e
		}
		dr.reloadState.markReloaded(reloadVersion)
		return nil, nil
	}, 2*time.Second, task.WithNameGroup("Reload drives", driveReloadTaskGroup))
	if e != nil {
		_ = c.Error(e)
		return
	}
	setResult(c, reloadTask)
}

type scriptDrivesRoute struct {
	config        common.Config
	driveRegistry *driveutil.DriveRegistry
	runner        task.Runner
	repoLock      sync.Mutex
	syncTaskID    string
}

func (sdr *scriptDrivesRoute) listDriveScripts(c *gin.Context) {
	result, e := script.ListAllDriveScripts(sdr.config)
	if e != nil {
		_ = c.Error(e)
		return
	}
	setResult(c, result)
}

func (sdr *scriptDrivesRoute) syncAvailableDrives(c *gin.Context) {
	if sdr.runner == nil {
		_ = c.Error(err.NewNotAllowedError())
		return
	}

	sdr.repoLock.Lock()
	defer sdr.repoLock.Unlock()

	if sdr.syncTaskID != "" {
		existing, e := sdr.runner.GetTask(sdr.syncTaskID)
		if e == nil && !existing.Finished() {
			setResult(c, existing)
			return
		}
	}

	created, e := sdr.runner.Execute(func(ctx types.TaskCtx) (any, error) {
		return script.SyncDriveScriptsFromRepository(ctx, sdr.config, sdr.config.DriveRepositoryURL)
	}, task.WithNameGroup(sdr.config.DriveRepositoryURL, "admin/drive-scripts"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	sdr.syncTaskID = created.ID
	setResult(c, created)
}

func (sdr *scriptDrivesRoute) installDrive(c *gin.Context) {
	if e := script.InstallDriveScript(sdr.config, c.Param("name")); e != nil {
		_ = c.Error(e)
		return
	}
	if e := script.RegisterAllScriptDrives(c.Request.Context(), sdr.config, sdr.driveRegistry); e != nil {
		_ = c.Error(e)
	}
}

func (sdr *scriptDrivesRoute) uninstallDrive(c *gin.Context) {
	name := c.Param("name")
	if e := script.UninstallDriveScript(sdr.config, name); e != nil {
		_ = c.Error(e)
		return
	}
	if e := script.RegisterAllScriptDrives(c.Request.Context(), sdr.config, sdr.driveRegistry); e != nil {
		_ = c.Error(e)
	}
}

func (sdr *scriptDrivesRoute) getDriveScriptContent(c *gin.Context) {
	content, e := script.GetDriveScript(sdr.config, c.Param("name"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	setResult(c, content)
}

func (sdr *scriptDrivesRoute) saveDriveScriptContent(c *gin.Context) {
	content := script.DriveScriptContent{}
	if e := c.Bind(&content); e != nil {
		_ = c.Error(e)
		return
	}
	if e := script.SaveDriveScript(sdr.config, c.Param("name"), content); e != nil {
		_ = c.Error(e)
		return
	}
	if e := script.RegisterAllScriptDrives(c.Request.Context(), sdr.config, sdr.driveRegistry); e != nil {
		_ = c.Error(e)
	}
}

const (
	secretPlaceholder = "__go-drive_secret__"
)

func isSecretPlaceholder(value string) bool {
	return value == secretPlaceholder
}

// escapeDriveInitConfigSecrets replaces persisted initialization values before
// the configuration is returned to the browser. Initialization forms use the
// shared marker even when a form item has a custom Secret value, because the
// marker can be restored on submission without evaluating InitConfig again.
func escapeDriveInitConfigSecrets(config *driveutil.DriveInitConfig) {
	if config == nil || config.Value == nil {
		return
	}
	for _, f := range config.Form {
		if (f.Type == "password" || f.Secret != "") && config.Value[f.Field] != "" {
			config.Value[f.Field] = secretPlaceholder
		}
	}
}

// restoreDriveInitSecrets restores unchanged initialization values from the
// drive's private data store. The submitted marker is intentionally handled
// without the form: InitConfig may have side effects such as rotating OAuth
// state, so it must not be called a second time just to recover field metadata.
func restoreDriveInitSecrets(data types.SM, store driveutil.DriveDataStore) error {
	keys := make([]string, 0)
	for key, value := range data {
		if isSecretPlaceholder(value) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)

	saved, e := store.Load(keys[0], keys[1:]...)
	if e != nil {
		return e
	}
	for _, key := range keys {
		data[key] = saved[key]
	}
	return nil
}

func escapeDriveConfigSecrets(form []types.FormItem, config string) string {
	val := types.SM{}
	_ = json.Unmarshal([]byte(config), &val)
	for _, f := range form {
		if (f.Type == "password" || f.Secret != "") && val[f.Field] != "" {
			val[f.Field] = secretPlaceholder
			if f.Secret != "" {
				val[f.Field] = f.Secret
			}
		}
	}
	s, _ := json.Marshal(val)
	return string(s)
}

func unescapeDriveConfigSecrets(form []types.FormItem, savedConfig string, config string) string {
	savedVal := types.SM{}
	val := types.SM{}
	_ = json.Unmarshal([]byte(savedConfig), &savedVal)
	_ = json.Unmarshal([]byte(config), &val)
	for _, f := range form {
		if (f.Type == "password" || f.Secret != "") &&
			(isSecretPlaceholder(val[f.Field]) || (f.Secret != "" && val[f.Field] == f.Secret)) {
			val[f.Field] = savedVal[f.Field]
		}
	}
	s, _ := json.Marshal(val)
	return string(s)
}
