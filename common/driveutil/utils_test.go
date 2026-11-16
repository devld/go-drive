package driveutil

import (
	"net/http"
	"strings"
	"testing"
)

func TestExtensionByMimeTypePrefersInlineFileExts(t *testing.T) {
	if got := ExtensionByMimeType("image/jpeg"); got != ".jpg" {
		t.Fatalf("image/jpeg = %q, want .jpg", got)
	}
	if got := ExtensionByMimeType("video/mp4"); got != ".mp4" {
		t.Fatalf("video/mp4 = %q, want .mp4", got)
	}
	if got := ExtensionByMimeType("image/svg+xml"); got != ".svg" {
		t.Fatalf("image/svg+xml = %q, want stdlib fallback .svg", got)
	}
	if got := ExtensionByMimeType(""); got != "" {
		t.Fatalf("empty mime = %q, want empty", got)
	}
	if got := ExtensionByMimeType("application/x-unknown-type"); got != "" {
		t.Fatalf("unknown mime = %q, want empty", got)
	}
}

// xml must never be served inline: application/xml is rendered as XHTML when
// the root element declares the XHTML namespace, which executes embedded
// script in the application origin.
func TestSetContentDispositionHeaderXMLIsAttachment(t *testing.T) {
	dispositions := []string{
		"evil.xml",
		"evil.XML",
		"evil.Xml",
		"evil.svg",
		"evil.SVG",
		"evil.html",
		"evil.htm",
		"evil.xhtml",
	}
	for _, name := range dispositions {
		h := http.Header{}
		setContentDispositionHeaderIfNeeded(h, name)
		cd := h.Get("Content-Disposition")
		if !strings.HasPrefix(cd, "attachment") {
			t.Errorf("%s: Content-Disposition = %q, want attachment", name, cd)
		}
		if ct := h.Get("Content-Type"); ct != "" {
			t.Errorf("%s: Content-Type = %q, want empty", name, ct)
		}
	}
}

// Types that are genuinely safe to show inline keep their inline behaviour.
func TestSetContentDispositionHeaderInlineAllowlist(t *testing.T) {
	cases := map[string]string{
		"a.jpg": "image/jpeg",
		"a.png": "image/png",
		"a.mp4": "video/mp4",
		"a.mp3": "audio/mpeg",
		"a.txt": "text/plain",
		"a.TXT": "text/plain",
	}
	for name, wantType := range cases {
		h := http.Header{}
		setContentDispositionHeaderIfNeeded(h, name)
		if ct := h.Get("Content-Type"); ct != wantType {
			t.Errorf("%s: Content-Type = %q, want %q", name, ct, wantType)
		}
		if cd := h.Get("Content-Disposition"); cd != "" {
			t.Errorf("%s: Content-Disposition = %q, want empty", name, cd)
		}
	}
}

func TestSetDownloadSecurityHeaders(t *testing.T) {
	h := http.Header{}
	SetDownloadSecurityHeaders(h)
	if got := h.Get("Content-Security-Policy"); got != "sandbox" {
		t.Errorf("Content-Security-Policy = %q, want sandbox", got)
	}
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}
