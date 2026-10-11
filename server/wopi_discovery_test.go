package server

import (
	"go-drive/common/types"
	"net/url"
	"strings"
	"testing"
)

func TestParseWOPIDiscoveryAndBuildActionURL(t *testing.T) {
	discoveryXML := []byte(`<?xml version="1.0" encoding="utf-8"?>
<wopi-discovery>
  <net-zone name="external-http">
    <app name="Writer" favIconUrl="https://office.test/favicon.ico">
      <action ext="docx" name="view" urlsrc="https://office.test/view?&lt;ui=UI_LLCC&amp;&gt;&amp;WOPISrc=WOPI_SOURCE&amp;" />
      <action ext="DOCX" name="edit" requires="locks,update" urlsrc="https://office.test/edit?&lt;ui=UI_LLCC&amp;&gt;&amp;WOPISrc=WOPI_SOURCE&amp;foo=bar" />
    </app>
  </net-zone>
</wopi-discovery>`)

	discovery, e := parseWOPIDiscovery(discoveryXML)
	if e != nil {
		t.Fatal(e)
	}
	action, ok := discovery.action(".DOCX", true)
	if !ok || action.Name != "edit" || action.Favicon != "https://office.test/favicon.ico" {
		t.Fatalf("unexpected action: %#v, ok=%v", action, ok)
	}
	wopiSrc := "https://drive.example/api/wopi/files/file-id"
	actionURL, e := makeWOPIActionURL(action.URLSrc, wopiSrc)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := url.Parse(actionURL)
	if e != nil {
		t.Fatal(e)
	}
	if got := parsed.Query().Get("WOPISrc"); got != wopiSrc {
		t.Fatalf("WOPISrc=%q, want %q", got, wopiSrc)
	}
	if got := parsed.Query().Get("foo"); got != "bar" {
		t.Fatalf("foo=%q, want bar", got)
	}
	if strings.Contains(actionURL, "UI_LLCC") || strings.Contains(actionURL, "WOPI_SOURCE") {
		t.Fatalf("placeholders were not removed: %s", actionURL)
	}

	config := discovery.sysConfig()
	extensions := config["extensions"].(types.M)
	docx := extensions["docx"].(types.M)
	if docx["view"] != true || docx["edit"] != true {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestParseWOPIDiscoveryRejectsEmptyActions(t *testing.T) {
	if _, e := parseWOPIDiscovery([]byte(`<wopi-discovery><net-zone /></wopi-discovery>`)); e == nil {
		t.Fatal("expected an error")
	}
}

func TestParseWOPIDiscoverySkipsUnsupportedRequirements(t *testing.T) {
	discovery, e := parseWOPIDiscovery([]byte(`<wopi-discovery><net-zone><app>` +
		`<action ext="docx" name="view" urlsrc="https://office.test/view" />` +
		`<action ext="docx" name="edit" requires="locks,update,rename" urlsrc="https://office.test/edit" />` +
		`</app></net-zone></wopi-discovery>`))
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := discovery.actions["docx"]["edit"]; ok {
		t.Fatal("action with unsupported requirements was retained")
	}
	if action, ok := discovery.action("docx", true); !ok || action.Name != "view" {
		t.Fatalf("view fallback missing: %#v, ok=%v", action, ok)
	}
}

func TestMakeWOPIActionURLRejectsNonHTTPURL(t *testing.T) {
	if _, e := makeWOPIActionURL("javascript:alert(1)", "https://drive.example/wopi/files/id"); e == nil {
		t.Fatal("expected an error")
	}
}
