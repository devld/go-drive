package scheduled

import (
	"context"
	"errors"
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
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder) error {
			src := strings.Split(params["src"], "\n")
			dest := params["dest"]
			move := params.GetBool("move")
			override := params.GetBool("override")

			err := make([]error, 0)

			drive := ch.Get("rootDrive").(*drive.RootDrive).Get()
			for _, from := range src {
				if from == "" {
					continue
				}
				fromEntry, e := drive.Get(ctx, from)
				if e != nil {
					err = append(err, e)
					continue
				}

				if move {
					_, e = drive.Move(
						task.NewContextWrapper(ctx),
						fromEntry,
						utils.CleanPath(path.Join(dest, fromEntry.Name())),
						override)
				} else {
					_, e = drive.Copy(
						task.NewContextWrapper(ctx),
						fromEntry,
						utils.CleanPath(path.Join(dest, fromEntry.Name())),
						override)
				}
				if e != nil {
					err = append(err, e)
				}
			}
			if len(err) > 0 {
				m := strings.Builder{}
				for _, e := range err {
					m.WriteString(e.Error() + ";")
				}
				return errors.New(m.String())
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
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder) error {
			paths := strings.Split(params["paths"], "\n")

			drive := ch.Get("rootDrive").(*drive.RootDrive).Get()
			for _, p := range paths {
				if p == "" {
					continue
				}
				e := drive.Delete(task.NewContextWrapper(ctx), p)
				if e != nil && !err.IsNotFoundError(e) {
					return e
				}
			}
			return nil
		},
	})

}
