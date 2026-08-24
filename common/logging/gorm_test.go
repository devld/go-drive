package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	gormLogger "gorm.io/gorm/logger"
)

func TestGormLoggerMapsSQLLevels(t *testing.T) {
	var output bytes.Buffer
	appLogger := New(&output)
	gormLog := newGormLogger(appLogger, gormLogger.Config{
		SlowThreshold:             time.Millisecond,
		LogLevel:                  gormLogger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})

	gormLog.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)
	if output.Len() != 0 {
		t.Fatalf("regular SQL logged at info level: %s", output.String())
	}

	appLogger.SetLevel(DebugLevel)
	gormLog = gormLog.LogMode(gormLogger.Info)
	gormLog.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)
	if !strings.Contains(output.String(), "[DEBUG]") {
		t.Fatalf("regular SQL missing debug level: %s", output.String())
	}
	if !strings.Contains(output.String(), "[rows:1] SELECT 1\n  at ") {
		t.Fatalf("regular SQL has unexpected layout: %s", output.String())
	}

	output.Reset()
	appLogger.SetLevel(InfoLevel)
	gormLog.Trace(context.Background(), time.Now().Add(-time.Second), func() (string, int64) {
		return "SELECT slow", 1
	}, nil)
	if !strings.Contains(output.String(), "[WARN]") {
		t.Fatalf("slow SQL missing warn level: %s", output.String())
	}

	output.Reset()
	gormLog.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT failed", -1
	}, context.Canceled)
	if !strings.Contains(output.String(), "[ERROR]") {
		t.Fatalf("failed SQL missing error level: %s", output.String())
	}
}
