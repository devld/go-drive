package script

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	err "go-drive/common/errors"
	"go-drive/common/types"
)

func TestGeneratedErrorClassesAreInstalled(t *testing.T) {
	vm := newPoolTestVM(t)
	for _, class := range jsErrorClasses {
		got, e := vm.Run(context.Background(), `
			(function (name) {
				var desc = Object.getOwnPropertyDescriptor(this, name);
				var ctor = this[name];
				return [
					typeof ctor,
					desc.writable,
					desc.configurable,
					desc.enumerable,
					Object.isFrozen(ctor),
					new ctor("x") instanceof Error,
					new ctor("x") instanceof ctor,
					new ctor("x").name === name,
					String(new ctor("x").stack || "").indexOf(name) === 0,
					Object.getPrototypeOf(ctor.prototype) === Error.prototype,
					Object.getPrototypeOf(ctor) === Error
				].join("|");
			})("`+class.name+`");
		`, "")
		if e != nil {
			t.Fatalf("%s: %v", class.name, e)
		}
		const want = "function|false|false|true|true|true|true|true|true|true|true"
		if got.String() != want {
			t.Errorf("%s installed as %q, want %q", class.name, got.String(), want)
		}
	}
}

func TestErrorClassesIgnoreReplacedGlobalError(t *testing.T) {
	for _, replacement := range []string{"null", "function () { throw 'replacement called'; }"} {
		t.Run(replacement, func(t *testing.T) {
			vm := newPoolTestVM(t)
			mustDefineGlobal(t, vm, "fail", NativeFunction(func(vm *VM, _ Values) any {
				vm.ThrowError(err.NewNotFoundMessageError("from go"))
				return nil
			}))
			got, e := vm.Run(context.Background(), `
				const OriginalError = Error;
				globalThis.Error = `+replacement+`;
				const results = [];
				for (const action of [
					() => { throw new NotFoundError("from js"); },
					() => fail(),
				]) {
					try { action(); } catch (e) {
						results.push(e instanceof OriginalError, e instanceof NotFoundError, e.message);
					}
				}
				results.join("|");
			`, "errors.js")
			if e != nil {
				t.Fatal(e)
			}
			if got.String() != "true|true|from js|true|true|from go" {
				t.Fatalf("caught errors = %q", got.String())
			}
			if _, e := vm.Run(context.Background(), `throw new NotFoundError("missing")`, ""); !err.IsNotFoundError(e) || e.Error() != "missing" {
				t.Fatalf("uncaught error = %v", e)
			}
		})
	}
}

func TestErrorClassStackIncludesJavaScriptFrames(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "fail", NativeFunction(func(vm *VM, _ Values) any {
		vm.ThrowError(err.NewNotFoundMessageError("from go"))
		return nil
	}))
	got, e := vm.Run(context.Background(), `
		function inner() {
			try { throw new NotFoundError("missing"); } catch (e) { return e.stack; }
		}
		function outer() { return inner(); }
		function fromGo() {
			try { fail(); } catch (e) { return e.stack; }
		}
		[outer(), fromGo()].join("\n---\n");
	`, "stack.js")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(got.String(), "\n---\n")
	if len(parts) != 2 {
		t.Fatalf("stacks = %q", got.String())
	}
	jsThrow, fromGo := parts[0], parts[1]
	for _, s := range []string{"NotFoundError: missing", "at inner (stack.js:", "at outer (stack.js:"} {
		if !strings.Contains(jsThrow, s) {
			t.Errorf("JS throw stack missing %q:\n%s", s, jsThrow)
		}
	}
	for _, s := range []string{"NotFoundError: from go", "at fromGo (stack.js:"} {
		if !strings.Contains(fromGo, s) {
			t.Errorf("Go throw stack missing %q:\n%s", s, fromGo)
		}
	}
}

func TestJavaScriptErrorClassesMapToGoErrors(t *testing.T) {
	vm := newPoolTestVM(t)
	got, e := vm.Run(context.Background(), `
		var missing;
		try { throw new NotFoundError(); } catch (e) { missing = e; }
		var remote;
		try { throw new RemoteApiError(502, "bad gateway"); } catch (e) { remote = e; }
		[
			missing instanceof Error,
			missing instanceof NotFoundError,
			missing.name,
			remote instanceof RemoteApiError,
			String(remote.status)
		].join("|");
	`, "")
	if e != nil {
		t.Fatal(e)
	}
	const want = "true|true|NotFoundError|true|502"
	if got.String() != want {
		t.Fatalf("javascript error classes = %q, want %q", got.String(), want)
	}

	if _, e = vm.Run(context.Background(), `throw new NotFoundError()`, ""); !err.IsNotFoundError(e) {
		t.Fatalf("empty NotFoundError = %v", e)
	}
	if _, e = vm.Run(context.Background(), `throw new NotFoundError("missing")`, ""); !err.IsNotFoundError(e) || e.Error() != "missing" {
		t.Fatalf("NotFoundError message = %v", e)
	}

	_, e = vm.Run(context.Background(), `throw new RemoteApiError(502, "bad gateway")`, "")
	var remote err.RemoteApiError
	if !errors.As(e, &remote) || remote.Status() != 502 || remote.Error() != "bad gateway" {
		t.Fatalf("RemoteApiError = %#v (%v)", e, e)
	}
	apiError, ok := e.(err.Error)
	if !ok || apiError.Code() != 502 {
		t.Fatalf("direct API error = %#v, want status 502", e)
	}
}

func TestUnhandledUnsupportedErrorRetainsStack(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `
function unsupportedOperation() {
	throw new UnsupportedError();
}
unsupportedOperation();
`, "unsupported.js")
	if !err.IsUnsupportedError(e) {
		t.Fatalf("error type = %T, want UnsupportedError", e)
	}
	if e.Error() != "" {
		t.Fatalf("Error() = %q, want original empty message", e.Error())
	}
	stack := FormatError(e)
	if !strings.Contains(stack, "UnsupportedError") {
		t.Fatalf("formatted error missing exception name: %q", stack)
	}
	if !strings.Contains(stack, "at unsupportedOperation (unsupported.js:") {
		t.Fatalf("formatted error missing JavaScript call stack: %q", stack)
	}
}

func TestJavaScriptErrorClassWorksWithoutNew(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `throw NotFoundError("missing")`, "")
	if !err.IsNotFoundError(e) || e.Error() != "missing" {
		t.Fatalf("call without new = %v", e)
	}
}

func TestForgedErrorNameIsNotMapped(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.Run(context.Background(), `throw Object.assign(new Error("missing"), { name: "NotFoundError" })`, "")
	if e == nil {
		t.Fatal("expected JavaScript exception")
	}
	if err.IsNotFoundError(e) {
		t.Fatalf("forged name mapped to NotFoundError: %v", e)
	}
}

func TestThrowErrorProducesJavaScriptClass(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "fail", NativeFunction(func(vm *VM, _ Values) any {
		vm.ThrowError(err.NewNotFoundMessageError("missing"))
		return nil
	}))
	if _, e := vm.Run(context.Background(), `
		try {
			fail();
		} catch (e) {
			if (!(e instanceof NotFoundError)) throw new Error("not a class instance: " + e);
			if (e.message !== "missing") throw new Error("message: " + e.message);
		}
	`, ""); e != nil {
		t.Fatal(e)
	}
}

func TestPermissionDeniedNotFoundRoundTrip(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "fail", NativeFunction(func(vm *VM, _ Values) any {
		vm.ThrowError(err.NewPermissionDeniedNotFoundError("hidden"))
		return nil
	}))
	_, e := vm.Run(context.Background(), `fail()`, "")
	if !err.IsPermissionDeniedNotFoundError(e) {
		t.Fatalf("permission-denied not found = %v", e)
	}
}

func TestDriveGetThrowsTypedJavaScriptError(t *testing.T) {
	vm := newPoolTestVM(t)
	mustDefineGlobal(t, vm, "drive", notFoundDrive{})
	if _, e := vm.Run(context.Background(), `
		try {
			drive.get("missing");
		} catch (e) {
			if (!(e instanceof NotFoundError)) throw new Error("not a class instance: " + e);
			if (e.message !== "missing") throw new Error("message: " + e.message);
		}
	`, ""); e != nil {
		t.Fatal(e)
	}
	if _, e := vm.Run(context.Background(), `drive.get("missing")`, ""); !err.IsNotFoundError(e) || e.Error() != "missing" {
		t.Fatalf("Drive.get error = %v", e)
	}
}

type notFoundDrive struct{}

func (notFoundDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{}, err.NewUnsupportedError()
}

func (notFoundDrive) Get(context.Context, string) (types.IEntry, error) {
	return nil, err.NewNotFoundMessageError("missing")
}

func (notFoundDrive) Save(types.TaskCtx, string, int64, bool, io.Reader) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (notFoundDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (notFoundDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (notFoundDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (notFoundDrive) List(context.Context, string) ([]types.IEntry, error) {
	return nil, err.NewUnsupportedError()
}

func (notFoundDrive) Delete(types.TaskCtx, string) error {
	return err.NewUnsupportedError()
}

func (notFoundDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	return nil, err.NewUnsupportedError()
}
