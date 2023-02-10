package registry

import (
	"fmt"
	"go-drive/common/types"
)

type ComponentsHolder struct {
	c map[string]interface{}
}

func NewComponentHolder() *ComponentsHolder {
	return &ComponentsHolder{c: make(map[string]interface{})}
}

func (c *ComponentsHolder) Add(name string, component interface{}) {
	if _, ok := c.c[name]; ok {
		panic(fmt.Sprintf("component with name '%s' already added", name))
	}
	c.c[name] = component
}

func (c *ComponentsHolder) Get(name string) interface{} {
	if v, ok := c.c[name]; ok {
		return v
	}
	panic(fmt.Sprintf("cannot find component '%s'", name))
}

func (c *ComponentsHolder) Gets(matches func(c interface{}) bool) []interface{} {
	cs := make([]interface{}, 0)
	for _, v := range c.c {
		if matches == nil || matches(v) {
			cs = append(cs, v)
		}
	}
	return cs
}

func (c *ComponentsHolder) Dispose() error {
	disposables := c.Gets(func(c interface{}) bool {
		_, ok := c.(types.IDisposable)
		return ok
	})
	for _, a := range disposables {
		_ = a.(types.IDisposable).Dispose()
	}
	return nil
}
