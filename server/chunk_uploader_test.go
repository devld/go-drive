package server

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_isValidUploadIdPart(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"f47ac10b-58cc-0372-8567-0e02b2c3d479", true},
		{"ABCDEF0123456789", true},
		{"", false},
		{"../secret", false},
		{"..", false},
		{"a/b", false},
		{"a\\b", false},
		{"name_with_underscore", false},
		{"g123", false}, // 'g' is not a hex char
		{"with space", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := isValidUploadIdPart(tt.in); got != tt.want {
				t.Errorf("isValidUploadIdPart(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestChunkUploader_getUpload_RejectsTraversalId(t *testing.T) {
	c := &ChunkUploader{dir: t.TempDir()}
	for _, id := range []string{"../secret_1_1", "..%2f_1_1", "a/b_1_1", "..\\x_1_1"} {
		if _, e := c.getUpload(id); e == nil {
			t.Errorf("expected error for malicious id %q", id)
		}
	}
}

// TestChunkUploader_DeleteUpload_DoesNotEscapeDir ensures a crafted upload id
// cannot cause DeleteUpload to remove a directory outside the upload root.
func TestChunkUploader_DeleteUpload_DoesNotEscapeDir(t *testing.T) {
	base := t.TempDir()
	uploadDir := filepath.Join(base, "upload")
	if e := os.Mkdir(uploadDir, 0755); e != nil {
		t.Fatal(e)
	}

	// a sibling directory the crafted id "../secret_1_1" would resolve to
	target := filepath.Join(base, "secret_1_1")
	if e := os.Mkdir(target, 0755); e != nil {
		t.Fatal(e)
	}
	sentinel := filepath.Join(target, "keep.txt")
	if e := os.WriteFile(sentinel, []byte("x"), 0644); e != nil {
		t.Fatal(e)
	}

	c := &ChunkUploader{dir: uploadDir}
	_ = c.DeleteUpload("../secret_1_1")

	if _, e := os.Stat(sentinel); e != nil {
		t.Fatalf("sentinel file should be preserved, but got: %v", e)
	}
}
