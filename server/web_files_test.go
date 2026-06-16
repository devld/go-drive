package server

import (
	"bytes"
	"io/fs"
	"sync"
	"testing"
	"time"
)

type fakeFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() fs.FileMode  { return 0444 }
func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f *fakeFileInfo) IsDir() bool        { return false }
func (f *fakeFileInfo) Sys() any           { return nil }

type fakeHTTPFile struct {
	*bytes.Reader
	name    string
	modTime time.Time
}

func newFakeHTTPFile(name, content string, modTime time.Time) *fakeHTTPFile {
	return &fakeHTTPFile{
		Reader:  bytes.NewReader([]byte(content)),
		name:    name,
		modTime: modTime,
	}
}

func (f *fakeHTTPFile) Close() error                       { return nil }
func (f *fakeHTTPFile) Readdir(int) ([]fs.FileInfo, error) { return nil, nil }
func (f *fakeHTTPFile) Stat() (fs.FileInfo, error) {
	return &fakeFileInfo{name: f.name, size: f.Reader.Size(), modTime: f.modTime}, nil
}

func TestTemplateProcessor_Process(t *testing.T) {
	tp := newTemplateProcessor(func(string) bool { return true })

	f := newFakeHTTPFile("index.html", "Hello {{.Name}}", time.Unix(1000, 0))
	out, e := tp.Process("/index.html", f, struct{ Name string }{"World"})
	if e != nil {
		t.Fatal(e)
	}
	if string(out) != "Hello World" {
		t.Fatalf("got %q, want %q", string(out), "Hello World")
	}
}

// TestTemplateProcessor_Concurrent runs Process concurrently with both stable
// and changing tags so that re-parses and executes overlap. Run with -race to
// catch the data race that the lock fix addresses.
func TestTemplateProcessor_Concurrent(t *testing.T) {
	tp := newTemplateProcessor(func(string) bool { return true })

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// alternate modTime to force tag changes (re-parse) on the same key
			modTime := time.Unix(int64(1000+i%2), 0)
			f := newFakeHTTPFile("index.html", "Hi {{.Name}}", modTime)
			out, e := tp.Process("/index.html", f, struct{ Name string }{"X"})
			if e != nil {
				t.Errorf("process error: %v", e)
				return
			}
			if string(out) != "Hi X" {
				t.Errorf("got %q, want %q", string(out), "Hi X")
			}
		}(i)
	}
	wg.Wait()
}
