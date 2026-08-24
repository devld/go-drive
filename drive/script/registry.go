package script

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/common/logging"
	"go-drive/common/types"
)

const scriptDriveTypePrefix = "script/"

// RegisterAllScriptDrives rebuilds the dynamically registered Script Drive
// factories from the installed scripts. The registry replaces the complete
// script/ group atomically, so readers never observe a partially refreshed
// set of factories.
func RegisterAllScriptDrives(ctx context.Context, config common.Config, driveRegistry *driveutil.DriveRegistry) error {
	scripts, e := ListDriveScripts(config)
	if e != nil {
		return e
	}

	factories := make([]driveutil.DriveFactoryConfig, 0, len(scripts))
	for _, script := range scripts {
		staticForm, e := GetDriveScriptConfigForm(ctx, config, script.Name)
		if e != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logging.For("scr-drv").Warnf("cannot register script drive %q: %v", script.Name, e)
			continue
		}

		configForm := make([]types.FormItem, 0, len(staticForm)+1)
		configForm = append(configForm, staticForm...)
		configForm = append(configForm, scriptPoolFormItem())
		factories = append(factories, driveutil.DriveFactoryConfig{
			Type:        scriptDriveTypePrefix + script.Name,
			DisplayName: script.DisplayName,
			README:      script.Description,
			ConfigForm:  configForm,
			Factory:     scriptDriveFactory(script.Name),
		})
	}

	return driveRegistry.ReplaceDriveGroup(scriptDriveTypePrefix, factories)
}

func scriptPoolFormItem() types.FormItem {
	return types.FormItem{
		Field:       poolConfigField,
		Label:       t("form.pool.label"),
		Type:        "text",
		Description: t("form.pool.description"),
	}
}

func scriptDriveFactory(name string) driveutil.DriveFactory {
	return driveutil.DriveFactory{
		Create: func(ctx context.Context, config types.SM, driveUtils driveutil.DriveUtils) (types.IDrive, error) {
			return newScriptDrive(ctx, withScriptName(name, config), driveUtils)
		},
		InitConfig: func(ctx context.Context, config types.SM, driveUtils driveutil.DriveUtils) (*driveutil.DriveInitConfig, error) {
			return initConfig(ctx, withScriptName(name, config), driveUtils)
		},
		Init: func(ctx context.Context, data, config types.SM, driveUtils driveutil.DriveUtils) error {
			return init_(ctx, data, withScriptName(name, config), driveUtils)
		},
	}
}
