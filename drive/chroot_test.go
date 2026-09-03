package drive

import (
	"testing"
)

func TestChrootWrapPathNormalizesBackslashTraversal(t *testing.T) {
	c := NewChroot("files/users/bob", nil)

	got, e := c.WrapPath(`..\..\secret`)
	if e != nil {
		t.Fatalf("WrapPath: %v", e)
	}
	if got != "files/users/bob/secret" {
		t.Fatalf("WrapPath backslash traversal = %q, want jailed path", got)
	}

	got, e = c.WrapPath(`users/bob/..\..\..\Windows\Temp\x`)
	if e != nil {
		t.Fatalf("WrapPath mixed: %v", e)
	}
	if got != "files/users/bob/Windows/Temp/x" {
		t.Fatalf("WrapPath mixed = %q", got)
	}
}
