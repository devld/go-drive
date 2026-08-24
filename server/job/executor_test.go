package job

import "testing"

func TestJobExecutionLoggerPreservesMessage(t *testing.T) {
	var got string
	logger := newJobExecutionLogger(42, 960, func(message string) {
		got = message
	})

	logger.Log("executed: 1")

	if got != "executed: 1" {
		t.Fatalf("onLog message = %q, want original message", got)
	}
	if logs := logger.String(); logs != "executed: 1\n" {
		t.Fatalf("stored logs = %q, want original message", logs)
	}
}
