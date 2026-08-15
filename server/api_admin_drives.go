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

	"github.com/gin-gonic/gin"
)

type drivesRoute struct {
	config       common.Config
	driveDAO     *storage.DriveDAO
	driveDataDAO *storage.DriveDataDAO
	rootDrive    *drive.RootDrive
}

func (dr *drivesRoute) getDriveFactories(c *gin.Context) {
	ds := driveutil.GetRegisteredDrives(dr.config)
	sort.Slice(ds, func(i, j int) bool { return ds[i].Type < ds[j].Type })
	SetResult(c, ds)
}

func (dr *drivesRoute) getDrives(c *gin.Context) {
	drives, e := dr.driveDAO.GetDrives()
	if e != nil {
		_ = c.Error(e)
		return
	}
	for i, d := range drives {
		f := driveutil.GetDrive(d.Type, dr.config)
		if f == nil {
			continue
		}
		drives[i].Config = escapeDriveConfigSecrets(f.ConfigForm, d.Config)
	}
	SetResult(c, drives)
}

func (dr *drivesRoute) createDrive(c *gin.Context) {
	d := types.Drive{}
	if e := c.Bind(&d); e != nil {
		_ = c.Error(e)
		return
	}
	if e := CheckPathSegment(d.Name, "api.admin.invalid_drive_name"); e != nil {
		_ = c.Error(e)
		return
	}
	d, e := dr.driveDAO.AddDrive(d)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, d)
}

func (dr *drivesRoute) updateDrive(c *gin.Context) {
	name := c.Param("name")
	d := types.Drive{}
	if e := c.Bind(&d); e != nil {
		_ = c.Error(e)
		return
	}
	f := driveutil.GetDrive(d.Type, dr.config)
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
	e = dr.driveDAO.UpdateDrive(name, d)
	if e != nil {
		_ = c.Error(e)
		return
	}
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
}

func (dr *drivesRoute) getDriveInitConfig(c *gin.Context) {
	name := c.Param("name")
	data, e := dr.rootDrive.DriveInitConfig(c.Request.Context(), name)
	if e != nil {
		_ = c.Error(e)
		return
	}
	escapeDriveInitConfigSecrets(data)
	SetResult(c, data)
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
}

func (dr *drivesRoute) reloadDrives(c *gin.Context) {
	if e := dr.rootDrive.ReloadDrive(c.Request.Context(), false); e != nil {
		_ = c.Error(e)
	}
}

type scriptDrivesRoute struct {
	config     common.Config
	runner     task.Runner
	repoLock   sync.Mutex
	syncTaskID string
}

func (sdr *scriptDrivesRoute) listDriveScripts(c *gin.Context) {
	result, e := script.ListAllDriveScripts(sdr.config)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, result)
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
			SetResult(c, existing)
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
	sdr.syncTaskID = created.Id
	SetResult(c, created)
}

func (sdr *scriptDrivesRoute) installDrive(c *gin.Context) {
	if e := script.InstallDriveScript(sdr.config, c.Param("name")); e != nil {
		_ = c.Error(e)
		return
	}
}

func (sdr *scriptDrivesRoute) uninstallDrive(c *gin.Context) {
	name := c.Param("name")
	if e := script.UninstallDriveScript(sdr.config, name); e != nil {
		_ = c.Error(e)
		return
	}
}

func (sdr *scriptDrivesRoute) getDriveScriptContent(c *gin.Context) {
	content, e := script.GetDriveScript(sdr.config, c.Param("name"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, content)
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

	saved, e := store.Load(keys...)
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
