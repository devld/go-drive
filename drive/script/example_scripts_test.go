package script

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerSideExampleScriptsEvaluate(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "script-drives", "dropbox.js"),
		filepath.Join("..", "..", "script-drives", "qiniu.js"),
		filepath.Join("..", "..", "docs", "script-drive-template.js"),
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			contents, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}

			vm := baseVM.Fork()
			t.Cleanup(func() { _ = vm.Dispose() })
			if _, e = vm.Run(context.Background(), contents); e != nil {
				t.Fatalf("evaluate %s: %v", path, e)
			}
		})
	}
}

func TestAgentGuideCompleteExampleEvaluates(t *testing.T) {
	path := filepath.Join("..", "..", "script-drives", "AGENTS.md")
	contents, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}

	section := strings.Index(string(contents), "## 8. Minimal complete example")
	if section < 0 {
		t.Fatal("complete example section not found")
	}
	codeStart := strings.Index(string(contents[section:]), "```js\n")
	if codeStart < 0 {
		t.Fatal("complete example code block not found")
	}
	codeStart += section + len("```js\n")
	codeEnd := strings.Index(string(contents[codeStart:]), "\n```")
	if codeEnd < 0 {
		t.Fatal("complete example code block is not closed")
	}

	vm := baseVM.Fork()
	t.Cleanup(func() { _ = vm.Dispose() })
	if _, e = vm.Run(context.Background(), contents[codeStart:codeStart+codeEnd]); e != nil {
		t.Fatalf("evaluate complete example: %v", e)
	}
}
