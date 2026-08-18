package script

import (
	"context"
	"strings"
	"testing"
)

func TestRunNamedUsesScriptNameInStack(t *testing.T) {
	vm := newPoolTestVM(t)

	_, e := vm.RunNamed(context.Background(), "helper.js", `
function helperBoom() {
	throw new Error("from helper");
}
`)
	if e != nil {
		t.Fatal(e)
	}

	_, e = vm.RunNamed(context.Background(), "github.js", `
function driveBoom() {
	helperBoom();
}
`)
	if e != nil {
		t.Fatal(e)
	}

	vm = vm.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })

	_, e = vm.Call(context.Background(), "driveBoom")
	if e == nil {
		t.Fatal("expected error")
	}

	stack := ottoStack(e)
	if !strings.Contains(stack, "github.js") {
		t.Fatalf("stack missing drive script name:\n%s", stack)
	}
	if !strings.Contains(stack, "helper.js") {
		t.Fatalf("stack missing helper name:\n%s", stack)
	}
	if strings.Contains(stack, "helperBoom (<anonymous>") || strings.Contains(stack, "driveBoom (<anonymous>") {
		t.Fatalf("named functions still attributed to anonymous:\n%s", stack)
	}
}

func TestRunNamedSyntaxErrorUsesScriptName(t *testing.T) {
	vm := newPoolTestVM(t)
	_, e := vm.RunNamed(context.Background(), "broken.js", `function {`)
	if e == nil {
		t.Fatal("expected syntax error")
	}
	msg := e.Error()
	if !strings.Contains(msg, "broken.js") {
		t.Fatalf("syntax error missing script name: %v", e)
	}
}

func ottoStack(e error) string {
	if s, ok := e.(interface{ String() string }); ok {
		return s.String()
	}
	return e.Error()
}
