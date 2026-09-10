package job

import "fmt"

type triggerDefRegistration struct {
	triggerType JobTriggerType
	def         *JobTriggerDef
}

var registeredTriggerDefs []triggerDefRegistration

// RegisterTriggerDef registers a trigger type
func RegisterTriggerDef(triggerType JobTriggerType, def JobTriggerDef) {
	for _, registered := range registeredTriggerDefs {
		if registered.triggerType == triggerType {
			panic(fmt.Sprintf("trigger type already registered: %s", triggerType))
		}
	}
	registeredTriggerDefs = append(registeredTriggerDefs, triggerDefRegistration{
		triggerType: triggerType,
		def:         &def,
	})
}

// GetTriggerDef returns a trigger definition by type
func GetTriggerDef(triggerType JobTriggerType) *JobTriggerDef {
	for _, registered := range registeredTriggerDefs {
		if registered.triggerType == triggerType {
			return registered.def
		}
	}
	return nil
}

// GetTriggerTypes returns all registered trigger types
func GetTriggerTypes() []JobTriggerType {
	types := make([]JobTriggerType, 0, len(registeredTriggerDefs))
	for _, registered := range registeredTriggerDefs {
		types = append(types, registered.triggerType)
	}
	return types
}

// GetTriggerDefs returns all registered trigger definitions (for API / definitions)
func GetTriggerDefs() []JobTriggerDef {
	defs := make([]JobTriggerDef, 0, len(registeredTriggerDefs))
	for _, registered := range registeredTriggerDefs {
		defs = append(defs, *registered.def)
	}
	return defs
}
