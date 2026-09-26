package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/secretbox"
	"go-drive/common/types"
	"strings"

	"gorm.io/gorm"
)

type DriveDAO struct {
	db      *DB
	secrets *secretbox.Box
}

func NewDriveDAO(db *DB, secrets *secretbox.Box) *DriveDAO {
	return &DriveDAO{db: db, secrets: secrets}
}

func (d *DriveDAO) GetDrives() ([]types.Drive, error) {
	var drivesConfig []types.Drive
	if e := d.db.C().Find(&drivesConfig).Error; e != nil {
		return nil, e
	}
	for i := range drivesConfig {
		opened, e := openDriveConfig(d.secrets, drivesConfig[i].Config)
		if e != nil {
			return nil, fmt.Errorf("drive %s: %w", drivesConfig[i].Name, e)
		}
		drivesConfig[i].Config = opened
	}
	return drivesConfig, nil
}

func (d *DriveDAO) GetDrive(name string) (types.Drive, error) {
	var config types.Drive
	e := d.db.C().Where("`name` = ?", name).Take(&config).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return config, err.NewNotFoundError()
	}
	if e != nil {
		return config, e
	}
	opened, e := openDriveConfig(d.secrets, config.Config)
	if e != nil {
		return config, fmt.Errorf("drive %s: %w", name, e)
	}
	config.Config = opened
	return config, nil
}

func (d *DriveDAO) AddDrive(drive types.Drive, form []types.FormItem) (types.Drive, error) {
	e := d.db.C().Where("`name` = ?", drive.Name).Take(&types.Drive{}).Error
	if e == nil {
		return types.Drive{},
			err.NewNotAllowedMessageError(i18n.T("storage.drives.drive_exists", drive.Name))
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return types.Drive{}, e
	}
	sealed, e := sealDriveConfig(d.secrets, drive.Config, secretConfigFields(form))
	if e != nil {
		return types.Drive{}, e
	}
	drive.Config = sealed
	e = d.db.C().Create(&drive).Error
	return drive, e
}

func (d *DriveDAO) UpdateDrive(name string, drive types.Drive, form []types.FormItem) error {
	sealed, e := sealDriveConfig(d.secrets, drive.Config, secretConfigFields(form))
	if e != nil {
		return e
	}
	drive.Name = name
	drive.Config = sealed
	return d.db.C().Save(drive).Error
}

func (d *DriveDAO) DeleteDrive(name string) error {
	return d.db.C().Delete(&types.Drive{}, "`name` = ?", name).Error
}

func sealDriveConfig(secrets *secretbox.Box, configJSON string, fields []string) (string, error) {
	values, e := parseDriveConfig(configJSON)
	if e != nil {
		return "", e
	}
	changed := false
	for _, field := range fields {
		value := values[field]
		if value == "" {
			continue
		}
		sealed, e := secrets.Encrypt(value)
		if e != nil {
			return "", e
		}
		values[field] = sealed
		changed = true
	}
	if !changed {
		return configJSON, nil
	}
	return encodeDriveConfig(values)
}

func openDriveConfig(secrets *secretbox.Box, configJSON string) (string, error) {
	values, e := parseDriveConfig(configJSON)
	if e != nil {
		return configJSON, nil
	}
	changed := false
	for key, value := range values {
		opened, e := secrets.Decrypt(value)
		if e != nil {
			return "", fmt.Errorf("decrypt drive config %s: %w", key, e)
		}
		if opened == value {
			continue
		}
		values[key] = opened
		changed = true
	}
	if !changed {
		return configJSON, nil
	}
	return encodeDriveConfig(values)
}

func parseDriveConfig(configJSON string) (types.SM, error) {
	if strings.TrimSpace(configJSON) == "" {
		return types.SM{}, nil
	}
	values := types.SM{}
	if e := json.Unmarshal([]byte(configJSON), &values); e != nil {
		return nil, fmt.Errorf("invalid drive config: %w", e)
	}
	if values == nil {
		values = types.SM{}
	}
	return values, nil
}

func encodeDriveConfig(values types.SM) (string, error) {
	encoded, e := json.Marshal(values)
	if e != nil {
		return "", e
	}
	return string(encoded), nil
}

func secretConfigFields(form []types.FormItem) []string {
	fields := make([]string, 0)
	for _, f := range form {
		if f.Type == "password" || f.Secret != "" {
			fields = append(fields, f.Field)
		}
	}
	return fields
}
