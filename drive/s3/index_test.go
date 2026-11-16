package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/req"
	"go-drive/common/task"
	"go-drive/common/types"
)

func TestRequestHeadersApplyOnlyToServerRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("User-Agent"); got != "rclone/v1.68.0" {
			t.Errorf("User-Agent = %q", got)
		}
		if got := r.Header.Get("X-Data-Space"); got != "capsule" {
			t.Errorf("X-Data-Space = %q", got)
		}
		if authorization := strings.ToLower(r.Header.Get("Authorization")); strings.Contains(authorization, "x-data-space") {
			t.Errorf("custom header was included in the signature: %q", authorization)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	drive, e := NewDrive(context.Background(), types.SM{
		"id":         "access-key",
		"secret":     "secret-key",
		"bucket":     "bucket",
		"path_style": "1",
		"region":     "us-east-1",
		"endpoint":   server.URL,
		"request_headers": `[
			{"$key":"header","name":"User-Agent","value":"rclone/v1.68.0"},
			{"$key":"header","name":"X-Data-Space","value":"capsule"}
		]`,
	}, driveutil.DriveEnv{})
	if e != nil {
		t.Fatalf("NewDrive: %v", e)
	}

	upload, e := drive.Upload(context.Background(), "file.txt", 1, true, nil)
	if e != nil {
		t.Fatalf("Upload: %v", e)
	}
	presigned, e := url.Parse(upload.Config["url"])
	if e != nil {
		t.Fatalf("parse presigned URL: %v", e)
	}
	if signed := strings.ToLower(presigned.Query().Get("X-Amz-SignedHeaders")); strings.Contains(signed, "x-data-space") {
		t.Fatalf("custom header unexpectedly signed into direct upload URL: %q", signed)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("server requests = %d, want only the initial HeadBucket request", got)
	}
}

func TestServerRequestsUseDefaultUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	_, e := NewDrive(context.Background(), types.SM{
		"id":         "access-key",
		"secret":     "secret-key",
		"bucket":     "bucket",
		"path_style": "1",
		"region":     "us-east-1",
		"endpoint":   server.URL,
	}, driveutil.DriveEnv{})
	if e != nil {
		t.Fatalf("NewDrive: %v", e)
	}
	if got != req.DefaultUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
}

func TestNativeTransfersRejectExistingTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("conflict issued %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Last-Modified", "Mon, 19 Oct 2026 00:00:00 GMT")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	d, e := NewDrive(context.Background(), types.SM{"id": "key", "secret": "secret", "bucket": "bucket", "path_style": "1", "region": "us-east-1", "endpoint": server.URL}, driveutil.DriveEnv{})
	if e != nil {
		t.Fatal(e)
	}
	source, e := d.Get(context.Background(), "source.txt")
	if e != nil {
		t.Fatal(e)
	}
	for _, move := range []bool{false, true} {
		if move {
			_, e = d.Move(task.DummyContext(), source, "target.txt", false)
		} else {
			_, e = d.Copy(task.DummyContext(), source, "target.txt", false)
		}
		if !err.IsNotAllowedError(e) {
			t.Fatalf("move=%v conflict reported success: %v", move, e)
		}
	}
}
