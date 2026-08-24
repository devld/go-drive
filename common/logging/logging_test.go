package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerForPrintfPreservesMessage(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out).For("script")
	logger.Printf("runtime error\n    at test.js:1:1")

	got := out.String()
	if !strings.Contains(got, " [INFO] [script ] runtime error\n      at test.js:1:1\n") {
		t.Fatalf("log = %q, want component prefix and original newlines", got)
	}
	if strings.Contains(got, `\n`) {
		t.Fatalf("log = %q, want no newline escaping", got)
	}
}

func TestSanitizeEscapesLineBreaks(t *testing.T) {
	got := Sanitize("line 1\r\nline 2")
	if want := `line 1\r\nline 2`; got != want {
		t.Fatalf("Sanitize() = %q, want %q", got, want)
	}
}

func TestComponentViewsShareOutput(t *testing.T) {
	var out bytes.Buffer
	root := New(&out)
	root.For("http").Printf("request completed")
	root.For("db").Printf("query completed")

	got := out.String()
	if !strings.Contains(got, " [INFO] [ http  ] request completed") ||
		!strings.Contains(got, " [INFO] [  db   ] query completed") {
		t.Fatalf("log = %q, want both component prefixes", got)
	}
}

func TestLoggerLevels(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out).For("test")

	logger.Debugf("debug hidden")
	logger.Infof("info visible")
	logger.Warnf("warn visible")
	logger.Errorf("error visible")

	got := out.String()
	if strings.Contains(got, "debug hidden") {
		t.Fatalf("log = %q, want debug message filtered by default", got)
	}
	for _, want := range []string{
		" [INFO] [ test  ] info visible",
		" [WARN] [ test  ] warn visible",
		"[ERROR] [ test  ] error visible",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("log = %q, want %q", got, want)
		}
	}

	logger.SetLevel(DebugLevel)
	logger.Debugf("debug visible")
	if !strings.Contains(out.String(), "[DEBUG] [ test  ] debug visible") {
		t.Fatalf("log = %q, want debug message after lowering level", out.String())
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		value string
		want  Level
	}{
		{value: "", want: InfoLevel},
		{value: "info", want: InfoLevel},
		{value: "DEBUG", want: DebugLevel},
		{value: "warning", want: WarnLevel},
		{value: "error", want: ErrorLevel},
	}
	for _, tt := range tests {
		got, err := ParseLevel(tt.value)
		if err != nil {
			t.Fatalf("ParseLevel(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
	if _, err := ParseLevel("trace"); err == nil {
		t.Fatal("ParseLevel(\"trace\") error = nil, want unsupported level error")
	}
}
