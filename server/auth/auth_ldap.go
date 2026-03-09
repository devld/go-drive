package auth

import (
	"crypto/tls"
	"encoding/json"
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

	"golang.org/x/crypto/bcrypt"
)

func init() {
	RegisterAuthProviderDef(AuthProviderDef{
		Name:        "ldap",
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
	GroupMapping  map[string]string
}

func parseLDAPConfig(c types.SM) (ldapConfig, error) {
	cfg := ldapConfig{
		UserFilter:    "(uid=%s)",
		UsernameAttr:  "uid",
		GroupFilter:   "(memberUid=%s)",
		GroupNameAttr: "cn",
	}
	if v := c["url"]; v != "" {
		cfg.URL = v
	}
	if v := c["start-tls"]; v == "true" || v == "1" {
		cfg.StartTLS = true
	}
	if v := c["skip-tls-verify"]; v == "true" || v == "1" {
		cfg.SkipTLSVerify = true
	}
	if v := c["bind-dn"]; v != "" {
		cfg.BindDN = v
	}
	if v := c["bind-password"]; v != "" {
		cfg.BindPassword = v
	}
	if v := c["base-dn"]; v != "" {
		cfg.BaseDN = v
	}
	if v := c["user-filter"]; v != "" {
		cfg.UserFilter = v
	}
	if v := c["username-attr"]; v != "" {
		cfg.UsernameAttr = v
	}
	if v := c["group-base-dn"]; v != "" {
		cfg.GroupBaseDN = v
	}
	if v := c["group-filter"]; v != "" {
		cfg.GroupFilter = v
	}
	if v := c["group-name-attr"]; v != "" {
		cfg.GroupNameAttr = v
	}
	if v := c["group-mapping"]; v != "" {
		if e := json.Unmarshal([]byte(v), &cfg.GroupMapping); e != nil {
			return cfg, fmt.Errorf("group-mapping: %w", e)
		}
	}
	if cfg.URL == "" || cfg.BindDN == "" || cfg.BindPassword == "" || cfg.BaseDN == "" {
		return cfg, fmt.Errorf("ldap config missing required: url, bind-dn, bind-password, base-dn")
	}
	return cfg, nil
}

type ldapAuthProvider struct {
	config  ldapConfig
	userDAO *storage.UserDAO
	groupDAO *storage.GroupDAO
}

func newLDAPAuthProvider(config types.SM, ch *registry.ComponentsHolder) (AuthProvider, error) {
	cfg, e := parseLDAPConfig(config)
	if e != nil {
		return nil, e
	}
	userDAO := ch.Get("userDAO").(*storage.UserDAO)
	groupDAO := ch.Get("groupDAO").(*storage.GroupDAO)
	return &ldapAuthProvider{config: cfg, userDAO: userDAO, groupDAO: groupDAO}, nil
}

func (p *ldapAuthProvider) EntryPoint(r *http.Request) (*AuthForm, error) {
	return nil, nil
}

func (p *ldapAuthProvider) Callback(r *http.Request, formData types.SM) (types.User, error) {
	username := formData["username"]
	password := formData["password"]
	if username == "" || password == "" {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

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

	if e := conn.Bind(userDN, password); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	if e := conn.Bind(p.config.BindDN, p.config.BindPassword); e != nil {
		return types.User{}, err.NewNotAllowedMessageError(i18n.T("api.auth.invalid_username_or_password"))
	}

	ldapGroups := p.fetchUserGroups(conn, username)
	goDriveGroups := p.mapGroups(ldapGroups)
	if e := p.ensureGroupsExist(goDriveGroups); e != nil {
		return types.User{}, e
	}

	user, e := p.userDAO.GetUser(username)
	if err.IsNotFoundError(e) {
		user, e = p.jitCreateUser(username, goDriveGroups)
		if e != nil {
			return types.User{}, e
		}
		return user, nil
	}
	if e != nil {
		return types.User{}, e
	}

	user.Groups = make([]types.Group, 0, len(goDriveGroups))
	for _, g := range goDriveGroups {
		user.Groups = append(user.Groups, types.Group{Name: g})
	}
	if e := p.userDAO.UpdateUser(username, user); e != nil {
		return types.User{}, e
	}
	return user, nil
}

func (p *ldapAuthProvider) connect() (*ldap.Conn, error) {
	u, e := url.Parse(p.config.URL)
	if e != nil {
		return nil, fmt.Errorf("ldap url: %w", e)
	}
	host := u.Host
	if host == "" {
		host = u.Path
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "ldap"
	}

	if scheme == "ldaps" {
		tlsCfg := &tls.Config{ServerName: u.Hostname()}
		if p.config.SkipTLSVerify {
			tlsCfg.InsecureSkipVerify = true
		}
		return ldap.DialTLS("tcp", host, tlsCfg)
	}

	conn, e := ldap.DialURL(p.config.URL)
	if e != nil {
		return nil, e
	}
	if p.config.StartTLS {
		tlsCfg := &tls.Config{ServerName: u.Hostname()}
		if p.config.SkipTLSVerify {
			tlsCfg.InsecureSkipVerify = true
		}
		if e := conn.StartTLS(tlsCfg); e != nil {
			conn.Close()
			return nil, e
		}
	}
	return conn, nil
}

func (p *ldapAuthProvider) fetchUserGroups(conn *ldap.Conn, username string) []string {
	if p.config.GroupBaseDN == "" {
		return nil
	}
	filter := strings.ReplaceAll(p.config.GroupFilter, "%s", ldap.EscapeFilter(username))
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
		if gd, ok := p.config.GroupMapping[lg]; ok && gd != "" && !seen[gd] {
			seen[gd] = true
			out = append(out, gd)
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
	groups := make([]types.Group, 0, len(groupNames))
	for _, n := range groupNames {
		groups = append(groups, types.Group{Name: n})
	}
	randomPass, e := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("ldap-%s-%d", username, 1)), bcrypt.DefaultCost)
	if e != nil {
		return types.User{}, e
	}
	user := types.User{
		Username: username,
		Password: string(randomPass),
		Groups:   groups,
	}
	return p.userDAO.AddUser(user)
}
