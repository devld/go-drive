package archive

import (
	"context"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	pathpkg "path"
)

// MemberExtractor writes selected archive members into a destination drive.
type MemberExtractor interface {
	ExtractMembers(ctx types.TaskCtx, d types.IDrive, source types.IEntry, dest string, members []string, override bool) error
}

// ExtractMembers writes selected archive members into dest, preserving archive
// relative paths. dest and every Save/MakeDir go through drive, so chroot and
// permission wrappers on that drive still apply. members is a set of archive
// paths; a directory path includes all descendants. The caller does not need to
// merge selections.
func (s *Previewer) ExtractMembers(ctx types.TaskCtx, d types.IDrive, source types.IEntry, dest string, members []string, override bool) error {
	dest = utils.CleanPath(dest)
	selected, err := selectedMemberSet(members)
	if err != nil {
		return err
	}

	srcCtx := task.NewTaskCtxWrapper(ctx, false, false)
	opened, format, err := s.openSourceAndFormat(srcCtx, source)
	if err != nil {
		return err
	}
	defer func() { _ = opened.Close() }()

	archiveFS := s.newArchiveFS(ctx, opened, format)
	items, err := s.collectExtractItems(archiveFS, selected)
	if err != nil {
		return err
	}

	var total int64
	for _, item := range items {
		if !item.isDir && item.size > 0 {
			total += item.size
		}
	}
	if total > 0 {
		ctx.Total(total, true)
	}
	saveCtx := task.NewTaskCtxWrapper(ctx, true, false)

	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		target, err := joinExtractPath(dest, item.path)
		if err != nil {
			return err
		}
		if item.isDir {
			if err := ensureDirs(ctx, d, target); err != nil {
				return err
			}
			continue
		}
		if err := ensureDirs(ctx, d, utils.PathParent(target)); err != nil {
			return err
		}
		file, err := archiveFS.Open(item.path)
		if err != nil {
			return invalidArchiveError(err)
		}
		_, err = d.Save(saveCtx, target, item.size, override, file)
		_ = file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func joinExtractPath(dest, member string) (string, error) {
	target := utils.CleanPath(pathpkg.Join(dest, member))
	if dest != "" && target != dest && !utils.IsPathParent(target, dest) {
		return "", notFound("invalid archive path")
	}
	return target, nil
}

func ensureDirs(ctx context.Context, d types.IDrive, path string) error {
	if path == "" {
		return nil
	}
	chain := make([]string, 0, 4)
	for p := path; p != ""; p = utils.PathParent(p) {
		chain = append(chain, p)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if _, err := d.MakeDir(ctx, chain[i]); err != nil {
			return err
		}
	}
	return nil
}
