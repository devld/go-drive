package archive

import (
	"archive/zip"
	"bytes"
	"go-drive/common/task"
	"go-drive/server/artifact"
	"io"
	"testing"
)

func TestProducePackIncludesOnlySelectedMembers(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	body := produceBody(t, previewer, nil, artifact.Request{
		Source: &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size},
		Args:   `pack:["docs"]`,
	})
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]struct{}{}
	for _, file := range reader.File {
		names[file.Name] = struct{}{}
		if file.Name == "README.txt" {
			t.Fatal("unselected README.txt was packed")
		}
		if file.Name == "docs/info.md" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil || string(data) != "nested content" {
				t.Fatalf("docs/info.md = %q err=%v", data, err)
			}
		}
	}
	if _, ok := names["docs/info.md"]; !ok {
		t.Fatalf("zip entries = %v", names)
	}
}

type capturingWriter struct {
	bytes.Buffer
	meta artifact.Meta
}

func (w *capturingWriter) WriteMeta(meta artifact.Meta) error {
	w.meta = meta
	return nil
}

func TestProducePackUsesPackagedFilename(t *testing.T) {
	filename, size := makeZip(t)
	previewer := newTestPreviewer(t, archiveTestConfig(), t.TempDir())
	var out capturingWriter
	err := previewer.Produce(task.DummyContext(), artifact.Request{
		Source: &testEntry{path: "demo.zip", filename: filename, seekable: true, size: size},
		Args:   `pack:["docs","README.txt"]`,
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.meta.Name != "packaged_2.zip" {
		t.Fatalf("pack name = %q, want packaged_2.zip", out.meta.Name)
	}
}
