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

const jobEventName = "$event"

//go:embed script-helper.js
var helperScript []byte
var baseVM *s.VM

func init() {
	vm, e := s.NewVM()
	if e != nil {
		panic(e)
	}

	_, e = vm.RunNamed(context.Background(), "script-helper.js", helperScript)
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
			eventJson := params[jobEventName]
			event := make(types.M, 2)
			e := json.Unmarshal([]byte(eventJson), &event)
			if e != nil {
				return fmt.Errorf("failed to parse event: %s", e.Error())
			}
			return ExecuteJobCode(ctx, code, types.M{jobEventName: event}, ch, onLog)
		},
	})
}

// ExecuteJobCode executes the code, and return the log and error
func ExecuteJobCode(ctx context.Context, code any, globals types.M, ch *registry.ComponentsHolder, onLog func(string)) error {
	vm := baseVM.Fork()
	defer func() { _ = vm.Dispose() }()

	vm.Set("drive", s.NewDrive(ch.Get(registry.KeyDriveAccess).(*drive.Access).GetRootDrive(nil)))
	bindJobLog(vm, onLog)
	setJobGlobals(vm, globals)

	_, e := vm.RunNamed(ctx, "job.js", code)
	return e
}

func bindJobLog(vm *s.VM, onLog func(string)) {
	vm.Set("log", s.WrapVmCall(vm, func(_ *s.VM, args s.Values) any {
		if onLog != nil {
			onLog(s.FormatConsoleArgs(args))
		}
		return nil
	}))
}

func setJobGlobals(vm *s.VM, globals types.M) {
	hasEvent := false
	for k, v := range globals {
		vm.Set(k, v)
		if k == jobEventName {
			hasEvent = true
		}
	}
	if !hasEvent {
		vm.SetUndefined(jobEventName)
	}
}

var defaultCodeValue = strings.TrimLeft(fmt.Sprintf(`
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

log('triggered by event:', %s)

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

`, jobEventName), "\t\n\r ")
