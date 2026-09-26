package drive

import (
	"context"
	"go-drive/common"
	"go-drive/common/driveutil"
	"go-drive/drive/fs"
	"go-drive/drive/ftp"
	"go-drive/drive/gdrive"
	"go-drive/drive/mega"
	"go-drive/drive/onedrive"
	"go-drive/drive/s3"
	"go-drive/drive/script"
	"go-drive/drive/sftp"
	"go-drive/drive/webdav"
)

// RegisterAllDrives registers all built-in drive implementations and refreshes
// the dynamically installed Script Drive implementations.
func RegisterAllDrives(ctx context.Context, config common.Config, driveRegistry *driveutil.DriveRegistry) error {
	fs.RegisterDrive(driveRegistry)
	ftp.RegisterDrive(driveRegistry)
	gdrive.RegisterDrive(driveRegistry)
	mega.RegisterDrive(driveRegistry)
	onedrive.RegisterDrive(driveRegistry)
	s3.RegisterDrive(driveRegistry)
	sftp.RegisterDrive(driveRegistry)
	webdav.RegisterDrive(driveRegistry)
	return script.RegisterAllScriptDrives(ctx, config, driveRegistry)
}
