package secretbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	box, e := New("test-secret")
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := box.Encrypt("s3-secret")
	if e != nil {
		t.Fatal(e)
	}
	if !IsSealed(sealed) {
		t.Fatalf("sealed = %q", sealed)
	}
	if strings.Contains(sealed, "s3-secret") {
		t.Fatalf("ciphertext contains plaintext: %q", sealed)
	}
	opened, e := box.Decrypt(sealed)
	if e != nil {
		t.Fatal(e)
	}
	if opened != "s3-secret" {
		t.Fatalf("opened = %q", opened)
	}
}

func TestDecryptLeavesOrdinaryPasswordUntouched(t *testing.T) {
	box, e := New("test-secret")
	if e != nil {
		t.Fatal(e)
	}
	for _, value := range []string{"password", "gd1:abc", "$go-drive-secret-v1$short", ""} {
		got, e := box.Decrypt(value)
		if e != nil {
			t.Fatal(e)
		}
		if got != value {
			t.Fatalf("Decrypt(%q) = %q", value, got)
		}
	}
}

func TestDecryptRejectsForeignSeal(t *testing.T) {
	box, e := New("one")
	if e != nil {
		t.Fatal(e)
	}
	other, e := New("two")
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := other.Encrypt("secret")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = box.Decrypt(sealed); e == nil {
		t.Fatal("expected decrypt failure")
	}
}

func TestOpenUsesEnvAndRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvKey, "env-secret")
	box, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := box.Encrypt("token")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(dir, keyFileName)); !os.IsNotExist(e) {
		t.Fatalf("env key created a file: %v", e)
	}

	if e = os.WriteFile(filepath.Join(dir, keyFileName), []byte("file-secret\n"), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(dir); e == nil {
		t.Fatal("expected mismatch")
	}

	t.Setenv(EnvKey, "")
	fromFile, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = fromFile.Decrypt(sealed); e == nil {
		t.Fatal("file key decrypted an env ciphertext")
	}
}

func TestOpenGeneratesKeyFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvKey, "")
	if _, e := Open(dir); e != nil {
		t.Fatal(e)
	}
	info, e := os.Stat(filepath.Join(dir, keyFileName))
	if e != nil {
		t.Fatal(e)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	again, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := again.Encrypt("again")
	if e != nil {
		t.Fatal(e)
	}
	if !IsSealed(sealed) {
		t.Fatalf("sealed = %q", sealed)
	}
}
