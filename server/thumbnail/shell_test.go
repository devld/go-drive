package thumbnail

import (
	"go-drive/common/types"
	"reflect"
	"testing"
)

func TestShellThumbnailMultilineScript(t *testing.T) {
	script := "printf first\nprintf second"
	handler, e := newShellThumbnailTypeHandler(types.SM{
		"shell":     script,
		"mime-type": "image/jpeg",
	})
	if e != nil {
		t.Fatal(e)
	}
	shellHandler := handler.(*shellThumbnailTypeHandler)
	wantCommand, wantArgs := platformShellCommand("linux", script)
	if shellHandler.command != wantCommand {
		t.Skip("test assertions target the Unix test environment")
	}
	if !reflect.DeepEqual(shellHandler.args, wantArgs) {
		t.Fatalf("unexpected arguments: %#v", shellHandler.args)
	}
}

func TestPlatformShellCommand(t *testing.T) {
	tests := []struct {
		goos        string
		wantCommand string
		wantArgs    []string
	}{
		{goos: "linux", wantCommand: "/bin/sh", wantArgs: []string{"-c", "first\nsecond"}},
		{goos: "windows", wantCommand: "cmd.exe", wantArgs: []string{"/D", "/S", "/C", "first\nsecond"}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			command, args := platformShellCommand(tt.goos, "first\nsecond")
			if command != tt.wantCommand || !reflect.DeepEqual(args, tt.wantArgs) {
				t.Fatalf("unexpected command: %q %#v", command, args)
			}
		})
	}
}
