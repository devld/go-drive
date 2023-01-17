package script

import "go-drive/common/utils"

func initVarsForVm(v *VM) {
	v.o.Set("DEBUG", utils.IsDebugOn)

	v.o.Set("consoleWrite", WrapVmCall(v, vm_consoleWrite))

	v.o.Set("newFormData", WrapVmCall(v, vm_newFormData))
	v.o.Set("http", WrapVmCall(v, vm_http))

	v.o.Set("sleep", WrapVmCall(v, vm_sleep))

	v.o.Set("newContext", WrapVmCall(v, vm_newContext))
	v.o.Set("newContextWithTimeout", WrapVmCall(v, vm_newContextWithTimeout))
	v.o.Set("newTaskCtx", WrapVmCall(v, vm_newTaskCtx))
	v.o.Set("newLocker", WrapVmCall(v, vm_newLocker))

	v.o.Set("newBytes", WrapVmCall(v, func(vm *VM, args Values) interface{} {
		return NewBytes(vm, args.Get(0).Raw())
	}))
	v.o.Set("newEmptyBytes", WrapVmCall(v, func(vm *VM, args Values) interface{} {
		return NewEmptyBytes(vm, args.Get(0).Integer())
	}))
	v.o.Set("newTempFile", WrapVmCall(v, func(vm *VM, args Values) interface{} {
		return NewTempFile(vm)
	}))

	v.o.Set("__encToHex__", WrapVmCall(v, vm_toHex))
	v.o.Set("__encFromHex__", WrapVmCall(v, vm_fromHex))
	v.o.Set("__encBase64Encode__", WrapVmCall(v, vm_base64Encode))
	v.o.Set("__encBase64Decode__", WrapVmCall(v, vm_base64Decode))
	v.o.Set("__encURLBase64Encode__", WrapVmCall(v, vm_urlBase64Encode))
	v.o.Set("__encURLBase64Decode__", WrapVmCall(v, vm_urlBase64Decode))

	v.o.Set("__newHash__", WrapVmCall(v, vm_newHash))
	v.o.Set("__hmac__", WrapVmCall(v, vm_hmac))

	v.o.Set("buildEntriesTree", WrapVmCall(v, vm_buildEntriesTree))
	v.o.Set("flattenEntriesTree", WrapVmCall(v, vm_flattenEntriesTree))
	v.o.Set("findEntries", WrapVmCall(v, vm_findEntries))
}
