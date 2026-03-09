package auth

import (
	"fmt"
	"net/http"

	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
)

const (
	AuthTypeForm     = "form"
	AuthTypeRedirect = "redirect"
)

// AuthForm describes the entry point of an auth provider.
type AuthForm struct {
	Provider    string           `json:"provider"`
	DisplayName string           `json:"displayName"`
	Type        string           `json:"type"`
	Form        []types.FormItem `json:"form,omitempty"`
	RedirectURL string           `json:"redirectUrl,omitempty"`
}

// AuthProvider is the core authentication abstraction.
// Name is specified at registration time, not on the interface.
type AuthProvider interface {
	// EntryPoint returns the entry point for this auth method:
	//   form-type: returns form fields for the client to render
	//   redirect-type: returns a redirect URL
	//   nil: transparent provider, no UI entry (e.g. LDAP piggybacks on password form)
	EntryPoint(r *http.Request) (*AuthForm, error)

	// Callback processes the authentication response.
	// formData contains submitted form values or redirect callback parameters.
	Callback(r *http.Request, formData types.SM) (types.User, error)
}

type namedProvider struct {
	name     string
	provider AuthProvider
}

// UserAuth orchestrates multiple auth providers.
type UserAuth struct {
	providers []namedProvider
}

// NewUserAuth builds the provider chain.
// Local provider is always first (implicit, no config needed).
// Additional providers are created from config using the factory registry.
func NewUserAuth(configs []common.AuthProviderConfig, ch *registry.ComponentsHolder) (*UserAuth, error) {
	local := newLocalAuthProvider(ch)
	providers := []namedProvider{{name: "local", provider: local}}

	for _, cfg := range configs {
		def := GetAuthProviderDef(cfg.Type)
		if def == nil {
			return nil, fmt.Errorf("unknown auth provider type: %s", cfg.Type)
		}
		p, e := def.Factory(cfg.Config, ch)
		if e != nil {
			return nil, fmt.Errorf("init auth provider %q: %w", cfg.Type, e)
		}
		providers = append(providers, namedProvider{name: cfg.Type, provider: p})
	}
	return &UserAuth{providers: providers}, nil
}

// GetForms returns all non-nil EntryPoints for the frontend.
func (ua *UserAuth) GetForms(r *http.Request) []AuthForm {
	var out []AuthForm
	for _, np := range ua.providers {
		form, e := np.provider.EntryPoint(r)
		if e != nil || form == nil {
			continue
		}
		form.Provider = np.name
		out = append(out, *form)
	}
	return out
}

// AuthenticateForm tries all form-type providers in order (for POST /auth/login).
func (ua *UserAuth) AuthenticateForm(r *http.Request, formData types.SM) (types.User, error) {
	for _, np := range ua.providers {
		user, e := np.provider.Callback(r, formData)
		if e == nil {
			return user, nil
		}
		if err.IsNotAllowedError(e) || err.IsNotFoundError(e) {
			continue
		}
		return types.User{}, e
	}
	return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
}

// AuthenticateCallback dispatches to a named provider (for GET /auth/callback/:provider).
func (ua *UserAuth) AuthenticateCallback(provider string, r *http.Request, formData types.SM) (types.User, error) {
	for _, np := range ua.providers {
		if np.name != provider {
			continue
		}
		return np.provider.Callback(r, formData)
	}
	return types.User{}, err.NewNotFoundMessageError(i18n.T("api.auth.provider_not_found", provider))
}

// AuthByUsernamePassword is for backward compat (BasicAuth/WebDAV).
func (ua *UserAuth) AuthByUsernamePassword(username, password string) (types.User, error) {
	formData := types.SM{"username": username, "password": password}
	return ua.AuthenticateForm(nil, formData)
}
