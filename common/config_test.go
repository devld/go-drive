package common

import (
	"go-drive/common/logging"
	"testing"
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

func TestInvalidLoggingLevel(t *testing.T) {
	t.Setenv("GO_DRIVE_LOGGING_LEVEL", "trace")
	if err := applyLoggingConfig(&LoggingConfig{Level: DefaultLoggingLevel}); err == nil {
		t.Fatal("applyLoggingConfig() error = nil, want invalid environment level error")
	}
}
