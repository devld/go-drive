package job

import "fmt"

var registeredActionDefs = make(map[string]*JobActionDef)

func RegisterActionDef(def JobActionDef) {
	if _, exists := registeredActionDefs[def.Name]; exists {
		panic(fmt.Sprintf("action '%s' already registered", def.Name))
	}
	registeredActionDefs[def.Name] = &def
}

func GetActionDef(name string) *JobActionDef {
	def, exists := registeredActionDefs[name]
	if !exists {
		return nil
	}
	d := *def
	return &d
}

func GetActionDefs() []JobActionDef {
	defs := make([]JobActionDef, 0, len(registeredActionDefs))
	for _, d := range registeredActionDefs {
		defs = append(defs, *d)
	}
	return defs
}
