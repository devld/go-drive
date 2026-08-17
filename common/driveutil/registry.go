package driveutil

import (
	"fmt"
	"go-drive/common/registry"
	"strings"
	"sync"
)

// DriveRegistry stores the drive factories for one application instance.
type DriveRegistry struct {
	sync.RWMutex
	factories []DriveFactoryConfig
}

// NewDriveRegistry creates an empty drive registry.
func NewDriveRegistry(ch *registry.ComponentsHolder) *DriveRegistry {
	driveRegistry := &DriveRegistry{}
	ch.Add(registry.KeyDriveRegistry, driveRegistry)
	return driveRegistry
}

// RegisterDrive registers a drive factory. Static factories are normally
// registered during application initialization, but the same API can also be
// used for a runtime factory whose type is not already registered.
func (r *DriveRegistry) RegisterDrive(factory DriveFactoryConfig) {
	r.Lock()
	defer r.Unlock()

	if r.indexOfLocked(factory.Type) >= 0 {
		panic(factory.Type + " already registered")
	}
	r.factories = append(r.factories, factory)
}

// UnregisterDrive removes a factory from the registry. It is intended for
// runtime registrations; removing a factory does not dispose instances that
// have already been created from it.
func (r *DriveRegistry) UnregisterDrive(typeName string) bool {
	r.Lock()
	defer r.Unlock()

	i := r.indexOfLocked(typeName)
	if i < 0 {
		return false
	}
	r.factories = append(r.factories[:i], r.factories[i+1:]...)
	return true
}

// ReplaceDriveGroup atomically replaces all factories whose type starts with
// prefix. This keeps a dynamic provider from exposing a partially refreshed
// set of factories while its files are being installed or removed.
func (r *DriveRegistry) ReplaceDriveGroup(prefix string, factories []DriveFactoryConfig) error {
	seen := make(map[string]struct{}, len(factories))
	for _, factory := range factories {
		if !strings.HasPrefix(factory.Type, prefix) {
			return fmt.Errorf("drive factory %q is not in group %q", factory.Type, prefix)
		}
		if _, exists := seen[factory.Type]; exists {
			return fmt.Errorf("duplicate drive factory %q in group %q", factory.Type, prefix)
		}
		seen[factory.Type] = struct{}{}
	}

	r.Lock()
	defer r.Unlock()

	kept := r.factories[:0]
	for _, factory := range r.factories {
		if !strings.HasPrefix(factory.Type, prefix) {
			kept = append(kept, factory)
		}
	}
	r.factories = append(kept, factories...)
	return nil
}

func (r *DriveRegistry) GetDrive(typeName string) *DriveFactoryConfig {
	r.RLock()
	defer r.RUnlock()

	i := r.indexOfLocked(typeName)
	if i < 0 {
		return nil
	}
	factory := r.factories[i]
	return &factory
}

func (r *DriveRegistry) GetRegisteredDrives() []DriveFactoryConfig {
	r.RLock()
	factories := make([]DriveFactoryConfig, len(r.factories))
	copy(factories, r.factories)
	r.RUnlock()
	return factories
}

func (r *DriveRegistry) indexOfLocked(typeName string) int {
	for i, factory := range r.factories {
		if factory.Type == typeName {
			return i
		}
	}
	return -1
}
