package driveutil

import "testing"

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
