package job

import (
	"fmt"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"path"
	"strings"
)

func init() {
	t := i18n.TPrefix("jobs.copy.")
	RegisterActionDef(JobActionDef{
		Name:        "copy",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "src", Label: t("src"), Description: t("src_desc"), Type: "textarea", Required: true},
			{Field: "dest", Label: t("dest"), Description: t("dest_desc"), Type: "text", Required: true},
			{Field: "override", Label: t("override"), Description: t("override_desc"), Type: "checkbox"},
			{Field: "move", Label: t("move"), Description: t("move_desc"), Type: "checkbox"},
		},
		Do: func(ctx types.TaskCtx, params types.SM, ch *registry.ComponentsHolder, log func(string)) error {
			src := strings.Split(params["src"], "\n")
			dest := params["dest"]
			move := params.GetBool("move")
			override := params.GetBool("override")

			drive := ch.Get(registry.KeyDriveAccess).(*drive.Access).GetRootDrive(nil)
			opCtx := task.NewTaskCtxWrapper(ctx, false, false)
			matched := make([][]types.IEntry, 0, len(src))

			for _, from := range src {
				if from == "" {
					continue
				}
				fromEntries, e := driveutil.FindEntries(opCtx, drive, from, false)
				if e != nil {
					return e
				}
				log(fmt.Sprintf("'%s' matched %d entries", from, len(fromEntries)))
				matched = append(matched, fromEntries)
			}

			return runEntryOperations(ctx, matched, false, func(fromEntry types.IEntry) error {
				var e error
				if move {
					log(fmt.Sprintf("  move '%s'", fromEntry.Path()))
					_, e = drive.Move(
						opCtx,
						fromEntry,
						utils.CleanPath(path.Join(dest, fromEntry.Name())),
						override)
				} else {
					log(fmt.Sprintf("  copy '%s'", fromEntry.Path()))
					_, e = drive.Copy(
						opCtx,
						fromEntry,
						utils.CleanPath(path.Join(dest, fromEntry.Name())),
						override)
				}
				if e != nil {
					return e
				}
				return nil
			})
		},
	})

	t = i18n.TPrefix("jobs.delete.")
	RegisterActionDef(JobActionDef{
		Name:        "delete",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "paths", Label: t("paths"), Description: t("paths_desc"), Type: "textarea", Required: true},
		},
		Do: func(ctx types.TaskCtx, params types.SM, ch *registry.ComponentsHolder, log func(string)) error {
			paths := strings.Split(params["paths"], "\n")

			drive := ch.Get(registry.KeyDriveAccess).(*drive.Access).GetRootDrive(nil)
			opCtx := task.NewTaskCtxWrapper(ctx, false, false)
			matched := make([][]types.IEntry, 0, len(paths))
			for _, p := range paths {
				if p == "" {
					continue
				}
				entries, e := driveutil.FindEntries(opCtx, drive, p, false)
				if e != nil {
					return e
				}
				log(fmt.Sprintf("'%s' matched %d entries", p, len(entries)))
				matched = append(matched, entries)
			}

			return runEntryOperations(ctx, matched, true, func(entry types.IEntry) error {
				log(fmt.Sprintf("  delete '%s'", entry.Path()))
				e := drive.Delete(opCtx, entry.Path())
				if e != nil && !err.IsNotFoundError(e) {
					return e
				}
				return nil
			})
		},
	})

}

func runEntryOperations(ctx types.TaskCtx, groups [][]types.IEntry, reverse bool, do func(types.IEntry) error) error {
	var total int64
	for _, entries := range groups {
		total += int64(len(entries))
	}
	ctx.Total(total, true)

	run := func(entry types.IEntry) error {
		if e := do(entry); e != nil {
			return e
		}
		ctx.Progress(1, false)
		return nil
	}
	for _, entries := range groups {
		if reverse {
			for i := len(entries) - 1; i >= 0; i-- {
				if e := run(entries[i]); e != nil {
					return e
				}
			}
			continue
		}
		for _, entry := range entries {
			if e := run(entry); e != nil {
				return e
			}
		}
	}
	return nil
}
