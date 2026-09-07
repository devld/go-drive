package script

import (
	"fmt"

	"github.com/dop251/goja"
)

// Program is precompiled JavaScript, reusable across VMs.
type Program struct {
	compiled *goja.Program
}

func Compile(name string, source any) (*Program, error) {
	text, e := sourceString(source)
	if e != nil {
		return nil, e
	}
	program, e := goja.Compile(name, text, false)
	if e != nil {
		return nil, e
	}
	return &Program{compiled: program}, nil
}

func MustCompile(name string, source any) *Program {
	program, e := Compile(name, source)
	if e != nil {
		panic(fmt.Sprintf("compile %s: %v", name, e))
	}
	return program
}

func sourceString(code any) (string, error) {
	switch source := code.(type) {
	case string:
		return source, nil
	case []byte:
		return string(source), nil
	default:
		return "", fmt.Errorf("unsupported script source %T", code)
	}
}
