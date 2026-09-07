package script

import (
	"io"
	"time"

	"go-drive/common/logging"
)

var jsGlobalFunctions = map[string]NativeFunction{
	"http":             jsHTTP,
	"sleep":            jsSleep,
	"parseDuration":    jsParseDuration,
	"buildEntriesTree": jsBuildEntriesTree,
	"findEntries":      jsFindEntries,
}

var jsConsole = map[string]any{
	"debug": jsConsoleLevel("debug"),
	"error": jsConsoleLevel("error"),
	"info":  jsConsoleLevel("info"),
	"log":   jsConsoleLevel("log"),
	"warn":  jsConsoleLevel("warn"),
}

func (vm *VM) installHostGlobals() error {
	for _, name := range sortedMapKeys(jsGlobalFunctions) {
		if e := vm.DefineGlobal(name, jsGlobalFunctions[name]); e != nil {
			return e
		}
	}
	if e := vm.DefineGlobal("urlUtils", jsURLUtils); e != nil {
		return e
	}
	if e := vm.DefineGlobal("console", jsConsole); e != nil {
		return e
	}
	if e := vm.DefineGlobal("SEEK_START", io.SeekStart); e != nil {
		return e
	}
	if e := vm.DefineGlobal("SEEK_CURRENT", io.SeekCurrent); e != nil {
		return e
	}
	return vm.DefineGlobal("SEEK_END", io.SeekEnd)
}

func jsConsoleLevel(level string) NativeFunction {
	return func(_ *VM, args Values) any {
		message := formatConsoleArgs(args, 0)
		logger := logging.For("script")
		switch level {
		case "debug":
			logger.Debugf("%s", message)
		case "warn", "warning":
			logger.Warnf("%s", message)
		case "error":
			logger.Errorf("%s", message)
		default:
			logger.Infof("%s", message)
		}
		return nil
	}
}

var jsSleep = NativeFunction(func(vm *VM, args Values) any {
	duration := GetDuration(vm, args.Get(0), "sleep requires a Duration or duration string")
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-vm.ExecutionContext().Done():
		vm.ThrowError(vm.ExecutionContext().Err())
	}
	return nil
})

var jsParseDuration = NativeFunction(func(vm *VM, args Values) any {
	v := args.Get(0)
	if v.IsNil() {
		return time.Duration(0)
	}
	duration := GetDuration(vm, v, "parseDuration requires a Duration or duration string")
	return duration
})

// NativeFunction is a Go function exposed to JavaScript. Throw from inside a
// call; after Run/Call returns, use a normal Go error. ToJSValue panics if a
// Go function does not have this signature.
type NativeFunction func(vm *VM, args Values) any
