package common

import (
	"os"
	"path/filepath"
	"testing"

	"go-drive/common/logging"
)

func TestApplyLoggingConfigDefaultsToInfo(t *testing.T) {
	t.Setenv("GO_DRIVE_LOGGING_LEVEL", "")
	defer logging.SetLevel(logging.InfoLevel)
	logging.SetLevel(logging.ErrorLevel)

	if err := applyLoggingConfig(&LoggingConfig{Level: DefaultLoggingLevel}); err != nil {
		t.Fatalf("applyLoggingConfig() error = %v", err)
	}
	if !logging.Enabled(logging.InfoLevel) || !logging.Enabled(logging.ErrorLevel) {
		t.Fatal("default info level should enable info and error logs")
	}
	if logging.Enabled(logging.DebugLevel) {
		t.Fatal("default info level should filter debug logs")
	}
}

func TestLoggingLevelEnvironmentOverridesConfig(t *testing.T) {
	t.Setenv("GO_DRIVE_LOGGING_LEVEL", "error")
	defer logging.SetLevel(logging.InfoLevel)

	if err := applyLoggingConfig(&LoggingConfig{Level: "debug"}); err != nil {
		t.Fatalf("applyLoggingConfig() error = %v", err)
	}
	if logging.Enabled(logging.WarnLevel) || !logging.Enabled(logging.ErrorLevel) {
		t.Fatal("environment error level should override config debug level")
	}
}

func TestGetTempDirCreatesNestedPrivateDir(t *testing.T) {
	root := t.TempDir()
	dir, err := (Config{TempDir: root}).GetTempDir(filepath.Join("mega-cache", "cloud"), true)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(root, "mega-cache", "cloud") {
		t.Fatalf("dir = %s", dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if _, err = (Config{}).GetTempDir("upload", true); err == nil {
		t.Fatal("empty temp dir was accepted")
	}
}

func TestInvalidLoggingLevel(t *testing.T) {
	t.Setenv("GO_DRIVE_LOGGING_LEVEL", "trace")
	if err := applyLoggingConfig(&LoggingConfig{Level: DefaultLoggingLevel}); err == nil {
		t.Fatal("applyLoggingConfig() error = nil, want invalid environment level error")
	}
}
