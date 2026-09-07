package job

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go-drive/common/i18n"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/drive"
	s "go-drive/script"
	"strings"
	"time"
)

const jobEventName = "$event"

//go:embed script-helper.js
var helperScript []byte
var helperProgram = s.MustCompile("script-helper.js", helperScript)

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
	started := time.Now()
	logging.For("job").Debugf("job script started")
	vm, e := newJobVM(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = vm.Dispose() }()

	if e = vm.DefineGlobal("drive", ch.Get(registry.KeyDriveAccess).(*drive.Access).GetRootDrive(nil)); e != nil {
		return e
	}
	if e = bindJobLog(vm, onLog); e != nil {
		return e
	}
	if e = setJobGlobals(vm, globals); e != nil {
		return e
	}

	_, e = vm.Run(ctx, code, "job.js")
	if e != nil {
		logging.For("job").Errorf("job script failed duration=%s: %s", time.Since(started), s.FormatError(e))
	} else {
		logging.For("job").Debugf("job script completed duration=%s", time.Since(started))
	}
	return e
}

func newJobVM(ctx context.Context) (*s.VM, error) {
	vm, e := s.NewVM()
	if e != nil {
		return nil, e
	}
	if _, e = vm.Run(ctx, helperProgram, "script-helper.js"); e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	return vm, nil
}

func bindJobLog(vm *s.VM, onLog func(string)) error {
	return vm.DefineGlobal("log", s.NativeFunction(func(_ *s.VM, args s.Values) any {
		if onLog != nil {
			onLog(s.FormatConsoleArgs(args))
		}
		return nil
	}))
}

func setJobGlobals(vm *s.VM, globals types.M) error {
	hasEvent := false
	for k, v := range globals {
		if e := vm.DefineGlobal(k, v); e != nil {
			return e
		}
		if k == jobEventName {
			hasEvent = true
		}
	}
	if !hasEvent {
		return vm.DefineGlobal(jobEventName, nil)
	}
	return nil
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
// log(http('https://example.com').text())

`, jobEventName), "\t\n\r ")
