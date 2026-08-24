package script

import (
	"context"
	"strings"
	"testing"
)

func TestURLUtilsParseAndBuild(t *testing.T) {
	vm := newScriptTestVM(t)

	got := evalJSString(t, vm, `
(function() {
  var parsed = urlUtils.parse(
    "https://alice:secret@example.com:8443/a%2Fb/file.txt" +
    "?tag=a&tag=b&q=hello+world#frag%20ment"
  );
  return [
    parsed.origin,
    parsed.protocol,
    parsed.username,
    parsed.password,
    parsed.host,
    parsed.hostname,
    parsed.port,
    parsed.pathname,
    typeof parsed.search,
    JSON.stringify(parsed.searchParams),
    parsed.hash
  ].join("|");
})()
`)
	want := "https://example.com:8443|https:|alice|secret|example.com:8443|example.com|8443|/a%2Fb/file.txt|undefined|{\"q\":[\"hello world\"],\"tag\":[\"a\",\"b\"]}|#frag%20ment"
	if got != want {
		t.Fatalf("urlUtils.parse() = %q, want %q", got, want)
	}

	got = evalJSString(t, vm, `
(function() {
  var parsed = urlUtils.parse(
    "https://alice:secret@example.com:8443/a%2Fb/file.txt" +
    "?tag=a&tag=b&q=hello+world#frag%20ment"
  );
  parsed.pathname = "/changed%2Ffile";
  parsed.searchParams.q = ["x", "y"];
  return urlUtils.build(parsed);
})()
`)
	want = "https://alice:secret@example.com:8443/changed%2Ffile?q=x&q=y&tag=a&tag=b#frag%20ment"
	if got != want {
		t.Fatalf("urlUtils.build() after JS mutation = %q, want %q", got, want)
	}
}

func TestURLUtilsSearchParams(t *testing.T) {
	vm := newScriptTestVM(t)

	got := evalJSString(t, vm, `JSON.stringify(urlUtils.parseSearchParams("?b=2&a=1&a=two&space=hello+world&flag"))`)
	want := "{\"a\":[\"1\",\"two\"],\"b\":[\"2\"],\"flag\":[\"\"],\"space\":[\"hello world\"]}"
	if got != want {
		t.Fatalf("urlUtils.parseSearchParams() = %q, want %q", got, want)
	}

	got = evalJSString(t, vm, `urlUtils.buildSearchParams({b: ["2"], a: ["1", "two"], space: ["hello world"]})`)
	want = "?a=1&a=two&b=2&space=hello+world"
	if got != want {
		t.Fatalf("urlUtils.buildSearchParams() = %q, want %q", got, want)
	}

	got = evalJSString(t, vm, `urlUtils.buildSearchParams({})`)
	if got != "" {
		t.Fatalf("urlUtils.buildSearchParams({}) = %q, want empty string", got)
	}

	got = evalJSString(t, vm, `
urlUtils.build({
  protocol: "https:",
  hostname: "example.com",
  pathname: "/path",
  searchParams: {q: ["hello world"]}
})
`)
	want = "https://example.com/path?q=hello+world"
	if got != want {
		t.Fatalf("urlUtils.build() = %q, want %q", got, want)
	}

	got = evalJSString(t, vm, `urlUtils.build(urlUtils.parse("https://example.com/path"))`)
	if got != "https://example.com/path" {
		t.Fatalf("urlUtils.build(urlUtils.parse()) = %q, want https://example.com/path", got)
	}
}

func TestURLUtilsRejectsInvalidInput(t *testing.T) {
	vm := newScriptTestVM(t)

	assertURLUtilsError(t, vm, `urlUtils.parse("https://example.com/?bad=%zz")`, "urlUtils.parse")
	assertURLUtilsError(t, vm, `urlUtils.parseSearchParams("?bad=%zz")`, "urlUtils.parseSearchParams")
	assertURLUtilsError(t, vm, `urlUtils.buildSearchParams({q: ["ok", 1]})`, "urlUtils.buildSearchParams")
	assertURLUtilsError(t, vm, `urlUtils.build({searchParams: {q: "ok"}})`, "urlUtils.build")
	assertURLUtilsError(t, vm, `urlUtils.build({protocol: null})`, "urlUtils.build")
	assertURLUtilsError(t, vm, `urlUtils.build({searchParams: null})`, "urlUtils.build")
}

func assertURLUtilsError(t *testing.T, vm *VM, code, want string) {
	t.Helper()
	_, e := vm.Run(context.Background(), code)
	if e == nil || !strings.Contains(e.Error(), want) {
		t.Fatalf("running %s error = %v, want an error containing %q", code, e, want)
	}
}
