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
		if authorization := strings.ToLower(r.Header.Get("Authorization")); !strings.Contains(authorization, "x-data-space") {
			t.Errorf("custom header was not included in the server request signature: %q", authorization)
		}
		if authorization := r.Header.Get("Authorization"); strings.HasPrefix(authorization, "Bearer ") {
			t.Errorf("configured Authorization was not replaced by S3 signing: %q", authorization)
		}
		if got := r.Header.Get("X-Amz-Date"); got == "ignored" || got == "" {
			t.Errorf("configured X-Amz-Date was not replaced by S3 signing: %q", got)
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
			{"$key":"header","name":"X-Data-Space","value":"capsule"},
			{"$key":"header","name":"Authorization","value":"Bearer ignored"},
			{"$key":"header","name":"X-Amz-Date","value":"ignored"}
		]`,
	}, driveutil.DriveUtils{})
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
