package scheduled

import (
	"context"
	_ "embed"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	s "go-drive/script"
)

//go:embed jobs_script-helper.js
var helperScript []byte
var baseVM *s.VM

func init() {
	vm, e := s.NewVM()
	if e != nil {
		panic(e)
	}

	_, e = vm.Run(helperScript)
	if e != nil {
		panic(e)
	}

	baseVM = vm
}

func init() {
	t := i18n.TPrefix("jobs.script.")

	RegisterJob(JobDefinition{
		Name:        "script",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{
				Field: "code", Label: t("code"), Description: t("code_desc"),
				Type: "code", Code: &types.FormItemCode{Type: "javascript-jobs"},
				Required: true,
			},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder) error {
			vm := baseVM.Fork()
			defer func() { _ = vm.Dispose() }()

			vm.Set("rootDrive", s.NewRootDrive(vm, ch.Get("rootDrive").(types.IRootDrive)))

			_, e := vm.Run(params["code"])
			return e
		},
	})
}
