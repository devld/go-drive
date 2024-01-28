package server

import (
	"context"
	"encoding/json"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/drive/script"
	"go-drive/storage"
	"os"
	"path/filepath"
	"regexp"
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
	ds := drive_util.GetRegisteredDrives(dr.config)
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
		f := drive_util.GetDrive(d.Type, dr.config)
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
	if e := checkDriveName(d.Name); e != nil {
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
	if e := checkDriveName(name); e != nil {
		_ = c.Error(e)
		return
	}
	d := types.Drive{}
	if e := c.Bind(&d); e != nil {
		_ = c.Error(e)
		return
	}
	f := drive_util.GetDrive(d.Type, dr.config)
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
	SetResult(c, data)
}

func (dr *drivesRoute) doDriveInit(c *gin.Context) {
	name := c.Param("name")
	data := make(types.SM, 0)
	if e := c.Bind(&data); e != nil {
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
	config   common.Config
	repoLock sync.Mutex
}

func (sdr *scriptDrivesRoute) _loadAvailableDriveScripts(ctx context.Context, forceLoad bool) ([]script.AvailableDriveScript, error) {
	sdr.repoLock.Lock()
	defer sdr.repoLock.Unlock()

	cacheFile := filepath.Join(sdr.config.TempDir, "drives-repository-cache.json")

	var result []script.AvailableDriveScript = nil

	if !forceLoad {
		if data, e := os.ReadFile(cacheFile); e == nil {
			temp := make([]script.AvailableDriveScript, 0)
			if e := json.Unmarshal(data, &temp); e == nil {
				result = temp
			}
		}
	}

	if result == nil {
		scripts, e := script.ListAvailableScriptsFromRepository(ctx, sdr.config.DriveRepositoryURL)
		if e != nil {
			return result, e
		}
		result = scripts

		data, e := json.Marshal(scripts)
		if e != nil {
			return result, e
		}
		if e := os.WriteFile(cacheFile, data, 0644); e != nil {
			return result, e
		}
	}
	return result, nil
}

func (sdr *scriptDrivesRoute) getAvailableDrives(c *gin.Context) {
	result, e := sdr._loadAvailableDriveScripts(c.Request.Context(), utils.ToBool(c.Query("force")))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, result)
}

func (sdr *scriptDrivesRoute) getInstalledDrives(c *gin.Context) {
	scripts, e := script.ListDriveScripts(sdr.config)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, scripts)
}

func (sdr *scriptDrivesRoute) installDrive(c *gin.Context) {
	name := c.Param("name")
	scripts, e := sdr._loadAvailableDriveScripts(c.Request.Context(), false)
	if e != nil {
		_ = c.Error(e)
		return
	}

	ads, ok := utils.ArrayFind(scripts, func(item script.AvailableDriveScript, _ int) bool { return item.Name == name })
	if !ok {
		_ = c.Error(err.NewNotFoundError())
		return
	}

	if e := script.InstallDriveScript(c.Request.Context(), sdr.config, ads); e != nil {
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

var driveNamePattern = regexp.MustCompile("^[^/\\\x00:*\"<>|]+$")

func checkDriveName(name string) error {
	if name == "" || name == "." || name == ".." || !driveNamePattern.MatchString(name) {
		return err.NewBadRequestError(i18n.T("api.admin.invalid_drive_name", name))
	}
	return nil
}

const escapedPassword = "YOU CAN'T SEE ME"

func escapeDriveConfigSecrets(form []types.FormItem, config string) string {
	val := types.SM{}
	_ = json.Unmarshal([]byte(config), &val)
	for _, f := range form {
		if (f.Type == "password" || f.Secret != "") && val[f.Field] != "" {
			val[f.Field] = escapedPassword
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
			(val[f.Field] == escapedPassword || (f.Secret != "" && val[f.Field] == f.Secret)) {
			val[f.Field] = savedVal[f.Field]
		}
	}
	s, _ := json.Marshal(val)
	return string(s)
}
