package registry

import (
	"fmt"
	"go-drive/common/types"
)

type ComponentsHolder struct {
	c map[componentKey]any
}

func NewComponentHolder() *ComponentsHolder {
	return &ComponentsHolder{c: make(map[componentKey]any)}
}

func (c *ComponentsHolder) Add(key componentKey, component any) {
	if _, ok := c.c[key]; ok {
		panic(fmt.Sprintf("component with key '%s' already added", key))
	}
	c.c[key] = component
}

func (c *ComponentsHolder) Get(key componentKey) any {
	if v, ok := c.c[key]; ok {
		return v
	}
	panic(fmt.Sprintf("cannot find component '%s'", key))
}

func (c *ComponentsHolder) Gets(matches func(c any) bool) []any {
	cs := make([]any, 0)
	for _, v := range c.c {
		if matches == nil || matches(v) {
			cs = append(cs, v)
		}
	}
	return cs
}

func (c *ComponentsHolder) Dispose() error {
	disposables := c.Gets(func(c any) bool {
		_, ok := c.(types.IDisposable)
		return ok
	})
	for _, a := range disposables {
		_ = a.(types.IDisposable).Dispose()
	}
	return nil
}
