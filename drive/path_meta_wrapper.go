package drive

import (
	"context"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/storage"
)

var _ types.IDrive = (*PathMetaWrapper)(nil)

type PathMetaWrapper struct {
	types.IDrive
	pathMeta *storage.PathMetaDAO

	principal types.Principal
}

func NewPathMetaWrapper(root types.IDrive, pathMeta *storage.PathMetaDAO,
	principal types.Principal) *PathMetaWrapper {
	return &PathMetaWrapper{IDrive: root, pathMeta: pathMeta, principal: principal}
}

// checkPassword validates the path password for anonymous callers. The password
// is read from the per-request principal injected from the request header.
// On mismatch it returns an error carrying the protected ancestor path so the
// client can cache the password for the whole subtree.
func (pm *PathMetaWrapper) checkPassword(path string) error {
	if !pm.principal.IsAnonymous() {
		return nil
	}
	// an authenticated caller (e.g. a valid signature/access key) already proves
	// authorization for this path, so the path password is not required
	if pm.principal.AuthType != types.AuthTypeNone {
		return nil
	}
	meta, e := pm.pathMeta.GetMerged(path)
	if e != nil {
		return e
	}
	if meta == nil || meta.Password.V == "" {
		return nil
	}
	if pm.principal.PathPassword != meta.Password.V {
		return err.NewNotAllowedMessageDataError(
			i18n.T("drive.path_meta.incorrect_password"),
			types.M{"passwordRequired": true, "passwordPath": meta.Password.Path},
		)
	}
	return nil
}

// Get injects pathMeta into Entry's props
func (pm *PathMetaWrapper) Get(ctx context.Context, path string) (types.IEntry, error) {
	if e := pm.checkPassword(path); e != nil {
		return nil, e
	}
	entry, e := pm.IDrive.Get(ctx, path)
	if e != nil {
		return nil, e
	}
	meta, e := pm.pathMeta.GetMerged(path)
	if e != nil {
		return nil, e
	}
	if meta == nil {
		return entry, nil
	}
	return driveutil.WrapEntryWithMeta(entry, types.M{"pathMeta": *meta}), nil
}

// List checks if the path has a password set and validates it
func (pm *PathMetaWrapper) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if e := pm.checkPassword(path); e != nil {
		return nil, e
	}
	return pm.IDrive.List(ctx, path)
}
