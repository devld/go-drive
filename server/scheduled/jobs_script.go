package scheduled

import (
	"context"
	_ "embed"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	s "go-drive/script"
	"strings"
)

//go:embed jobs_script-helper.js
var helperScript []byte
var baseVM *s.VM

func init() {
	vm, e := s.NewVM()
	if e != nil {
		panic(e)
	}

	_, e = vm.Run(context.Background(), helperScript)
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
				DefaultValue: defaultCodeValue, Required: true,
			},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, onLog func(s string)) error {
			return ExecuteJobCode(ctx, params["code"], ch, onLog)
		},
	})
}

// ExecuteJobCode executes the code, and return the log and error
func ExecuteJobCode(ctx context.Context, code string, ch *registry.ComponentsHolder, onLog func(string)) error {
	vm := baseVM.Fork()
	defer func() { _ = vm.Dispose() }()

	vm.Set("rootDrive", s.NewRootDrive(vm, ch.Get("rootDrive").(types.IRootDrive)))
	vm.Set("log", onLog)

	_, e := vm.Run(ctx, code)
	return e
}

var defaultCodeValue = strings.TrimLeft(`
// Available functions:
// - cp: copy file/directory
// - mv: move file/directory
// - rm: delete file/directory
// - ls: list directory
// - mkdir: create a directory
//
// Or you can use 'rootDrive.Get()' to do any thing.

log('started...')
var drive = rootDrive.Get()

// do something



`, "\t\n\r ")
