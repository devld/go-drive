package scheduled

import (
	"context"
	"fmt"
	"go-drive/common/drive_util"
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
	RegisterJob(JobDefinition{
		Name:        "copy",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "src", Label: t("src"), Description: t("src_desc"), Type: "textarea", Required: true},
			{Field: "dest", Label: t("dest"), Description: t("dest_desc"), Type: "text", Required: true},
			{Field: "override", Label: t("override"), Description: t("override_desc"), Type: "checkbox"},
			{Field: "move", Label: t("move"), Description: t("move_desc"), Type: "checkbox"},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, log func(string)) error {
			src := strings.Split(params["src"], "\n")
			dest := params["dest"]
			move := params.GetBool("move")
			override := params.GetBool("override")

			drive := ch.Get("driveAccess").(*drive.Access).GetRootDrive(nil)

			for _, from := range src {
				if from == "" {
					continue
				}
				fromEntries, e := drive_util.FindEntries(task.NewContextWrapper(ctx), drive, from, false)
				if e != nil {
					return e
				}
				log(fmt.Sprintf("'%s' matched %d entries", from, len(fromEntries)))

				for _, fromEntry := range fromEntries {
					if move {
						log(fmt.Sprintf("  move '%s'", fromEntry.Path()))
						_, e = drive.Move(
							task.NewContextWrapper(ctx),
							fromEntry,
							utils.CleanPath(path.Join(dest, fromEntry.Name())),
							override)
					} else {
						log(fmt.Sprintf("  copy '%s'", fromEntry.Path()))
						_, e = drive.Copy(
							task.NewContextWrapper(ctx),
							fromEntry,
							utils.CleanPath(path.Join(dest, fromEntry.Name())),
							override)
					}
					if e != nil {
						return e
					}
				}
			}
			return nil
		},
	})

	t = i18n.TPrefix("jobs.delete.")
	RegisterJob(JobDefinition{
		Name:        "delete",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{Field: "paths", Label: t("paths"), Description: t("paths_desc"), Type: "textarea", Required: true},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, log func(string)) error {
			paths := strings.Split(params["paths"], "\n")

			drive := ch.Get("driveAccess").(*drive.Access).GetRootDrive(nil)
			for _, p := range paths {
				if p == "" {
					continue
				}
				entries, e := drive_util.FindEntries(task.NewContextWrapper(ctx), drive, p, false)
				if e != nil {
					return e
				}
				log(fmt.Sprintf("'%s' matched %d entries", p, len(entries)))
				for i := len(entries) - 1; i >= 0; i-- {
					log(fmt.Sprintf("  delete '%s'", entries[i].Path()))
					e := drive.Delete(task.NewContextWrapper(ctx), entries[i].Path())
					if e != nil && !err.IsNotFoundError(e) {
						return e
					}
				}
			}
			return nil
		},
	})

}
