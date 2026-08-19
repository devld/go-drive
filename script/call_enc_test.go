package script

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func newScriptTestVM(t *testing.T) *VM {
	t.Helper()
	vm := newPoolTestVM(t).Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	return vm
}

func TestEncBase64Padding(t *testing.T) {
	vm := newScriptTestVM(t)
	padded := evalJSString(t, vm, `encUtils.urlBase64Encode(newBytes("hello"))`)
	if padded != "aGVsbG8=" {
		t.Fatalf("urlBase64Encode padded = %q, want aGVsbG8=", padded)
	}
	raw := evalJSString(t, vm, `encUtils.urlBase64Encode(newBytes("hello"), false)`)
	if raw != "aGVsbG8" {
		t.Fatalf("urlBase64Encode raw = %q, want aGVsbG8", raw)
	}
	decoded := evalJSString(t, vm, `encUtils.urlBase64Decode("aGVsbG8", false).String()`)
	if decoded != "hello" {
		t.Fatalf("urlBase64Decode raw = %q, want hello", decoded)
	}
	stdRaw := evalJSString(t, vm, `encUtils.base64Encode(newBytes("hello"), false)`)
	if stdRaw != "aGVsbG8" {
		t.Fatalf("base64Encode raw = %q, want aGVsbG8", stdRaw)
	}
	stdDecoded := evalJSString(t, vm, `encUtils.base64Decode("aGVsbG8", false).String()`)
	if stdDecoded != "hello" {
		t.Fatalf("base64Decode raw = %q, want hello", stdDecoded)
	}
}

func TestEncRandomBytes(t *testing.T) {
	vm := newScriptTestVM(t)
	a := evalJSString(t, vm, `encUtils.toHex(encUtils.randomBytes(16))`)
	b := evalJSString(t, vm, `encUtils.toHex(encUtils.randomBytes(16))`)
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("randomBytes hex length a=%d b=%d, want 32", len(a), len(b))
	}
	if a == b {
		t.Fatal("randomBytes returned the same value twice")
	}
	empty := evalJSInt(t, vm, `encUtils.randomBytes(0).Len()`)
	if empty != 0 {
		t.Fatalf("randomBytes(0).Len() = %d, want 0", empty)
	}
	if _, e := vm.Run(context.Background(), `encUtils.randomBytes(-1)`); e == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestEncWriteReaderRewindsTempFile(t *testing.T) {
	vm := newScriptTestVM(t)
	payload := "abc"
	md5Sum := md5.Sum([]byte(payload))
	mac := hmac.New(sha256.New, []byte("key"))
	_, _ = mac.Write([]byte(payload))
	wantHMAC := hex.EncodeToString(mac.Sum(nil))

	got := evalJSString(t, vm, `
(function() {
  var tmp = newTempFile();
  tmp.Write(newBytes("abc"));
  tmp.SeekTo(0, SEEK_START);
  var md5 = encUtils.toHex(encUtils.newHash(HASH.MD5).WriteReader(tmp).Sum());
  var hmacHex = encUtils.toHex(
    encUtils.newHmac(HASH.SHA256, newBytes("key")).WriteReader(tmp).Sum()
  );
  var oneShot = encUtils.toHex(
    encUtils.newHmac(HASH.SHA256, newBytes("key")).Write(newBytes("abc")).Sum()
  );
  var again = tmp.ReadAsString();
  tmp.Close();
  if (hmacHex !== oneShot) throw new Error("hmac mismatch");
  return md5 + " " + hmacHex + " " + again;
})()
`)
	want := hex.EncodeToString(md5Sum[:]) + " " + wantHMAC + " abc"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncWriteReaderFromCurrentOffset(t *testing.T) {
	vm := newScriptTestVM(t)
	want := md5.Sum([]byte("cdefgh"))
	got := evalJSString(t, vm, `
(function() {
  var tmp = newTempFile();
  tmp.Write(newBytes("abcdefgh"));
  tmp.SeekTo(2, SEEK_START);
  var sum = encUtils.toHex(encUtils.newHash(HASH.MD5).WriteReader(tmp).Sum());
  var rest = tmp.ReadAsString();
  tmp.Close();
  return sum + " " + rest;
})()
`)
	if got != hex.EncodeToString(want[:])+" cdefgh" {
		t.Fatalf("got %q, want hash of cdefgh then remainder cdefgh", got)
	}
}

func evalJSString(t *testing.T, vm *VM, code string) string {
	t.Helper()
	v, e := vm.Run(context.Background(), code)
	if e != nil {
		t.Fatal(e)
	}
	return v.String()
}

func evalJSInt(t *testing.T, vm *VM, code string) int64 {
	t.Helper()
	v, e := vm.Run(context.Background(), code)
	if e != nil {
		t.Fatal(e)
	}
	return v.Integer()
}

func TestEncRandomBytesRejectsTooLarge(t *testing.T) {
	vm := newScriptTestVM(t)
	_, e := vm.Run(context.Background(), `encUtils.randomBytes(1048577)`)
	if e == nil || !strings.Contains(e.Error(), "randomBytes") {
		t.Fatalf("expected size error, got %v", e)
	}
}
