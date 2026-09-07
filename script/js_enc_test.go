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
	vm := newPoolTestVM(t)
	t.Cleanup(func() { _ = vm.Dispose() })
	return vm
}

func TestEncBase64Padding(t *testing.T) {
	vm := newScriptTestVM(t)
	padded := evalJSString(t, vm, `Bytes.fromString("hello").toString("base64url")`)
	if padded != "aGVsbG8=" {
		t.Fatalf("base64url padded = %q, want aGVsbG8=", padded)
	}
	raw := evalJSString(t, vm, `Bytes.fromString("hello").toString("base64url", {padded: false})`)
	if raw != "aGVsbG8" {
		t.Fatalf("base64url raw = %q, want aGVsbG8", raw)
	}
	decoded := evalJSString(t, vm, `Bytes.fromBase64Url("aGVsbG8", {padded: false}).toString()`)
	if decoded != "hello" {
		t.Fatalf("fromBase64Url raw = %q, want hello", decoded)
	}
	stdRaw := evalJSString(t, vm, `Bytes.fromString("hello").toString("base64", {padded: false})`)
	if stdRaw != "aGVsbG8" {
		t.Fatalf("base64 raw = %q, want aGVsbG8", stdRaw)
	}
	stdDecoded := evalJSString(t, vm, `Bytes.fromBase64("aGVsbG8", {padded: false}).toString()`)
	if stdDecoded != "hello" {
		t.Fatalf("fromBase64 raw = %q, want hello", stdDecoded)
	}
}

func TestEncRandomBytes(t *testing.T) {
	vm := newScriptTestVM(t)
	a := evalJSString(t, vm, `Bytes.random(16).toString("hex")`)
	b := evalJSString(t, vm, `Bytes.random(16).toString("hex")`)
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("random hex length a=%d b=%d, want 32", len(a), len(b))
	}
	if a == b {
		t.Fatal("Bytes.random returned the same value twice")
	}
	empty := evalJSInt(t, vm, `Bytes.random(0).length`)
	if empty != 0 {
		t.Fatalf("Bytes.random(0).length = %d, want 0", empty)
	}
	if _, e := vm.Run(context.Background(), `Bytes.random(-1)`, ""); e == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestBytesSliceClampsAndOmitsEnd(t *testing.T) {
	vm := newScriptTestVM(t)
	got := evalJSString(t, vm, `
		[
			Bytes.fromString("hello").slice(1, 4).toString(),
			Bytes.fromString("hello").slice(1).toString(),
			Bytes.fromString("hello").slice(-2).toString(),
			Bytes.fromString("hello").slice(0, 99).toString(),
			String(Bytes.fromString("hello").slice(4, 1).length)
		].join("|");
	`)
	if got != "ell|ello|lo|hello|0" {
		t.Fatalf("Bytes.slice = %q", got)
	}
}

func TestBytesRejectsInvalidConstruction(t *testing.T) {
	vm := newScriptTestVM(t)
	if _, e := vm.Run(context.Background(), `new Bytes(-1)`, ""); e == nil || !strings.Contains(e.Error(), "non-negative") {
		t.Fatalf("new Bytes(-1) = %v", e)
	}
	for _, code := range []string{`new Bytes(Number.MAX_SAFE_INTEGER)`, `new Bytes({})`, `new Bytes(true)`, `new Bytes()`, `new Bytes("x")`} {
		_, e := vm.Run(context.Background(), "try { "+code+"; throw new Error('expected TypeError'); } catch (e) { if (!(e instanceof TypeError)) throw e; }", "")
		if e != nil {
			t.Fatalf("%s escaped VM.Run: %v", code, e)
		}
	}
}

func TestBytesFromStringCopiesUTF8(t *testing.T) {
	vm := newScriptTestVM(t)
	got := evalJSString(t, vm, `
		Bytes.fromString("hi").toString("hex") === "6869" &&
		Bytes.fromString("hi") instanceof Bytes
			? "ok" : "fail"
	`)
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
	if _, e := vm.Run(context.Background(), `Bytes.fromString(1)`, ""); e == nil || !strings.Contains(e.Error(), "fromString") {
		t.Fatalf("Bytes.fromString(1) = %v", e)
	}
}

func TestTempFileCopyFromDoesNotCloseSource(t *testing.T) {
	vm := newScriptTestVM(t)
	got := evalJSString(t, vm, `
(function() {
  var src = new TempFile();
  src.write(Bytes.fromString("abc"));
  src.seekTo(0, SEEK_START);
  var dst = new TempFile();
  dst.copyFrom(src);
  src.seekTo(0, SEEK_START);
  var again = src.readAsString();
  dst.seekTo(0, SEEK_START);
  var copied = dst.readAsString();
  src.close();
  dst.close();
  return again + "|" + copied;
})()
`)
	if got != "abc|abc" {
		t.Fatalf("copyFrom = %q, want abc|abc", got)
	}
}

func TestEncUtilsRequiresBytes(t *testing.T) {
	vm := newScriptTestVM(t)
	cases := []struct {
		code, want string
	}{
		{`Bytes.fromString("x").toString("nope")`, "unknown Bytes encoding"},
		{`new Hmac("sha256", "key")`, "Hmac requires Bytes"},
		{`new Hash("md5").write("abc")`, "Write requires Bytes"},
		{`new Hash("md5").writeFrom("x")`, "WriteFrom requires a Reader"},
		{`new Hash("nope")`, "unknown hash algorithm"},
		{`new TempFile().write("x")`, "Write requires Bytes"},
	}
	for _, c := range cases {
		if _, e := vm.Run(context.Background(), c.code, ""); e == nil || !strings.Contains(e.Error(), c.want) {
			t.Fatalf("%s = %v", c.code, e)
		}
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
  var tmp = new TempFile();
  tmp.write(Bytes.fromString("abc"));
  tmp.seekTo(0, SEEK_START);
  var md5 = new Hash("md5").writeFrom(tmp).sum().toString("hex");
  var hmacHex = new Hmac("sha256", Bytes.fromString("key")).writeFrom(tmp).sum().toString("hex");
  var oneShot = new Hmac("sha256", Bytes.fromString("key")).write(Bytes.fromString("abc")).sum().toString("hex");
  var again = tmp.readAsString();
  tmp.close();
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
  var tmp = new TempFile();
  tmp.write(Bytes.fromString("abcdefgh"));
  tmp.seekTo(2, SEEK_START);
  var sum = new Hash("md5").writeFrom(tmp).sum().toString("hex");
  var rest = tmp.readAsString();
  tmp.close();
  return sum + " " + rest;
})()
`)
	if got != hex.EncodeToString(want[:])+" cdefgh" {
		t.Fatalf("got %q, want hash of cdefgh then remainder cdefgh", got)
	}
}

func evalJSString(t *testing.T, vm *VM, code string) string {
	t.Helper()
	v, e := vm.Run(context.Background(), code, "")
	if e != nil {
		t.Fatal(e)
	}
	return v.String()
}

func evalJSInt(t *testing.T, vm *VM, code string) int64 {
	t.Helper()
	v, e := vm.Run(context.Background(), code, "")
	if e != nil {
		t.Fatal(e)
	}
	return v.Integer()
}

func TestTempFileReadRejectsNonBytes(t *testing.T) {
	vm := newScriptTestVM(t)
	if _, e := vm.Run(context.Background(), `
		var tmp = new TempFile();
		tmp.write(Bytes.fromString("hi"));
		tmp.seekTo(0, SEEK_START);
		try { tmp.read("nope"); throw new Error("read(string) succeeded"); }
		finally { tmp.close(); }
	`, ""); e == nil || strings.Contains(e.Error(), "succeeded") {
		t.Fatalf("expected Reader.read(string) to fail, got %v", e)
	}
}

func TestTempFileWriteAndReadBytes(t *testing.T) {
	vm := newScriptTestVM(t)
	got := evalJSString(t, vm, `
(function() {
  var tmp = new TempFile();
  tmp.write(Bytes.fromString("hi"));
  tmp.seekTo(0, SEEK_START);
  var buf = new Bytes(2);
  var n = tmp.read(buf);
  tmp.close();
  return n + ":" + buf.toString();
})()
`)
	if got != "2:hi" {
		t.Fatalf("got %q, want 2:hi", got)
	}
}

func TestEncRandomBytesRejectsTooLarge(t *testing.T) {
	vm := newScriptTestVM(t)
	_, e := vm.Run(context.Background(), `Bytes.random(1048577)`, "")
	if e == nil || !strings.Contains(e.Error(), "Bytes.random") {
		t.Fatalf("expected size error, got %v", e)
	}
}

func TestBytesFromHexAndDefaultToString(t *testing.T) {
	vm := newScriptTestVM(t)
	got := evalJSString(t, vm, `
		Bytes.fromHex("6869").toString() === "hi" &&
		Bytes.fromString("hi").toString("hex") === "6869"
			? "ok" : "fail"
	`)
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
}
