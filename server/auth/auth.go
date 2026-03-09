package auth

import (
	"fmt"
	"net/http"

	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"
)

const (
	AuthTypeForm     = "form"
	AuthTypeRedirect = "redirect"

	// ProviderIdentity is the built-in username/password login method.
	// Its callback runs the local-table-first authentication flow (local user
	// or external backend such as LDAP, routed by the user's source).
	ProviderIdentity = "identity"
)

// AuthForm describes a login method exposed to the client.
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
	// Callback processes the authentication response and resolves a user.
	// formData contains submitted form values or callback parameters. The
	// request is provided because some methods need request context to complete
	// (e.g. an OAuth provider reading the state cookie set during Start, or a
	// passkey provider verifying the assertion origin).
	Callback(r *http.Request, formData types.SM) (types.User, error)
}

// AuthEntryPoint is an optional capability for login methods that expose a UI
// entry to be rendered by the client (listed via /config). Providers that are
// transparent and do not appear as a separate method (e.g. LDAP, which
// piggybacks on the identity password form) simply do not implement it.
type AuthEntryPoint interface {
	EntryPoint() (*AuthForm, error)
}

// AuthStarter is an optional capability for login methods that need a
// server-side "begin" step before the callback, e.g. building an OAuth
// authorize URL or generating a WebAuthn/passkey challenge. It is the
// extension point behind POST /auth/:provider/start.
type AuthStarter interface {
	Start(r *http.Request, formData types.SM) (any, error)
}

type namedProvider struct {
	name     string
	provider AuthProvider
}

// UserAuth orchestrates the local provider and any configured external providers.
type UserAuth struct {
	userDAO   *storage.UserDAO
	local     AuthProvider
	providers []namedProvider // external providers, in config order
}

// NewUserAuth builds the provider chain.
// The local provider is always implicit (no config needed).
// Additional providers are created from config using the factory registry.
func NewUserAuth(configs []common.AuthProviderConfig, ch *registry.ComponentsHolder) (*UserAuth, error) {
	userDAO := ch.Get(registry.KeyUserDAO).(*storage.UserDAO)
	local := newLocalAuthProvider(ch)

	providers := make([]namedProvider, 0, len(configs))
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
	return &UserAuth{userDAO: userDAO, local: local, providers: providers}, nil
}

func (ua *UserAuth) findProvider(name string) AuthProvider {
	for _, np := range ua.providers {
		if np.name == name {
			return np.provider
		}
	}
	return nil
}

// SysConfig exposes the supported login methods to the client via /config.
func (ua *UserAuth) SysConfig() (string, types.M, error) {
	return "auth", types.M{
		"providers": ua.GetForms(),
	}, nil
}

// GetForms returns the descriptors of all login methods that have a UI entry.
// The built-in identity (username/password) method is always first; transparent
// providers (those not implementing AuthEntryPoint, e.g. LDAP) are not listed.
func (ua *UserAuth) GetForms() []AuthForm {
	out := make([]AuthForm, 0, 1+len(ua.providers))
	out = appendForm(out, ProviderIdentity, ua.local)
	for _, np := range ua.providers {
		out = appendForm(out, np.name, np.provider)
	}
	return out
}

// appendForm appends the provider's UI descriptor when it exposes one.
func appendForm(out []AuthForm, name string, p AuthProvider) []AuthForm {
	ep, ok := p.(AuthEntryPoint)
	if !ok {
		return out
	}
	form, e := ep.EntryPoint()
	if e != nil || form == nil {
		return out
	}
	form.Provider = name
	return append(out, *form)
}

// resolveProvider looks up a login method by name, including the built-in identity.
func (ua *UserAuth) resolveProvider(name string) AuthProvider {
	if name == ProviderIdentity {
		return ua.local
	}
	return ua.findProvider(name)
}

// Start runs the optional begin step of a login method (extension point behind
// POST /auth/:provider/start), e.g. an OAuth authorize URL or passkey challenge.
func (ua *UserAuth) Start(provider string, r *http.Request, formData types.SM) (any, error) {
	p := ua.resolveProvider(provider)
	if p == nil {
		return nil, err.NewNotFoundMessageError(i18n.T("api.auth.provider_not_found", provider))
	}
	starter, ok := p.(AuthStarter)
	if !ok {
		return nil, err.NewNotAllowedMessageError(i18n.T("api.auth.start_not_supported", provider))
	}
	return starter.Start(r, formData)
}

// AuthenticateCallback completes a login for the given provider (POST
// /auth/:provider/callback):
//   - The built-in identity provider runs the username/password flow against
//     the local user table (see authenticateIdentity).
//   - Any other provider is dispatched to its own Callback (extension point,
//     e.g. OAuth/passkey).
func (ua *UserAuth) AuthenticateCallback(provider string, r *http.Request, formData types.SM) (types.User, error) {
	if provider == ProviderIdentity {
		return ua.authenticateIdentity(r, formData)
	}
	p := ua.findProvider(provider)
	if p == nil {
		return types.User{}, err.NewNotFoundMessageError(i18n.T("api.auth.provider_not_found", provider))
	}
	return p.Callback(r, formData)
}

// authenticateIdentity authenticates a username/password submission. It backs
// the identity provider callback and BasicAuth/WebDAV.
//
// The local user table is the source of truth (usernames are matched exactly,
// case-sensitively):
//   - First look up the user in the local table.
//   - If the user exists and is owned by an external provider, authenticate
//     against that provider first and fall back to local auth if it fails
//     (e.g. the provider is unreachable). Local users go straight to local auth.
//   - If the user does not exist, try the external providers in order so they
//     (e.g. LDAP) can just-in-time provision a new user.
func (ua *UserAuth) authenticateIdentity(r *http.Request, formData types.SM) (types.User, error) {
	invalid := err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))

	username := formData["username"]
	if username == "" {
		return types.User{}, invalid
	}

	existing, e := ua.userDAO.GetUser(username)
	if e != nil && !err.IsNotFoundError(e) {
		return types.User{}, e
	}

	if e == nil {
		if existing.Source != "" {
			if provider := ua.findProvider(existing.Source); provider != nil {
				if user, ce := provider.Callback(r, formData); ce == nil {
					return user, nil
				}
			}
		}
		return ua.local.Callback(r, formData)
	}

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
	return types.User{}, invalid
}

// AuthByUsernamePassword authenticates with a username/password pair, used by
// BasicAuth/WebDAV.
func (ua *UserAuth) AuthByUsernamePassword(username, password string) (types.User, error) {
	return ua.authenticateIdentity(nil, types.SM{"username": username, "password": password})
}
