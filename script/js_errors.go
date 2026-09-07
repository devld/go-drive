package script

import (
	stderrors "errors"

	err "go-drive/common/errors"

	"github.com/dop251/goja"
)

const jsParentError = "Error"

type jsErrorClass struct {
	name     string
	fromJS   func(obj *goja.Object) error
	toJSArgs func(e error) ([]any, bool)
}

func jsMessageArgs[T error](e error) ([]any, bool) {
	typed, ok := stderrors.AsType[T](e)
	if !ok {
		return nil, false
	}
	return []any{typed.Error()}, true
}

var jsErrorClasses = []jsErrorClass{
	{
		name:     "BadRequestError",
		fromJS:   func(obj *goja.Object) error { return err.NewBadRequestError(jsErrorMessage(obj)) },
		toJSArgs: jsMessageArgs[err.BadRequestError],
	},
	{
		name: "NotFoundError",
		fromJS: func(obj *goja.Object) error {
			message := jsErrorMessage(obj)
			if message == "" {
				return err.NewNotFoundError()
			}
			return err.NewNotFoundMessageError(message)
		},
		toJSArgs: jsMessageArgs[err.NotFoundError],
	},
	{
		name: "NotAllowedError",
		fromJS: func(obj *goja.Object) error {
			message := jsErrorMessage(obj)
			if message == "" {
				return err.NewNotAllowedError()
			}
			return err.NewNotAllowedMessageError(message)
		},
		toJSArgs: jsMessageArgs[err.NotAllowedError],
	},
	{
		name: "UnsupportedError",
		fromJS: func(obj *goja.Object) error {
			message := jsErrorMessage(obj)
			if message == "" {
				return err.NewUnsupportedError()
			}
			return err.NewUnsupportedMessageError(message)
		},
		toJSArgs: jsMessageArgs[err.UnsupportedError],
	},
	{
		name: "RemoteApiError",
		fromJS: func(obj *goja.Object) error {
			status := 0
			if statusValue := obj.Get("status"); statusValue != nil && !goja.IsUndefined(statusValue) && !goja.IsNull(statusValue) {
				status = int(statusValue.ToInteger())
			}
			return err.NewRemoteApiError(status, jsErrorMessage(obj))
		},
		toJSArgs: func(e error) ([]any, bool) {
			remote, ok := stderrors.AsType[err.RemoteApiError](e)
			if !ok {
				return nil, false
			}
			return []any{remote.Status(), remote.Error()}, true
		},
	},
}

// Builtin Error subclasses are real JavaScript Error objects, not host handles.
var jsClassBadRequestError = JSClass{
	Name:      "BadRequestError",
	Parent:    jsParentError,
	Construct: ctorMessageError("BadRequestError"),
}

var jsClassNotFoundError = JSClass{
	Name:      "NotFoundError",
	Parent:    jsParentError,
	Construct: ctorMessageError("NotFoundError"),
}

var jsClassNotAllowedError = JSClass{
	Name:      "NotAllowedError",
	Parent:    jsParentError,
	Construct: ctorMessageError("NotAllowedError"),
}

var jsClassUnsupportedError = JSClass{
	Name:      "UnsupportedError",
	Parent:    jsParentError,
	Construct: ctorMessageError("UnsupportedError"),
}

var jsClassRemoteApiError = JSClass{
	Name:   "RemoteApiError",
	Parent: jsParentError,
	Construct: func(vm *VM, args Values) any {
		status := int(args.Get(0).Integer())
		msg := jsOptionalString(args.Get(1))
		return vm.errorInstance("RemoteApiError", msg, map[string]any{
			"status": status,
		})
	},
}

func ctorMessageError(name string) NativeFunction {
	return func(vm *VM, args Values) any {
		return vm.errorInstance(name, jsOptionalString(args.Get(0)), nil)
	}
}

func jsOptionalString(v *Value) string {
	if v == nil || v.IsNil() {
		return ""
	}
	return v.String()
}

func (vm *VM) errorInstance(name, message string, extra map[string]any) *goja.Object {
	obj, e := vm.j.New(vm.jsVars.errorCtor, vm.j.ToValue(message))
	if e != nil {
		vm.ThrowError(e)
	}
	if setErr := obj.Set("name", name); setErr != nil {
		vm.ThrowError(setErr)
	}
	for key, value := range extra {
		if setErr := obj.Set(key, value); setErr != nil {
			vm.ThrowError(setErr)
		}
	}
	return obj
}

func (vm *VM) jsErrorFromGo(e error) goja.Value {
	className, args, ok := jsDriveErrorArgs(e)
	if !ok {
		return vm.j.NewGoError(e)
	}
	ctor := vm.ctors[className]
	if ctor == nil {
		return vm.j.NewGoError(e)
	}
	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = vm.j.ToValue(arg)
	}
	obj, ctorErr := vm.j.New(ctor, jsArgs...)
	if ctorErr != nil {
		return vm.j.NewGoError(e)
	}
	_ = obj.DefineDataProperty(
		"value",
		vm.j.ToValue(e),
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
	)
	return obj
}

func jsDriveErrorArgs(e error) (string, []any, bool) {
	for _, class := range jsErrorClasses {
		if args, ok := class.toJSArgs(e); ok {
			return class.name, args, true
		}
	}
	return "", nil, false
}

func errorFromValue(value *Value) error {
	if value == nil {
		return nil
	}
	return errorFromGojaValue(value.vm, value.v)
}

func errorFromGojaValue(vm *VM, v goja.Value) error {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	obj, ok := v.(*goja.Object)
	if !ok {
		return nil
	}
	original := obj.Get("value")
	if original != nil && !goja.IsUndefined(original) && !goja.IsNull(original) {
		if e, isErr := original.Export().(error); isErr {
			return e
		}
	}
	if vm == nil {
		return nil
	}
	for _, class := range jsErrorClasses {
		ctor := vm.ctors[class.name]
		if ctor != nil && vm.j.InstanceOf(v, ctor) {
			return class.fromJS(obj)
		}
	}
	return nil
}

func jsErrorMessage(obj *goja.Object) string {
	message := obj.Get("message")
	if message == nil || goja.IsUndefined(message) || goja.IsNull(message) {
		return ""
	}
	return message.String()
}
