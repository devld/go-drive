package job

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/drive"
	s "go-drive/script"
	"strings"
)

//go:embed script-helper.js
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

	RegisterActionDef(JobActionDef{
		Name:        "script",
		DisplayName: t("name"),
		Description: t("desc"),
		ParamsForm: []types.FormItem{
			{
				Field: "code", Label: t("code"), Description: t("code_desc"),
				Type: "code", Code: &types.FormItemCode{Type: "javascript-server-jobs"},
				DefaultValue: defaultCodeValue, Required: true,
			},
		},
		Do: func(ctx context.Context, params types.SM, ch *registry.ComponentsHolder, onLog func(s string)) error {
			code := params["code"]
			eventJson := params["$event"]
			event := make(types.M, 2)
			e := json.Unmarshal([]byte(eventJson), &event)
			if e != nil {
				return fmt.Errorf("failed to parse event: %s", e.Error())
			}
			return ExecuteJobCode(ctx, code, types.M{"$event": event}, ch, onLog)
		},
	})
}

// ExecuteJobCode executes the code, and return the log and error
func ExecuteJobCode(ctx context.Context, code interface{}, globals types.M, ch *registry.ComponentsHolder, onLog func(string)) error {
	vm := baseVM.Fork()
	defer func() { _ = vm.Dispose() }()

	vm.Set("drive", s.NewDrive(vm, ch.Get("driveAccess").(*drive.Access).GetRootDrive(nil)))
	vm.Set("log", onLog)
	for k, v := range globals {
		vm.Set(k, v)
	}

	_, e := vm.Run(ctx, code)
	return e
}

var defaultCodeValue = strings.TrimLeft(`
// Available functions:
// - cp: copy files/directories
// - mv: move files/directories
// - rm: delete files/directories
// - ls: list directory
// - mkdir: create a directory
// - http: send a http request
//
// Or you can use 'drive' to do anything.

// See https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts
// See https://github.com/devld/go-drive/blob/master/docs/scripts/env/jobs.d.ts
// See https://github.com/devld/go-drive/tree/master/docs/scripts/libs

log('triggered by event: ' + JSON.stringify($event))

// do something

// examples:
// - Copy all '.js' files in 'a' to directory 'b'.
// 	 'true' means overwrite when there are existing files.
// cp('a/*.js', 'b', true)
//
// - Move all '.js' files in 'a' to directory 'b'.
//   auto rename when there are existing files.
// mv('a/*.js', 'b')

// - Delete all '.js' files in 'a' (including those in subdirectories)
// rm('a/**/*.js')

// - Do something
// drive.

// or send a http request
// log(http(newContext(), 'GET', 'https://example.com').Text())

`, "\t\n\r ")
