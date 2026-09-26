package registry

import (
	"fmt"
	"go-drive/common/types"
	"reflect"
)

// ComponentsHolder stores components anonymously. Callers that need one
// dependency receive it as a parameter. The holder is for collecting every
// component that implements an interface, and for disposing them together.
type ComponentsHolder struct {
	items []any
}

func NewComponentHolder() *ComponentsHolder {
	return &ComponentsHolder{}
}

// Add stores component. The same pointer cannot be added twice. A comparable
// value cannot be added twice when it is equal to one already stored, because
// a copied value has no separate identity.
func (c *ComponentsHolder) Add(component any) {
	if component == nil || isNilComponent(component) {
		panic("nil component")
	}
	for _, existing := range c.items {
		if sameComponent(existing, component) {
			panic(fmt.Sprintf("component %T already added", component))
		}
	}
	c.items = append(c.items, component)
}

// Gets returns every stored component that implements T.
func Gets[T any](c *ComponentsHolder) []T {
	result := make([]T, 0)
	for _, item := range c.items {
		if value, ok := item.(T); ok {
			result = append(result, value)
		}
	}
	return result
}

func (c *ComponentsHolder) Dispose() error {
	disposables := Gets[types.IDisposable](c)
	for i := len(disposables) - 1; i >= 0; i-- {
		_ = disposables[i].Dispose()
	}
	return nil
}

func isNilComponent(component any) bool {
	value := reflect.ValueOf(component)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func sameComponent(existing, component any) bool {
	left := reflect.ValueOf(existing)
	right := reflect.ValueOf(component)
	if left.Type() != right.Type() {
		return false
	}
	switch left.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Slice:
		return left.Pointer() == right.Pointer()
	default:
		if !left.Type().Comparable() {
			return false
		}
		return existing == component
	}
}
