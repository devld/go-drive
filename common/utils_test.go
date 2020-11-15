package common

import (
	"regexp"
	"testing"
	"time"
)

func TestPathParent(t *testing.T) {
	if p := PathParent("/"); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "/", "", p)
	}
	if p := PathParent(""); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "", "", p)
	}
	if p := PathParent(".."); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "..", "", p)
	}
	if p := PathParent("..."); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "...", "", p)
	}
	if p := PathParent("/a/b/c"); p != "a/b" {
		t.Errorf("'%s': expect '%s', but is '%s'", "/a/b/c", "a/b", p)
	}
	if p := PathParent("a/b/c"); p != "a/b" {
		t.Errorf("'%s': expect '%s', but is '%s'", "a/b/c", "a/b", p)
	}
	if p := PathParent("a/../../c"); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "a/../../c", "", p)
	}
	if p := PathParent("a/../../../c"); p != "" {
		t.Errorf("'%s': expect '%s', but is '%s'", "a/../../../c", "", p)
	}
	if p := PathParent("/a/b/c/"); p != "a/b" {
		t.Errorf("'%s': expect '%s', but is '%s'", "/a/b/c/", "a/b", p)
	}
	if p := PathParent("//a/b/c/"); p != "a/b" {
		t.Errorf("'%s': expect '%s', but is '%s'", "//a/b/c", "a/b", p)
	}
}

func TestPathParentTree(t *testing.T) {
	path := "a/b/c"

	r := PathParentTree(path)
	if r[3] != "" || r[2] != "a" || r[1] != "a/b" || r[0] != "a/b/c" {
		t.Errorf("'%s': expect '%v', but is '%v'", path, []string{"a/b/c", "a/b", "a"}, r)
	}
}

func TestRegSplit(t *testing.T) {
	ss := RegSplit("a, b,c;d", regexp.MustCompile("[,;]\\s*"))
	if len(ss) != 4 || ss[0] != "a" || ss[1] != "b" || ss[2] != "c" || ss[3] != "d" {
		t.Errorf("RegSplit: 'a, b,c;d', expect '[a, b, c, d]', but '%v'", ss)
	}
	ss = RegSplit("abcd", regexp.MustCompile("[,;]\\s*"))
	if len(ss) != 1 || ss[0] != "abcd" {
		t.Errorf("RegSplit: 'abcd', expect '[abcd]', but '%v'", ss)
	}
}

func TestTimeTick(t *testing.T) {
	n := 0
	stop := TimeTick(func() {
		n++
	}, 90*time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	if n != 3 {
		t.Errorf("expect n=%d, but it's %d", 3, n)
	}
	stop()
	time.Sleep(100 * time.Millisecond)
	if n != 3 {
		t.Errorf("expect n=%d, but it's %d", 3, n)
	}
}

func TestFormatBytes(t *testing.T) {
	if v := FormatBytes(12318263771, 2); v != "11.47 G" {
		t.Errorf("expect %s, but it's %s", "11.47 G", v)
	}
	if v := FormatBytes(2251799813685248, 2); v != "2048.00 T" {
		t.Errorf("expect %s, but it's %s", "2048.00 T", v)
	}
}

func TestBuildURL(t *testing.T) {
	if v := BuildURL("/a/{}/d/{}", "b/c", "e"); v != "/a/b/c/d/e" {
		t.Errorf("expect '%s', but it's '%s'", "/a/b/c/d/e", v)
	}
	if v := BuildURL("/a/{}/d/{}", "b/c"); v != "/a/b/c/d/{}" {
		t.Errorf("expect '%s', but it's '%s'", "/a/b/c/d/{}", v)
	}
	if v := BuildURL("/a/{}/d/{}", "b/c", "e", "F"); v != "/a/b/c/d/e" {
		t.Errorf("expect '%s', but it's '%s'", "/a/b/c/d/e", v)
	}
	if v := BuildURL("/a/{}/d", "b/c"); v != "/a/b/c/d" {
		t.Errorf("expect '%s', but it's '%s'", "/a/b/c/d", v)
	}
	if v := BuildURL("", "b/c"); v != "" {
		t.Errorf("expect '%s', but it's '%s'", "", v)
	}
	if v := BuildURL("/a/{}/d/{}", "b/你好", "世界"); v != "/a/b/%E4%BD%A0%E5%A5%BD/d/%E4%B8%96%E7%95%8C" {
		t.Errorf("expect '%s', but it's '%s'", "/a/%E4%BD%A0%E5%A5%BD/d/%E4%B8%96%E7%95%8C", v)
	}
}
