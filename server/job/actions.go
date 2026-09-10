package job

import "fmt"

var registeredActionDefs []*JobActionDef

func RegisterActionDef(def JobActionDef) {
	for _, registered := range registeredActionDefs {
		if registered.Name == def.Name {
			panic(fmt.Sprintf("action '%s' already registered", def.Name))
		}
	}
	registeredActionDefs = append(registeredActionDefs, &def)
}

func GetActionDef(name string) *JobActionDef {
	for _, def := range registeredActionDefs {
		if def.Name == name {
			d := *def
			return &d
		}
	}
	return nil
}

func GetActionDefs() []JobActionDef {
	defs := make([]JobActionDef, 0, len(registeredActionDefs))
	for _, d := range registeredActionDefs {
		defs = append(defs, *d)
	}
	return defs
}
