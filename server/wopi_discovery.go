package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"go-drive/common/types"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	wopiDiscoveryMaxSize = 2 << 20
	wopiDiscoveryTTL     = 12 * time.Hour
)

var wopiActionPlaceholderPattern = regexp.MustCompile(`<[^>]*>`)

type wopiDiscoveryDocument struct {
	NetZones []wopiDiscoveryNetZone `xml:"net-zone"`
}

type wopiDiscoveryNetZone struct {
	Apps []wopiDiscoveryApp `xml:"app"`
}

type wopiDiscoveryApp struct {
	Name       string                `xml:"name,attr"`
	FaviconURL string                `xml:"favIconUrl,attr"`
	Actions    []wopiDiscoveryAction `xml:"action"`
}

type wopiDiscoveryAction struct {
	Name     string `xml:"name,attr"`
	Ext      string `xml:"ext,attr"`
	URLSrc   string `xml:"urlsrc,attr"`
	Requires string `xml:"requires,attr"`
	Favicon  string `xml:"-"`
}

type wopiDiscovery struct {
	actions map[string]map[string]wopiDiscoveryAction
	loaded  time.Time
}

type wopiDiscoveryClient struct {
	url    string
	client *http.Client

	mu    sync.Mutex
	cache *wopiDiscovery
}

func newWOPIDiscoveryClient(discoveryURL string) (*wopiDiscoveryClient, error) {
	u, e := url.Parse(discoveryURL)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("invalid WOPI discovery URL %q", discoveryURL)
	}
	c := &wopiDiscoveryClient{
		url: discoveryURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many WOPI discovery redirects")
				}
				return nil
			},
		},
	}
	if _, e := c.get(context.Background()); e != nil {
		return nil, e
	}
	return c, nil
}

func (c *wopiDiscoveryClient) get(ctx context.Context) (*wopiDiscovery, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache != nil && time.Since(c.cache.loaded) < wopiDiscoveryTTL {
		return c.cache, nil
	}

	req, e := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if e != nil {
		return nil, e
	}
	resp, e := c.client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("load WOPI discovery: %w", e)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load WOPI discovery: unexpected status %s", resp.Status)
	}

	limited := io.LimitReader(resp.Body, wopiDiscoveryMaxSize+1)
	data, e := io.ReadAll(limited)
	if e != nil {
		return nil, fmt.Errorf("read WOPI discovery: %w", e)
	}
	if len(data) > wopiDiscoveryMaxSize {
		return nil, fmt.Errorf("WOPI discovery exceeds %d bytes", wopiDiscoveryMaxSize)
	}
	parsed, e := parseWOPIDiscovery(data)
	if e != nil {
		return nil, e
	}
	c.cache = parsed
	return parsed, nil
}

func parseWOPIDiscovery(data []byte) (*wopiDiscovery, error) {
	var document wopiDiscoveryDocument
	if e := xml.Unmarshal(data, &document); e != nil {
		return nil, fmt.Errorf("parse WOPI discovery: %w", e)
	}
	actions := make(map[string]map[string]wopiDiscoveryAction)
	for _, zone := range document.NetZones {
		for _, app := range zone.Apps {
			for _, action := range app.Actions {
				ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(action.Ext), "."))
				name := strings.ToLower(strings.TrimSpace(action.Name))
				if ext == "" || (name != "view" && name != "edit") || action.URLSrc == "" ||
					!wopiRequirementsSupported(action.Requires) {
					continue
				}
				if _, ok := actions[ext]; !ok {
					actions[ext] = make(map[string]wopiDiscoveryAction)
				}
				if _, exists := actions[ext][name]; exists {
					continue
				}
				action.Ext = ext
				action.Name = name
				action.Favicon = app.FaviconURL
				actions[ext][name] = action
			}
		}
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("WOPI discovery contains no view or edit actions")
	}
	return &wopiDiscovery{actions: actions, loaded: time.Now()}, nil
}

func wopiRequirementsSupported(raw string) bool {
	for _, requirement := range strings.FieldsFunc(strings.ToLower(raw), func(ch rune) bool {
		return ch == ',' || ch == ' ' || ch == ';'
	}) {
		if requirement != "locks" && requirement != "update" {
			return false
		}
	}
	return true
}

func (d *wopiDiscovery) action(ext string, writable bool) (wopiDiscoveryAction, bool) {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	byName := d.actions[ext]
	if writable {
		if action, ok := byName["edit"]; ok {
			return action, true
		}
	}
	action, ok := byName["view"]
	return action, ok
}

func (d *wopiDiscovery) sysConfig() types.M {
	extensions := make(types.M, len(d.actions))
	for ext, actions := range d.actions {
		extensions[ext] = types.M{
			"view": actions["view"].URLSrc != "",
			"edit": actions["edit"].URLSrc != "",
		}
	}
	return types.M{"enabled": true, "extensions": extensions}
}

func makeWOPIActionURL(urlSrc, wopiSrc string) (string, error) {
	cleaned := wopiActionPlaceholderPattern.ReplaceAllString(urlSrc, "")
	u, e := url.Parse(cleaned)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid WOPI action URL")
	}
	query := u.Query()
	query.Set("WOPISrc", wopiSrc)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
