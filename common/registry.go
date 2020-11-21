package common

import (
	"fmt"
	"sort"
)

var registry = newComponentRegistry()

func R() *ComponentRegistry {
	return registry
}

type RegistryFunc func(*ComponentRegistry) interface{}

func newComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		init: make(map[string]componentInit),
	}
}

type componentInit struct {
	name  string
	order int
	fn    RegistryFunc
}
type ComponentRegistry struct {
	init map[string]componentInit
	c    map[string]interface{}
}

func (c *ComponentRegistry) Gets(matches func(c interface{}) bool) []interface{} {
	cs := make([]interface{}, 0)
	for _, v := range c.c {
		if matches == nil || matches(v) {
			cs = append(cs, v)
		}
	}
	return cs
}

func (c *ComponentRegistry) Get(name string) interface{} {
	if v, ok := c.c[name]; ok {
		return v
	}
	panic(fmt.Sprintf("cannot find component '%s'", name))
}

func (c *ComponentRegistry) Register(name string, fn RegistryFunc, initOrder int) {
	if _, ok := c.init[name]; ok {
		panic(fmt.Sprintf("component with name '%s' already registered", name))
	}
	c.init[name] = componentInit{name: name, order: initOrder, fn: fn}
}

func (c *ComponentRegistry) Init() {
	if c.c != nil {
		panic("")
	}
	c.callInit()
}

func (c *ComponentRegistry) callInit() {
	c.c = make(map[string]interface{})
	inits := make([]componentInit, 0)
	for _, init := range c.init {
		inits = append(inits, init)
	}
	sort.Slice(inits, func(i, j int) bool {
		return inits[i].order < inits[j].order
	})
	for _, init := range inits {
		c.c[init.name] = init.fn(c)
	}
}
