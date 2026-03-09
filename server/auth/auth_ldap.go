package auth

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-ldap/ldap/v3"

	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/registry"
	"go-drive/common/types"
	"go-drive/storage"
)

const ldapProviderName = "ldap"

func init() {
	RegisterAuthProviderDef(AuthProviderDef{
		Name:        ldapProviderName,
		DisplayName: "LDAP",
		Factory:     newLDAPAuthProvider,
	})
}

type ldapConfig struct {
	URL           string
	StartTLS      bool
	SkipTLSVerify bool
	BindDN        string
	BindPassword  string
	BaseDN        string
	UserFilter    string
	UsernameAttr  string
	GroupBaseDN   string
	GroupFilter   string
	GroupNameAttr string
	// GroupMapping is the parsed mapping from an upstream (LDAP) group name to
	// the go-drive group names it grants. It is derived from the configured
	// go-drive -> upstream mapping (where each go-drive group lists its upstream
	// groups comma-separated), so several upstream groups can fold into the same
	// go-drive group.
	GroupMapping map[string][]string
}

func parseLDAPConfig(c types.M) (ldapConfig, error) {
	cfg := ldapConfig{
		URL:           c.GetStr("url", ""),
		StartTLS:      c.GetSV("start-tls", "").Bool(),
		SkipTLSVerify: c.GetSV("skip-tls-verify", "").Bool(),
		BindDN:        c.GetStr("bind-dn", ""),
		BindPassword:  c.GetStr("bind-password", ""),
		BaseDN:        c.GetStr("base-dn", ""),
		UserFilter:    c.GetStr("user-filter", "(uid=%s)"),
		UsernameAttr:  c.GetStr("username-attr", "uid"),
		GroupBaseDN:   c.GetStr("group-base-dn", ""),
		GroupFilter:   c.GetStr("group-filter", "(memberUid=%s)"),
		GroupNameAttr: c.GetStr("group-name-attr", "cn"),
		GroupMapping:  parseGroupMapping(c.GetSM("group-mapping")),
	}
	if cfg.URL == "" || cfg.BindDN == "" || cfg.BindPassword == "" || cfg.BaseDN == "" {
		return cfg, fmt.Errorf("ldap config missing required: url, bind-dn, bind-password, base-dn")
	}
	return cfg, nil
}

// parseGroupMapping inverts the configured go-drive -> upstream mapping (each
// value being a comma-separated list of upstream groups) into an
// upstream-group -> go-drive-groups lookup used at login time.
func parseGroupMapping(mapping types.SM) map[string][]string {
	rev := make(map[string][]string)
	for goDriveGroup, upstream := range mapping {
		if goDriveGroup == "" {
			continue
		}
		for name := range strings.SplitSeq(upstream, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			rev[name] = append(rev[name], goDriveGroup)
		}
	}
	return rev
}

type ldapAuthProvider struct {
	config   ldapConfig
	userDAO  *storage.UserDAO
	groupDAO *storage.GroupDAO
}

func newLDAPAuthProvider(config types.M, ch *registry.ComponentsHolder) (AuthProvider, error) {
	cfg, e := parseLDAPConfig(config)
	if e != nil {
		return nil, e
	}
	userDAO := ch.Get(registry.KeyUserDAO).(*storage.UserDAO)
	groupDAO := ch.Get(registry.KeyGroupDAO).(*storage.GroupDAO)
	return &ldapAuthProvider{
		config:   cfg,
		userDAO:  userDAO,
		groupDAO: groupDAO,
	}, nil
}

func (p *ldapAuthProvider) Callback(r *http.Request, formData types.SM) (types.User, error) {
	username := formData["username"]
	password := formData["password"]
	if username == "" || password == "" {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	// Each login opens its own short-lived connection and closes it on return;
	// there is no connection pool, so reusing this single connection for several
	// sequential binds (service account -> user -> service account) is safe and
	// supported by all common LDAP servers.
	conn, e := p.connect()
	if e != nil {
		return types.User{}, e
	}
	defer conn.Close()

	if e := conn.Bind(p.config.BindDN, p.config.BindPassword); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	filter := strings.ReplaceAll(p.config.UserFilter, "%s", ldap.EscapeFilter(username))
	sr, e := conn.Search(ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{p.config.UsernameAttr, "dn"},
		nil,
	))
	if e != nil {
		return types.User{}, e
	}
	if len(sr.Entries) == 0 {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}
	entry := sr.Entries[0]
	userDN := entry.DN
	uidValue := entry.GetAttributeValue(p.config.UsernameAttr)
	if uidValue == "" {
		uidValue = username
	}

	// Verify the user's password by binding as them, then rebind as the service
	// account so the connection can be reused for the group search below.
	if e := conn.Bind(userDN, password); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}
	if e := conn.Bind(p.config.BindDN, p.config.BindPassword); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	ldapGroups := p.fetchUserGroups(conn, uidValue, userDN)
	goDriveGroups := p.mapGroups(ldapGroups)
	if e := p.ensureGroupsExist(goDriveGroups); e != nil {
		return types.User{}, e
	}

	user, e := p.userDAO.GetUser(username)
	if err.IsNotFoundError(e) {
		return p.jitCreateUser(username, goDriveGroups)
	}
	if e != nil {
		return types.User{}, e
	}

	// A user with this name already exists locally but is not an LDAP user.
	// Do not let LDAP take over a local (or other-provider) account.
	if user.Source != ldapProviderName {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	// Existing LDAP user: when group sync is enabled, overwrite the local groups
	// with the ones resolved from LDAP on every successful login, so groups
	// removed remotely are also removed locally.
	// Never touch the password (passing an empty password keeps the stored hash;
	// passing the existing hash would cause it to be bcrypt-hashed again).
	if p.config.GroupBaseDN != "" {
		user.Password = ""
		user.Groups = toGroups(goDriveGroups)
		if e := p.userDAO.UpdateUser(username, user); e != nil {
			return types.User{}, e
		}
	}
	return user, nil
}

func toGroups(names []string) []types.Group {
	groups := make([]types.Group, 0, len(names))
	for _, n := range names {
		groups = append(groups, types.Group{Name: n})
	}
	return groups
}

func (p *ldapAuthProvider) connect() (*ldap.Conn, error) {
	u, e := url.Parse(p.config.URL)
	if e != nil {
		return nil, fmt.Errorf("ldap url: %w", e)
	}

	var tlsCfg *tls.Config
	if u.Scheme == "ldaps" || p.config.StartTLS {
		tlsCfg = &tls.Config{ServerName: u.Hostname()}
		if p.config.SkipTLSVerify {
			tlsCfg.InsecureSkipVerify = true
		}
	}

	// ldap.DialTLS is deprecated; DialURL handles both ldap:// and ldaps://
	// (TLS is negotiated automatically for the ldaps scheme) and the TLS config
	// is supplied via the DialWithTLSConfig option.
	var opts []ldap.DialOpt
	if u.Scheme == "ldaps" {
		opts = append(opts, ldap.DialWithTLSConfig(tlsCfg))
	}
	conn, e := ldap.DialURL(p.config.URL, opts...)
	if e != nil {
		return nil, e
	}
	if u.Scheme != "ldaps" && p.config.StartTLS {
		if e := conn.StartTLS(tlsCfg); e != nil {
			conn.Close()
			return nil, e
		}
	}
	return conn, nil
}

// fetchUserGroups resolves the groups a user belongs to.
// The group-filter supports two placeholders so it works across directory styles:
//   - %s : the user's username/uid attribute value (e.g. posixGroup memberUid)
//   - %d : the user's full DN (e.g. groupOfNames member / AD member)
func (p *ldapAuthProvider) fetchUserGroups(conn *ldap.Conn, uid, userDN string) []string {
	if p.config.GroupBaseDN == "" {
		return nil
	}
	filter := p.config.GroupFilter
	filter = strings.ReplaceAll(filter, "%s", ldap.EscapeFilter(uid))
	filter = strings.ReplaceAll(filter, "%d", ldap.EscapeFilter(userDN))
	sr, e := conn.Search(ldap.NewSearchRequest(
		p.config.GroupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{p.config.GroupNameAttr},
		nil,
	))
	if e != nil {
		return nil
	}
	names := make([]string, 0, len(sr.Entries))
	for _, e := range sr.Entries {
		if v := e.GetAttributeValue(p.config.GroupNameAttr); v != "" {
			names = append(names, v)
		}
	}
	return names
}

func (p *ldapAuthProvider) mapGroups(ldapGroups []string) []string {
	if len(p.config.GroupMapping) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, lg := range ldapGroups {
		for _, gd := range p.config.GroupMapping[lg] {
			if !seen[gd] {
				seen[gd] = true
				out = append(out, gd)
			}
		}
	}
	return out
}

func (p *ldapAuthProvider) ensureGroupsExist(names []string) error {
	existing, e := p.groupDAO.ListGroup()
	if e != nil {
		return e
	}
	set := make(map[string]bool)
	for _, g := range existing {
		set[g.Name] = true
	}
	for _, name := range names {
		if set[name] {
			continue
		}
		_, e := p.groupDAO.AddGroup(storage.GroupWithUsers{Group: types.Group{Name: name}})
		if e != nil && !err.IsNotAllowedError(e) {
			return e
		}
		set[name] = true
	}
	return nil
}

func (p *ldapAuthProvider) jitCreateUser(username string, groupNames []string) (types.User, error) {
	// The local password is never used for LDAP users (auth is delegated to LDAP),
	// but we still store an unguessable random value so the account can never be
	// logged into via the local provider.
	randomPass := make([]byte, 32)
	if _, e := rand.Read(randomPass); e != nil {
		return types.User{}, e
	}
	user := types.User{
		Username: username,
		Password: base64.RawStdEncoding.EncodeToString(randomPass),
		Source:   ldapProviderName,
		Groups:   toGroups(groupNames),
	}
	return p.userDAO.AddUser(user)
}
