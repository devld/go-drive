package driveutil

import (
	"context"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	"slices"
)

// PrepareWrite preserves wrapper-specific authorization and path translation.
// Plain storage drives use the same read-only inspection without user policy.
func PrepareWrite(ctx context.Context, drive types.IDrive, path string, mode types.WriteMode) (*types.WritePlan, error) {
	if p, ok := drive.(types.WritePreparer); ok {
		return p.PrepareWrite(ctx, path, mode)
	}
	return InspectWrite(ctx, drive, path, mode)
}

// InspectWrite selects a destination and lists missing parents without writing.
// Callers must execute the returned path exactly; a later conflict is an error.
func InspectWrite(ctx context.Context, drive types.IDrive, path string, mode types.WriteMode) (*types.WritePlan, error) {
	path = utils.CleanPath(path)
	if utils.IsRootPath(path) {
		return nil, err.NewNotAllowedError()
	}
	if mode == types.WriteRename {
		var e error
		path, e = FindNonExistsEntryName(ctx, drive, path)
		if e != nil {
			return nil, e
		}
	} else if mode != types.WriteReject && mode != types.WriteOverwrite {
		return nil, err.NewNotAllowedError()
	}
	if mode != types.WriteOverwrite {
		if _, e := RequireFileNotExists(ctx, drive, path); e != nil {
			return nil, e
		}
	}
	plan := &types.WritePlan{Path: path, Override: mode == types.WriteOverwrite}
	for parent := utils.PathParent(path); !utils.IsRootPath(parent); parent = utils.PathParent(parent) {
		entry, e := drive.Get(ctx, parent)
		if e == nil {
			if !entry.Type().IsDir() {
				return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
			}
			break
		}
		if !err.IsNotFoundError(e) {
			return nil, e
		}
		plan.Parents = append(plan.Parents, parent)
	}
	slices.Reverse(plan.Parents)
	return plan, nil
}

func WriteModeForOverride(override bool) types.WriteMode {
	if override {
		return types.WriteOverwrite
	}
	return types.WriteReject
}
