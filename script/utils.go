package script

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	err "go-drive/common/errors"
	"go-drive/common/logging"
	"go-drive/common/task"
	"go-drive/common/types"

	"github.com/dop251/goja"
)

func newValue(vm *VM, v goja.Value) *Value {
	if v == nil {
		v = goja.Undefined()
	}
	return &Value{vm, v, nil}
}

func newValues(vm *VM, vs []goja.Value) Values {
	return Values{vm, vs}
}

type Values struct {
	vm *VM
	vs []goja.Value
}

func (vs Values) Get(index int) *Value {
	if index >= len(vs.vs) {
		return newValue(vs.vm, goja.Undefined())
	}
	return newValue(vs.vm, vs.vs[index])
}

func (vs Values) Len() int {
	return len(vs.vs)
}

// FormatConsoleArgs formats arguments like console.log.
func FormatConsoleArgs(args Values) string {
	return formatConsoleArgs(args, 0)
}

func formatConsoleArgs(args Values, start int) string {
	n := args.Len() - start
	if n <= 0 {
		return ""
	}
	msg := make([]string, n)
	for i := range msg {
		msg[i] = formatConsoleArg(args.Get(i + start))
	}
	return strings.Join(msg, " ")
}

// ConsoleStringer is a log/console summary. It is not fmt.Stringer: some
// script types already use String() as a content API.
type ConsoleStringer interface {
	ConsoleString() string
}

func formatConsoleArg(v *Value) string {
	return formatConsoleArgSeen(v, nil)
}

func formatConsoleArgSeen(v *Value, seen map[goja.Value]struct{}) string {
	ov := v.v
	if goja.IsUndefined(ov) {
		return "undefined"
	}
	if goja.IsNull(ov) {
		return "null"
	}
	obj, isObject := ov.(*goja.Object)
	if !isObject {
		return ov.String()
	}
	if _, ok := goja.AssertFunction(ov); ok {
		return ov.String()
	}
	switch obj.ClassName() {
	case "Date", "RegExp", "String", "Number", "Boolean":
		return ov.String()
	case "Error":
		return formatConsoleError(v)
	case "Array", "GoSlice", "GoArray":
		return formatConsoleArray(v, seen)
	}
	if s, ok := consoleStringOf(v); ok {
		return s
	}
	encoded, e := obj.MarshalJSON()
	if e != nil || len(encoded) == 0 {
		return ov.String()
	}
	return string(encoded)
}

const maxConsoleArrayItems = 100

func formatConsoleArray(v *Value, seen map[goja.Value]struct{}) string {
	if seen == nil {
		seen = make(map[goja.Value]struct{})
	}
	if _, ok := seen[v.v]; ok {
		return "[ ... ]"
	}
	seen[v.v] = struct{}{}
	defer delete(seen, v.v)

	n := consoleArrayLen(v)
	if n <= 0 {
		return "[ ]"
	}
	show := min(n, maxConsoleArrayItems)
	parts := make([]string, 0, show+1)
	for i := range show {
		parts = append(parts, formatConsoleArgSeen(v.Get(strconv.Itoa(i)), seen))
	}
	if n > show {
		parts = append(parts, fmt.Sprintf("... %d more items", n-show))
	}
	return "[ " + strings.Join(parts, ", ") + " ]"
}

func consoleArrayLen(v *Value) int {
	lengthVal := v.Get("length")
	if lengthVal == nil || lengthVal.IsNil() {
		return 0
	}
	n := lengthVal.Integer()
	if n <= 0 {
		return 0
	}
	return int(n)
}

func consoleStringOf(v *Value) (string, bool) {
	if v == nil || v.IsNil() {
		return "", false
	}
	if stringer, ok := v.Raw().(ConsoleStringer); ok {
		return stringer.ConsoleString(), true
	}
	return "", false
}

func formatGoInspect(kind string, parts []string, more bool) string {
	var b strings.Builder
	b.WriteString(kind)
	b.WriteString(" { ")
	b.WriteString(strings.Join(parts, ", "))
	if more {
		if len(parts) > 0 {
			b.WriteString(", ")
		}
		b.WriteString("...")
	}
	b.WriteString(" }")
	return b.String()
}

func formatConsoleError(v *Value) string {
	if stack := v.Get("stack"); stack != nil && !stack.IsNil() {
		if s := stack.String(); s != "" {
			return s
		}
	}
	return v.v.String()
}

// Value is a JavaScript value bound to one VM. It cannot cross VM or goroutine
// boundaries.
type Value struct {
	vm *VM
	v  goja.Value

	obj *goja.Object
}

func (v *Value) IsNil() bool {
	return v == nil || v.v == nil || goja.IsUndefined(v.v) || goja.IsNull(v.v)
}

func (v *Value) IsNumber() bool {
	if v == nil || v.IsNil() || v.v.ExportType() == nil {
		return false
	}
	switch v.v.ExportType().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func (v *Value) IsString() bool {
	return v != nil && !v.IsNil() && v.v.ExportType() != nil && v.v.ExportType().Kind() == reflect.String
}

func (v *Value) IsUndefined() bool {
	return v == nil || v.v == nil || goja.IsUndefined(v.v)
}

func (v *Value) IsObject() bool {
	if v == nil {
		return false
	}
	_, ok := v.v.(*goja.Object)
	return ok
}

func (v *Value) Class() string {
	if obj := v.object(); obj != nil {
		return obj.ClassName()
	}
	return ""
}

func (v *Value) String() string {
	if v.IsNil() {
		return ""
	}
	return v.v.String()
}

func (v *Value) Bool() bool {
	if v.IsNil() {
		return false
	}
	return v.v.ToBoolean()
}

func (v *Value) Integer() int64 {
	if v.IsNil() {
		return 0
	}
	return v.v.ToInteger()
}

func (v *Value) Float() float64 {
	if v.IsNil() {
		return 0
	}
	return v.v.ToFloat()
}

func (v *Value) object() *goja.Object {
	if v.IsNil() {
		return nil
	}
	if v.obj == nil {
		v.obj, _ = v.v.(*goja.Object)
	}
	return v.obj
}

func (v *Value) Get(prop string) *Value {
	if v == nil {
		return newValue(nil, goja.Undefined())
	}
	obj := v.object()
	if obj == nil {
		return newValue(v.vm, goja.Undefined())
	}
	return newValue(v.vm, obj.Get(prop))
}

func (v *Value) Has(prop string) bool {
	return !v.Get(prop).IsNil()
}

func (v *Value) Keys() []string {
	obj := v.object()
	if obj == nil {
		return nil
	}
	return obj.Keys()
}

func (v *Value) SM() types.SM {
	obj := v.object()
	if obj == nil {
		return nil
	}
	r := make(types.SM)

	for _, k := range obj.Keys() {
		p := obj.Get(k)
		// An earlier getter (or a Proxy trap) may remove an enumerated key.
		if p == nil {
			continue
		}
		r[k] = p.String()
	}
	return r
}

func (v *Value) IsArray() bool {
	if v == nil || !v.IsObject() {
		return false
	}
	// Use the captured intrinsic: Proxy arrays have ClassName "Object".
	result, e := v.vm.jsVars.arrayIsArray(goja.Undefined(), v.v)
	if e != nil {
		panic(e)
	}
	return result.ToBoolean()
}

func (v *Value) isPromise() bool {
	if v == nil || v.IsNil() {
		return false
	}
	if v.Class() == "Promise" {
		return true
	}
	t := v.v.ExportType()
	return t != nil && t == reflect.TypeFor[*goja.Promise]()
}

func (v *Value) Array() []*Value {
	if !v.IsArray() {
		return nil
	}
	obj := v.object()
	length := obj.Get("length").ToInteger()
	if length < 0 {
		return nil
	}
	n := int(length)
	r := make([]*Value, n)
	for i := range n {
		ev := obj.Get(strconv.Itoa(i))
		r[i] = newValue(v.vm, ev)
	}
	return r
}

func (v *Value) M() types.M {
	if v.IsNil() {
		return nil
	}
	obj := v.object()
	if obj == nil {
		return nil
	}
	r := make(types.M)

	for _, k := range obj.Keys() {
		p := obj.Get(k)
		r[k] = unwrapHost(p)
	}
	return r
}

func (v *Value) ParseInto(dest any) (err error) {
	defer func() {
		if er := recover(); er != nil {
			if ee, ok := er.(error); ok {
				err = ee
			} else {
				err = fmt.Errorf("%v", er)
			}
		}
	}()
	parseValue(v, reflect.ValueOf(dest))
	return nil
}

// Parse is ParseInto for host functions: failure throws in the Value's VM.
func Parse[T any](v *Value) T {
	var out T
	if e := v.ParseInto(&out); e != nil {
		v.vm.ThrowError(e)
	}
	return out
}

func (v *Value) Raw() any {
	if v == nil || v.IsNil() {
		return nil
	}
	return unwrapExported(v.v.Export())
}

func (v *Value) Call(thisValue any, args ...any) *Value {
	rv, e := v.vm.callValue(v.vm.ExecutionContext(), v, v.vm.j.ToValue(thisValue), args...)
	if e != nil {
		v.vm.ThrowError(e)
	}
	return rv
}

// detachJSValue copies a JS value into Go data that does not hold Runtime
// objects (Proxy, Object, exported functions). Nested maps and slices are
// walked so entry meta can be read after the producing VM is returned.
func detachJSValue(ov *Value, seen map[goja.Value]any) any {
	if ov == nil || ov.IsNil() {
		return nil
	}
	if ov.IsObject() {
		if cached, ok := seen[ov.v]; ok {
			return cached
		}
	}

	if ov.v.ExportType() == hostObjectType {
		if handle := unwrapExported(ov.v.Export()); handle != nil && isJSClassHandleType(reflect.TypeOf(handle)) {
			if b, ok := HostAs[jsBytes](handle); ok {
				return append([]byte(nil), b.NativeBytes()...)
			}
			if cs, ok := handle.(ConsoleStringer); ok {
				return cs.ConsoleString()
			}
			return ov.Class()
		}
	}

	if _, ok := goja.AssertFunction(ov.v); ok {
		return ov.String()
	}

	if ov.IsArray() {
		arr := ov.Array()
		out := make([]any, len(arr))
		seen[ov.v] = out
		for i, item := range arr {
			out[i] = detachJSValue(item, seen)
		}
		return out
	}

	if ov.IsObject() {
		switch ov.Class() {
		case "Date":
			return ov.v.Export()
		case "RegExp", "String", "Number", "Boolean":
			return unwrapExported(ov.v.Export())
		}
		keys := ov.Keys()
		out := make(types.M, len(keys)+3)
		seen[ov.v] = out
		if strings.HasSuffix(ov.Class(), "Error") {
			if msg := ov.Get("message"); msg != nil && !msg.IsUndefined() {
				out["message"] = msg.String()
			}
			if name := ov.Get("name"); name != nil && !name.IsUndefined() {
				out["name"] = name.String()
			}
			if stack := ov.Get("stack"); stack != nil && !stack.IsUndefined() {
				out["stack"] = stack.String()
			}
		}
		for _, k := range keys {
			p := ov.Get(k)
			if p == nil || p.IsUndefined() {
				continue
			}
			out[k] = detachJSValue(p, seen)
		}
		return out
	}

	return unwrapExported(ov.v.Export())
}

func parseValue(ov *Value, v reflect.Value) {
	if ov == nil || !v.IsValid() {
		return
	}
	vt := v.Type()
	switch v.Kind() {
	case reflect.Ptr:
		if ov.IsNil() {
			return
		}
		if v.IsNil() {
			value := reflect.New(vt.Elem())
			v.Set(value)
		}
		parseValue(ov, v.Elem())
	case reflect.Map:
		keys := ov.Keys()
		if keys == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		valueType := vt.Elem()
		r := reflect.MakeMapWithSize(v.Type(), len(keys))
		for _, key := range keys {
			value := reflect.New(valueType)
			parseValue(ov.Get(key), value.Elem())
			r.SetMapIndex(reflect.ValueOf(key), value.Elem())
		}
		v.Set(r)
	case reflect.Slice:
		arr := ov.Array()
		if arr == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		n := len(arr)
		r := reflect.MakeSlice(vt, 0, n)

		for i := range n {
			value := reflect.New(vt.Elem())
			parseValue(arr[i], value.Elem())
			r = reflect.Append(r, value.Elem())
		}
		v.Set(r)
	case reflect.Array:
		arr := ov.Array()
		if arr == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		n := len(arr)
		var arrLen = vt.Len()
		r := reflect.New(reflect.ArrayOf(arrLen, vt.Elem()))
		for i := 0; i < n && i < arrLen; i++ {
			value := reflect.New(vt.Elem())
			parseValue(arr[i], value.Elem())
			r.Elem().Index(i).Set(value.Elem())
		}
		v.Set(r.Elem())
	case reflect.Interface:
		if ov.IsNil() {
			v.Set(reflect.Zero(vt))
			return
		}
		exported := detachJSValue(ov, make(map[goja.Value]any))
		if exported == nil {
			v.Set(reflect.Zero(vt))
			return
		}
		val := reflect.ValueOf(exported)
		if !val.IsValid() || !val.Type().AssignableTo(vt) {
			return
		}
		v.Set(val)
	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			fv := v.Field(i)
			if !fv.CanSet() {
				continue
			}
			var value *Value
			for _, fName := range jsObjectFieldNames(vt.Field(i)) {
				value = ov.Get(fName)
				if value != nil && !value.IsNil() {
					break
				}
			}
			if value == nil || value.IsNil() {
				continue
			}
			parseValue(value, fv)
		}
	case reflect.Bool:
		v.SetBool(ov.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(ov.Integer())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(uint64(ov.Integer()))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(ov.Float())
	case reflect.String:
		v.SetString(ov.String())
	}
}

func captureScriptResult(vm *VM, fn func() error) (e error) {
	defer func() {
		e = finalizeScriptError(vm, e, recover())
	}()
	return fn()
}

// finalizeScriptError turns a JS panic or goja error into a Go error.
// Native panics propagate. Conversion may read JS; conversionFallback
// handles a second throw without touching JS objects.
func finalizeScriptError(vm *VM, runErr error, panicVal any) (e error) {
	if panicVal != nil && !isJSException(panicVal) {
		panic(panicVal)
	}
	defer func() {
		if r := recover(); r != nil {
			e = conversionFallback(vm, runErr, r)
		}
	}()
	if panicVal != nil {
		switch x := panicVal.(type) {
		case error:
			runErr = x
		case goja.Value:
			if original := errorFromGojaValue(vm, x); original != nil {
				runErr = original
			} else {
				runErr = fmt.Errorf("%s", x.String())
			}
		}
	}
	return convertScriptError(vm, runErr)
}

func isJSException(r any) bool {
	switch r.(type) {
	case goja.Value, *goja.InterruptedError, *goja.Exception:
		return true
	default:
		return false
	}
}

func convertScriptError(vm *VM, e error) error {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *goja.InterruptedError:
		return unwrapInterrupted(x)
	case *goja.Exception:
		if original := errorFromGojaValue(vm, x.Value()); original != nil {
			return snapshotException(x, convertScriptError(vm, original))
		}
		return snapshotException(x, nil)
	}
	if re, ok := errors.AsType[err.Error](e); ok {
		return re
	}
	return e
}

// jsRuntimeError is a JavaScript exception copied into Go text. It does not
// hold Runtime or goja values, so Error/String are safe after Dispose.
type jsRuntimeError struct {
	text  string
	cause error
}

func (e *jsRuntimeError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.text
}

func (e *jsRuntimeError) Unwrap() error { return e.cause }

func (e *jsRuntimeError) String() string { return e.text }

func (e *jsRuntimeError) scriptStack() string { return e.text }

type jsRuntimeAPIError struct {
	*jsRuntimeError
	apiError err.Error
}

func (e *jsRuntimeAPIError) Code() int { return e.apiError.Code() }

type jsRuntimeAPIDataError struct {
	*jsRuntimeAPIError
	apiDataError err.ErrorWithData
}

func (e *jsRuntimeAPIDataError) Data() types.M { return e.apiDataError.Data() }

// FormatError returns the JavaScript exception and its call stack when e was
// produced by Run or Call. For other errors it returns Error().
func FormatError(e error) string {
	if e == nil {
		return ""
	}
	var stacked interface{ scriptStack() string }
	if errors.As(e, &stacked) {
		return stacked.scriptStack()
	}
	return e.Error()
}

func snapshotException(exception *goja.Exception, cause error) error {
	text := exception.String()
	logging.For("script").Errorf("goja runtime error: %s", text)
	runtimeError := &jsRuntimeError{text: text, cause: cause}
	if apiDataError, ok := errors.AsType[err.ErrorWithData](cause); ok {
		return &jsRuntimeAPIDataError{
			jsRuntimeAPIError: &jsRuntimeAPIError{runtimeError, apiDataError},
			apiDataError:      apiDataError,
		}
	}
	if apiError, ok := errors.AsType[err.Error](cause); ok {
		return &jsRuntimeAPIError{runtimeError, apiError}
	}
	return runtimeError
}

func unwrapInterrupted(interrupted *goja.InterruptedError) error {
	if cause := interrupted.Unwrap(); cause != nil {
		return cause
	}
	return interrupted
}

func runContextError(vm *VM) error {
	if vm == nil || vm.runCtx == nil {
		return nil
	}
	return vm.runCtx.Err()
}

func stackSnapshot(exception *goja.Exception) error {
	var b bytes.Buffer
	b.WriteString("javascript exception\n")
	for _, frame := range exception.Stack() {
		b.WriteString("\tat ")
		frame.Write(&b)
		b.WriteByte('\n')
	}
	return &jsRuntimeError{text: b.String()}
}

func opaqueScriptError(e error) error {
	if e == nil {
		return errors.New("javascript exception")
	}
	if exception, ok := e.(*goja.Exception); ok {
		return stackSnapshot(exception)
	}
	return e
}

// conversionFallback must not read JS objects. Proxy traps and getters can
// throw again, and this already runs inside recover.
func conversionFallback(vm *VM, runErr error, r any) (e error) {
	defer func() {
		if recover() != nil {
			if ctxErr := runContextError(vm); ctxErr != nil {
				e = ctxErr
				return
			}
			e = errors.New("javascript exception")
		}
	}()
	if ctxErr := runContextError(vm); ctxErr != nil {
		return ctxErr
	}
	switch x := r.(type) {
	case *goja.InterruptedError:
		return unwrapInterrupted(x)
	case *goja.Exception:
		return stackSnapshot(x)
	}
	return opaqueScriptError(runErr)
}

// ParseDuration parses a JS Duration or Go duration string (`2s`, `1h30m`,
// optional `2d`) without throwing.
func ParseDuration(v any) (time.Duration, bool) {
	return durationFrom(v)
}

// GetDuration parses a duration for a host function. A missing or invalid value
// throws a TypeError with required as its message.
func GetDuration(vm *VM, v any, required string) time.Duration {
	d, ok := durationFrom(v)
	if !ok {
		vm.ThrowTypeError(required)
	}
	return d
}

func durationFrom(v any) (time.Duration, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case *Value:
		if x == nil || x.IsNil() {
			return 0, false
		}
		if x.IsString() {
			return types.ParseDuration(x.String())
		}
		if x.IsNumber() {
			return time.Duration(x.Integer()), true
		}
		return durationFrom(x.Raw())
	case time.Duration:
		return x, true
	case string:
		return types.ParseDuration(x)
	case int64:
		return time.Duration(x), true
	case int:
		return time.Duration(x), true
	case int32:
		return time.Duration(x), true
	case uint64:
		return time.Duration(x), true
	case float64:
		return time.Duration(x), true
	default:
		return 0, false
	}
}

func runTaskCtx(ctx context.Context) types.TaskCtx {
	if tc, ok := ctx.(types.TaskCtx); ok {
		return tc
	}
	return task.NewContextWrapper(ctx)
}
