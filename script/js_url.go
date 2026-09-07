package script

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var jsURLUtils = map[string]any{
	"parse":             jsURLParse,
	"build":             jsURLBuild,
	"parseSearchParams": jsURLParseSearchParams,
	"buildSearchParams": jsURLBuildSearchParams,
}

var jsURLParse = NativeFunction(func(vm *VM, args Values) any {
	raw := requireURLString(vm, args.Get(0), "urlUtils.parse")
	parsed, e := parseURL(raw)
	if e != nil {
		vm.ThrowTypeError(fmt.Sprintf("urlUtils.parse: %v", e))
	}
	return newURLPartsValue(vm, parsed)
})

var jsURLBuild = NativeFunction(func(vm *VM, args Values) any {
	parts := args.Get(0)
	if parts == nil || parts.IsNil() || !parts.IsObject() || parts.IsArray() {
		vm.ThrowTypeError("urlUtils.build requires a URL parts object")
	}
	built, e := buildURL(vm, parts)
	if e != nil {
		vm.ThrowTypeError(fmt.Sprintf("urlUtils.build: %v", e))
	}
	return built
})

var jsURLParseSearchParams = NativeFunction(func(vm *VM, args Values) any {
	raw := requireURLString(vm, args.Get(0), "urlUtils.parseSearchParams")
	values, e := parseURLSearchParams(raw)
	if e != nil {
		vm.ThrowTypeError(fmt.Sprintf("urlUtils.parseSearchParams: %v", e))
	}
	return newURLSearchParamsValue(vm, values)
})

var jsURLBuildSearchParams = NativeFunction(func(vm *VM, args Values) any {
	values, e := parseURLSearchParamsObject(args.Get(0))
	if e != nil {
		vm.ThrowTypeError(fmt.Sprintf("urlUtils.buildSearchParams: %v", e))
	}
	return buildURLSearchParams(values)
})

// parsedURL is the read-only JS view of a URL. Copy before editing for build.
type parsedURL struct {
	origin       string
	protocol     string
	username     string
	password     string
	host         string
	hostname     string
	port         string
	pathname     string
	searchParams url.Values
	hash         string
}

func requireURLString(vm *VM, value *Value, name string) string {
	if value == nil || value.IsNil() || !value.IsString() {
		vm.ThrowTypeError(name + " requires a string")
	}
	return value.String()
}

func parseURL(raw string) (parsedURL, error) {
	u, e := url.Parse(raw)
	if e != nil {
		return parsedURL{}, e
	}

	searchParams, e := parseURLSearchParams(u.RawQuery)
	if e != nil {
		return parsedURL{}, fmt.Errorf("invalid query: %w", e)
	}

	protocol := ""
	if u.Scheme != "" {
		protocol = u.Scheme + ":"
	}

	origin := ""
	if u.Scheme != "" && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}

	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		if value, ok := u.User.Password(); ok {
			password = value
		}
	}

	pathname := u.EscapedPath()
	if u.Opaque != "" {
		pathname = u.Opaque
	}

	hash := ""
	if u.Fragment != "" || u.RawFragment != "" {
		hash = "#" + u.EscapedFragment()
	}

	return parsedURL{
		origin:       origin,
		protocol:     protocol,
		username:     username,
		password:     password,
		host:         u.Host,
		hostname:     u.Hostname(),
		port:         u.Port(),
		pathname:     pathname,
		searchParams: searchParams,
		hash:         hash,
	}, nil
}

func newURLPartsValue(vm *VM, value parsedURL) *Value {
	return vm.ToJSValue(map[string]any{
		"origin":       value.origin,
		"protocol":     value.protocol,
		"username":     value.username,
		"password":     value.password,
		"host":         value.host,
		"hostname":     value.hostname,
		"port":         value.port,
		"pathname":     value.pathname,
		"searchParams": newURLSearchParamsValue(vm, value.searchParams),
		"hash":         value.hash,
	})
}

func newURLSearchParamsValue(vm *VM, values url.Values) *Value {
	return vm.ToJSValue(map[string][]string(values))
}

func buildURL(vm *VM, parts *Value) (string, error) {
	origin, hasOrigin := urlPartString(vm, parts, "origin")
	protocol, hasProtocol := urlPartString(vm, parts, "protocol")
	username, hasUsername := urlPartString(vm, parts, "username")
	password, hasPassword := urlPartString(vm, parts, "password")
	host, hasHost := urlPartString(vm, parts, "host")
	hostname, hasHostname := urlPartString(vm, parts, "hostname")
	port, hasPort := urlPartString(vm, parts, "port")
	pathname, hasPathname := urlPartString(vm, parts, "pathname")
	hash, hasHash := urlPartString(vm, parts, "hash")

	scheme := strings.TrimSuffix(protocol, ":")
	if hasProtocol {
		if protocol != "" && !strings.HasSuffix(protocol, ":") {
			return "", fmt.Errorf("protocol must end with ':'")
		}
		if protocol != "" && scheme == "" {
			return "", fmt.Errorf("protocol must not be ':'")
		}
	}
	if scheme != "" {
		if _, e := url.Parse(scheme + ":"); e != nil {
			return "", fmt.Errorf("invalid protocol: %w", e)
		}
	}

	if hasOrigin && origin != "" {
		originURL, e := parseURLOrigin(origin)
		if e != nil {
			return "", e
		}
		if !hasProtocol || protocol == "" {
			scheme = originURL.Scheme
			hasProtocol = true
		} else if !strings.EqualFold(scheme, originURL.Scheme) {
			return "", fmt.Errorf("origin and protocol do not match")
		}
		if !hasHost || host == "" {
			host = originURL.Host
			hasHost = true
		}
	}

	host, e := buildURLHost(host, hostname, port, hasHost, hasHostname, hasPort)
	if e != nil {
		return "", e
	}
	if (hasUsername && username != "" || hasPassword && password != "") && host == "" {
		return "", fmt.Errorf("username/password require a host")
	}

	u := url.URL{Scheme: scheme, Host: host}
	if username != "" || password != "" {
		if password != "" {
			u.User = url.UserPassword(username, password)
		} else {
			u.User = url.User(username)
		}
	}

	if hasPathname {
		if e := setURLPath(&u, pathname); e != nil {
			return "", e
		}
	}

	searchParams := parts.Get("searchParams")
	if searchParams != nil && !searchParams.IsUndefined() {
		values, e := parseURLSearchParamsObject(searchParams)
		if e != nil {
			return "", e
		}
		u.RawQuery = values.Encode()
	}

	if hasHash {
		if e := setURLFragment(&u, hash); e != nil {
			return "", e
		}
	}

	built := u.String()
	if _, e := url.Parse(built); e != nil {
		return "", fmt.Errorf("invalid URL: %w", e)
	}
	return built, nil
}

func urlPartString(vm *VM, parts *Value, name string) (string, bool) {
	value := parts.Get(name)
	if value == nil || value.IsUndefined() {
		return "", false
	}
	if !value.IsString() {
		vm.ThrowTypeError(fmt.Sprintf("urlUtils.build: %s must be a string", name))
	}
	return value.String(), true
}

func parseURLOrigin(origin string) (*url.URL, error) {
	u, e := url.Parse(origin)
	if e != nil {
		return nil, fmt.Errorf("invalid origin: %w", e)
	}
	if u.Scheme == "" || u.Host == "" || u.User != nil || u.Opaque != "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("origin must contain only scheme and host")
	}
	return u, nil
}

func buildURLHost(host, hostname, port string, hasHost, hasHostname, hasPort bool) (string, error) {
	if hasHost && host != "" {
		parsed, e := validateURLHost(host)
		if e != nil {
			return "", e
		}
		if hasHostname && parsed.Hostname() != hostname {
			return "", fmt.Errorf("host and hostname do not match")
		}
		if hasPort && parsed.Port() != port {
			return "", fmt.Errorf("host and port do not match")
		}
		return host, nil
	}

	if hasHostname || hasPort {
		if !hasHostname {
			return "", fmt.Errorf("port requires hostname")
		}
		return buildURLHostFromParts(hostname, port)
	}
	return host, nil
}

func buildURLHostFromParts(hostname, port string) (string, error) {
	if hostname == "" {
		if port != "" {
			return "", fmt.Errorf("port requires hostname")
		}
		return "", nil
	}
	if strings.ContainsAny(hostname, "/?#@[]") {
		return "", fmt.Errorf("invalid hostname")
	}
	if port != "" {
		for _, r := range port {
			if r < '0' || r > '9' {
				return "", fmt.Errorf("invalid port")
			}
		}
	}

	host := hostname
	if strings.Contains(hostname, ":") {
		host = strings.TrimSuffix(net.JoinHostPort(hostname, port), ":")
	} else if port != "" {
		host = net.JoinHostPort(hostname, port)
	}
	if _, e := validateURLHost(host); e != nil {
		return "", e
	}
	return host, nil
}

func validateURLHost(host string) (*url.URL, error) {
	if host == "" {
		return nil, nil
	}
	u, e := url.Parse("//" + host)
	if e != nil {
		return nil, fmt.Errorf("invalid host: %w", e)
	}
	if u.Host != host || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid host")
	}
	return u, nil
}

func setURLPath(u *url.URL, pathname string) error {
	if pathname == "" {
		return nil
	}
	if strings.ContainsAny(pathname, "?#") {
		return fmt.Errorf("pathname must not contain raw '?' or '#'")
	}
	if u.Host != "" && !strings.HasPrefix(pathname, "/") {
		return fmt.Errorf("pathname with a host must start with '/'")
	}
	if u.Scheme != "" && u.Host == "" && !strings.HasPrefix(pathname, "/") {
		u.Opaque = pathname
		return nil
	}

	decoded, e := url.PathUnescape(pathname)
	if e != nil {
		return fmt.Errorf("invalid pathname: %w", e)
	}
	u.Path = decoded
	if u.EscapedPath() != pathname {
		u.RawPath = pathname
	}
	return nil
}

func setURLFragment(u *url.URL, hash string) error {
	if hash == "" {
		return nil
	}
	if !strings.HasPrefix(hash, "#") {
		return fmt.Errorf("hash must be empty or start with '#'")
	}
	raw := strings.TrimPrefix(hash, "#")
	if raw == "" {
		return nil
	}
	decoded, e := url.PathUnescape(raw)
	if e != nil {
		return fmt.Errorf("invalid hash: %w", e)
	}
	u.Fragment = decoded
	if u.EscapedFragment() != raw {
		u.RawFragment = raw
	}
	return nil
}

func parseURLSearchParams(raw string) (url.Values, error) {
	return url.ParseQuery(strings.TrimPrefix(raw, "?"))
}

func buildURLSearchParams(values url.Values) string {
	encoded := values.Encode()
	if encoded == "" {
		return ""
	}
	return "?" + encoded
}

func parseURLSearchParamsObject(value *Value) (url.Values, error) {
	if value == nil || value.IsNil() || !value.IsObject() || value.IsArray() {
		return nil, fmt.Errorf("searchParams must be an object of string arrays")
	}

	params := make(url.Values)
	for _, key := range value.Keys() {
		item := value.Get(key)
		if item == nil || item.IsNil() {
			return nil, fmt.Errorf("searchParams[%q] must be a string array", key)
		}
		if !item.IsObject() || !item.IsArray() {
			return nil, fmt.Errorf("searchParams[%q] must be a string array", key)
		}
		array := item.Array()
		values := make([]string, len(array))
		for i, element := range array {
			if element == nil || element.IsNil() || !element.IsString() {
				return nil, fmt.Errorf("searchParams[%q][%d] must be a string", key, i)
			}
			values[i] = element.String()
		}
		params[key] = values
	}
	return params, nil
}
