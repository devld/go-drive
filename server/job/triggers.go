package job

import "fmt"

var registeredTriggerDefs = make(map[JobTriggerType]*JobTriggerDef)

// RegisterTriggerDef registers a trigger type
func RegisterTriggerDef(triggerType JobTriggerType, def JobTriggerDef) {
	if _, exists := registeredTriggerDefs[triggerType]; exists {
		panic(fmt.Sprintf("trigger type already registered: %s", triggerType))
	}
	registeredTriggerDefs[triggerType] = &def
}

// GetTriggerDef returns a trigger definition by type
func GetTriggerDef(triggerType JobTriggerType) *JobTriggerDef {
	return registeredTriggerDefs[triggerType]
}

// GetTriggerTypes returns all registered trigger types
func GetTriggerTypes() []JobTriggerType {
	types := make([]JobTriggerType, 0, len(registeredTriggerDefs))
	for t := range registeredTriggerDefs {
		types = append(types, t)
	}
	return types
}

// GetTriggerDefs returns all registered trigger definitions (for API / definitions)
func GetTriggerDefs() []JobTriggerDef {
	defs := make([]JobTriggerDef, 0, len(registeredTriggerDefs))
	for _, d := range registeredTriggerDefs {
		defs = append(defs, *d)
	}
	return defs
}
