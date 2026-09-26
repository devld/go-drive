package auth

import (
	"fmt"
	"go-drive/common/types"
	"go-drive/storage"
)

// AuthProviderDef is the definition of an auth provider, registered at startup.
type AuthProviderDef struct {
	Name        string
	DisplayName string
	Factory     func(config types.M, userDAO *storage.UserDAO, groupDAO *storage.GroupDAO) (AuthProvider, error)
}

var registeredProviderDefs = make(map[string]*AuthProviderDef)

// RegisterAuthProviderDef registers an auth provider definition.
// Called from init() of provider implementations (e.g. ldap).
func RegisterAuthProviderDef(def AuthProviderDef) {
	if _, exists := registeredProviderDefs[def.Name]; exists {
		panic(fmt.Sprintf("auth provider '%s' already registered", def.Name))
	}
	registeredProviderDefs[def.Name] = &def
}

// GetAuthProviderDef returns the provider definition by name.
func GetAuthProviderDef(name string) *AuthProviderDef {
	return registeredProviderDefs[name]
}

// GetAuthProviderDefs returns all registered provider definitions.
func GetAuthProviderDefs() []AuthProviderDef {
	defs := make([]AuthProviderDef, 0, len(registeredProviderDefs))
	for _, d := range registeredProviderDefs {
		defs = append(defs, *d)
	}
	return defs
}
