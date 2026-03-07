package server

import (
	"net/http/httptest"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"go-drive/common"
	"github.com/gin-gonic/gin"
)

// fileBucketRouteForTest returns a minimal fileBucketRoute for testing pure logic (e.g. checkAllowedTypes, checkReferrers).
func fileBucketRouteForTest(config common.Config) *fileBucketRoute {
	return &fileBucketRoute{config: config}
}

func Test_fileBucketRoute_checkAllowedTypes(t *testing.T) {
	fr := fileBucketRouteForTest(common.Config{})

	tests := []struct {
		name        string
		mimeType    string
		fileExt     string
		allowedTypes string
		want        bool
	}{
		{"empty allowed", "image/png", ".png", "", true},
		{"exact mime match", "image/png", ".png", "image/png", true},
		{"exact ext match", "image/png", ".png", ".png", true},
		{"wildcard type match", "image/png", ".png", "image/*", true},
		{"wildcard type no match", "text/plain", ".txt", "image/*", false},
		{"spaces in list", "image/png", ".png", " image/png , .png ", true},
		{"no match", "image/png", ".png", "image/jpeg", false},
		{"another mime in list", "text/plain", ".txt", "image/*,text/plain", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fr.checkAllowedTypes(tt.mimeType, tt.fileExt, tt.allowedTypes)
			if got != tt.want {
				t.Errorf("checkAllowedTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fileBucketRoute_checkReferrers(t *testing.T) {
	fr := fileBucketRouteForTest(common.Config{})

	tests := []struct {
		name             string
		referrer         string
		allowedReferrers string
		want             bool
	}{
		{"empty allowed", "https://a.com/", "", true},
		{"empty referrer allowed", "", " , ", true},
		{"host match", "https://a.com/path", "a.com", true},
		{"wildcard subdomain match", "https://sub.example.com/x", "*.example.com", true},
		{"wildcard subdomain no match", "https://other.com/", "*.example.com", false},
		{"invalid referrer URL", "://bad", "a.com", false},
		{"no match", "https://a.com/", "b.com", false},
		{"multiple allowed", "https://b.com/", "a.com,b.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fr.checkReferrers(tt.referrer, tt.allowedReferrers)
			if got != tt.want {
				t.Errorf("checkReferrers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fileBucketRoute_generateKey(t *testing.T) {
	fr := fileBucketRouteForTest(common.Config{})
	fixedTime := time.Date(2025, 2, 24, 12, 30, 45, 0, time.UTC)

	t.Run("default template", func(t *testing.T) {
		values := keyTemplateValues{now: fixedTime, name: "myfile", ext: ".jpg"}
		got := fr.generateKey("", values)
		// Should contain date parts and name; rand is 16 chars; ext from path2.Ext(".jpg") is ".jpg"
		if !strings.Contains(got, "2025") || !strings.Contains(got, "02") || !strings.Contains(got, "24") {
			t.Errorf("generateKey() should contain year/month/date, got %q", got)
		}
		if !strings.Contains(got, "myfile") {
			t.Errorf("generateKey() should contain name, got %q", got)
		}
		if !strings.HasSuffix(got, ".jpg") {
			t.Errorf("generateKey() should end with .jpg, got %q", got)
		}
		// Format: {year}{month}{date}/{name}-{rand}{ext} => 20250224/myfile-<16chars>.jpg
		slashIdx := strings.Index(got, "/")
		if slashIdx < 0 {
			t.Errorf("generateKey() should contain /, got %q", got)
		}
	})

	t.Run("custom deterministic template", func(t *testing.T) {
		values := keyTemplateValues{now: fixedTime, name: "doc", ext: ".pdf"}
		got := fr.generateKey("{year}-{month}-{name}{ext}", values)
		want := "2025-02-doc.pdf"
		if path.Ext(got) != ".pdf" {
			t.Errorf("ext part: got %q", got)
		}
		if got != want {
			t.Errorf("generateKey() = %q, want %q", got, want)
		}
	})

	t.Run("unknown placeholders stripped", func(t *testing.T) {
		values := keyTemplateValues{now: fixedTime, name: "x", ext: ".txt"}
		got := fr.generateKey("{year}{unknown}{name}{ext}", values)
		if strings.Contains(got, "{unknown}") {
			t.Errorf("generateKey() should strip unknown placeholder, got %q", got)
		}
		// Still has known vars
		if !strings.Contains(got, "2025") || !strings.Contains(got, "x") || !strings.HasSuffix(got, ".txt") {
			t.Errorf("generateKey() should still substitute known vars, got %q", got)
		}
	})

	t.Run("empty name uses random string", func(t *testing.T) {
		values := keyTemplateValues{now: fixedTime, name: "", ext: ".bin"}
		got := fr.generateKey("{name}{ext}", values)
		// name when empty is 8-char random, so total len at least 8 + len(".bin") = 12
		if len(got) < 12 {
			t.Errorf("generateKey() with empty name should produce non-trivial string, got %q", got)
		}
		if !strings.HasSuffix(got, ".bin") {
			t.Errorf("generateKey() should end with .bin, got %q", got)
		}
		// Unrecognized placeholder regex should not leave {name} in output (it's recognized)
		if strings.Contains(got, "{") {
			t.Errorf("generateKey() should not contain unsubstituted placeholders, got %q", got)
		}
	})
}

func Test_fileBucketRoute_generateURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("default template", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "https://example.com/api/", nil)
		c.Request.Host = "example.com"
		fr := fileBucketRouteForTest(common.Config{APIPath: "/api"})
		values := urlTemplateValues{ctx: c, bucketName: "my-bucket", key: "path/to/file.png"}
		got := fr.generateURL("", values)
		// Origin from request: https://example.com, plus APIPath /api
		if !strings.Contains(got, "https://example.com") {
			t.Errorf("generateURL() should contain origin, got %q", got)
		}
		if !strings.Contains(got, "/api") {
			t.Errorf("generateURL() should contain APIPath, got %q", got)
		}
		if !strings.Contains(got, "my-bucket") || !strings.Contains(got, "path") {
			t.Errorf("generateURL() should contain bucket and key, got %q", got)
		}
		if strings.Contains(got, "{origin}") || strings.Contains(got, "{bucket}") || strings.Contains(got, "{key}") {
			t.Errorf("generateURL() should substitute all placeholders, got %q", got)
		}
	})

	t.Run("custom template substitution", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "https://cdn.example.com/", nil)
		fr := fileBucketRouteForTest(common.Config{})
		values := urlTemplateValues{ctx: c, bucketName: "b", key: "k/file.txt"}
		got := fr.generateURL("https://cdn.example.com/f/{bucket}/{key}", values)
		// bucket and key should be substituted (key may be URL-encoded)
		if !strings.Contains(got, "b") {
			t.Errorf("generateURL() should contain bucket, got %q", got)
		}
		if !strings.Contains(got, "k") || !strings.Contains(got, "file") {
			t.Errorf("generateURL() should contain key, got %q", got)
		}
		if strings.Contains(got, "{bucket}") || strings.Contains(got, "{key}") {
			t.Errorf("generateURL() should substitute placeholders, got %q", got)
		}
	})

	t.Run("unknown placeholders stripped", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "https://h.com/", nil)
		fr := fileBucketRouteForTest(common.Config{APIPath: ""})
		values := urlTemplateValues{ctx: c, bucketName: "b", key: "k"}
		got := fr.generateURL("https://h.com/{bucket}/{key}{foo}", values)
		if strings.Contains(got, "{foo}") {
			t.Errorf("generateURL() should strip unknown placeholder, got %q", got)
		}
	})

	// Ensure we don't leave any {...} in output (regex strips unknown vars)
	t.Run("no leftover placeholders", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "https://x.com/", nil)
		fr := fileBucketRouteForTest(common.Config{APIPath: "/api"})
		values := urlTemplateValues{ctx: c, bucketName: "b", key: "k"}
		got := fr.generateURL("", values)
		re := regexp.MustCompile(`\{[^}]*\}`)
		if re.MatchString(got) {
			t.Errorf("generateURL() should not contain any {...} placeholder, got %q", got)
		}
	})
}
