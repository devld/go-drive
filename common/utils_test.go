package common

import (
	"regexp"
	"testing"
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
	if r[2] != "a" || r[1] != "a/b" || r[0] != "a/b/c" {
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
