package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"go-drive/common/utils"
	"go-drive/drive"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/types"
)

func TestCheckCopyOrMovePathBoundary(t *testing.T) {
	if e := checkCopyOrMove("drive/dir", "drive/dir/child"); e == nil {
		t.Fatal("moving to a child path should be rejected")
	}
	if e := checkCopyOrMove("drive/dir", "drive/directory/child"); e != nil {
		t.Fatalf("a segment prefix is not a child path: %v", e)
	}
}

func TestFindNonExistsEntryNameWithReservedReservesBatchNames(t *testing.T) {
	drive := &mountNameTestDrive{existing: map[string]bool{"drive/foo.txt": true}}
	reserved := map[string]bool{"drive/foo_1.txt": true}
	got, e := driveutil.FindNonExistsEntryNameWithReserved(context.Background(), drive, "drive/foo.txt", reserved)
	if e != nil {
		t.Fatalf("FindNonExistsEntryNameWithReserved: %v", e)
	}
	if got != "drive/foo_2.txt" {
		t.Fatalf("path=%q, want drive/foo_2.txt", got)
	}
}

func TestDriveUploaderName(t *testing.T) {
	ok := []string{"qiniu", "my-drive", "a_b"}
	for _, name := range ok {
		if !driveUploaderNameRegexp.MatchString(name) {
			t.Fatalf("%q should be allowed", name)
		}
	}
	bad := []string{"", ".js", "foo.js", "foo.JS", "../qiniu", "a/b", "qiniu.js"}
	for _, name := range bad {
		if driveUploaderNameRegexp.MatchString(name) {
			t.Fatalf("%q should be rejected", name)
		}
	}
}

type mountNameTestDrive struct {
	existing map[string]bool
}

func (d *mountNameTestDrive) Meta(context.Context) (types.DriveMeta, error) {
	return types.DriveMeta{}, nil
}
func (d *mountNameTestDrive) Get(_ context.Context, path string) (types.IEntry, error) {
	if d.existing[path] {
		return &mountNameTestEntry{path: path, drive: d}, nil
	}
	return nil, err.NewNotFoundError()
}
func (d *mountNameTestDrive) Save(types.TaskCtx, string, int64, bool, io.Reader) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) MakeDir(context.Context, string) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) Copy(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) Move(types.TaskCtx, types.IEntry, string, bool) (types.IEntry, error) {
	panic("not used")
}
func (d *mountNameTestDrive) List(_ context.Context, path string) ([]types.IEntry, error) {
	entries := make([]types.IEntry, 0)
	for existing := range d.existing {
		if existing != path {
			entries = append(entries, &mountNameTestEntry{path: existing, drive: d})
		}
	}
	return entries, nil
}
func (d *mountNameTestDrive) Delete(types.TaskCtx, string) error { panic("not used") }
func (d *mountNameTestDrive) Upload(context.Context, string, int64, bool, types.SM) (*types.DriveUploadConfig, error) {
	panic("not used")
}

type mountNameTestEntry struct {
	path  string
	drive types.IDrive
}

func (e *mountNameTestEntry) Path() string          { return e.path }
func (e *mountNameTestEntry) Type() types.EntryType { return types.TypeFile }
func (e *mountNameTestEntry) Size() int64           { return 0 }
func (e *mountNameTestEntry) Meta() types.EntryMeta { return types.EntryMeta{} }
func (e *mountNameTestEntry) ModTime() int64        { return 0 }
func (e *mountNameTestEntry) Drive() types.IDrive   { return e.drive }
func (e *mountNameTestEntry) GetReader(context.Context, types.ReaderRange) (io.ReadCloser, error) {
	panic("not used")
}
func (e *mountNameTestEntry) GetURL(context.Context) (*types.ContentURL, error) {
	panic("not used")
}

func TestUploadPreparesOnceAndAuthorizesBeforeProvider(t *testing.T) {
	for _, deny := range []bool{false, true} {
		t.Run(map[bool]string{false: "allowed", true: "denied"}[deny], func(t *testing.T) {
			provider := &recordingUploadDrive{mountNameTestDrive: &mountNameTestDrive{existing: map[string]bool{"file.txt": true}}}
			r, forbidden := "", "file_1.txt"
			rules := []types.PathPermission{{Path: &r, Subject: types.AnySubject, Permission: types.PermissionReadWrite, Policy: types.PolicyAccept}}
			if deny {
				rules = append(rules, types.PathPermission{Path: &forbidden, Subject: types.AnySubject, Permission: types.PermissionWrite, Policy: types.PolicyReject})
			}
			d := drive.NewPermissionWrapperDrive(provider, utils.NewPermMap(rules))
			router := gin.New()
			route := driveRoute{}
			router.POST("/upload", func(c *gin.Context) {
				c.Set("drive", d)
				route.upload(c)
				if len(c.Errors) > 0 {
					c.JSON(http.StatusForbidden, gin.H{"error": c.Errors.Last().Error()})
				} else if result, ok := getResult(c); ok {
					c.JSON(http.StatusOK, result)
				}
			})
			request := func(url, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				return w
			}
			w := request("/upload?path=file.txt&size=1", "{}")
			if deny {
				if w.Code != http.StatusForbidden || len(provider.paths) != 0 {
					t.Fatalf("unauthorized provider called: status=%d paths=%v", w.Code, provider.paths)
				}
				return
			}
			if w.Code != http.StatusOK || len(provider.paths) != 1 || provider.paths[0] != "file_1.txt" || !strings.Contains(w.Body.String(), `"path":"file_1.txt"`) {
				t.Fatalf("initial upload: status=%d paths=%v response=%s", w.Code, provider.paths, w.Body.String())
			}
			provider.existing["file_1.txt"] = true
			w = request("/upload?path=file_1.txt&size=1&override=true", `{"action":"CompleteUpload"}`)
			if w.Code != http.StatusOK || len(provider.paths) != 2 || provider.paths[1] != "file_1.txt" {
				t.Fatalf("callback changed target: status=%d paths=%v", w.Code, provider.paths)
			}
		})
	}
}

type recordingUploadDrive struct {
	*mountNameTestDrive
	paths []string
}

func (d *recordingUploadDrive) Upload(_ context.Context, path string, _ int64, _ bool, _ types.SM) (*types.DriveUploadConfig, error) {
	d.paths = append(d.paths, path)
	return &types.DriveUploadConfig{Provider: types.S3Provider, Config: types.SM{"url": "https://upload.example.invalid/fixed-target"}}, nil
}
