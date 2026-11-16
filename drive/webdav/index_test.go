package webdav

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/task"
	"go-drive/common/types"
)

func TestRequestHeadersApplyToWebDAVRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "rclone/v1.68.0" {
			t.Errorf("User-Agent = %q", got)
		}
		if got := r.Header.Get("X-Data-Space"); got != "capsule" {
			t.Errorf("X-Data-Space = %q", got)
		}
		if got := r.Header.Get("Depth"); got != "9" {
			t.Errorf("Depth = %q, want configured value", got)
		}
		wantAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))
		if got := r.Header.Get("Authorization"); got != wantAuthorization {
			t.Errorf("Authorization = %q, want configured Basic credentials", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
			<multistatus xmlns="DAV:">
				<response>
					<href>/</href>
					<propstat><prop><resourcetype><collection/></resourcetype></prop></propstat>
				</response>
			</multistatus>`))
	}))
	defer server.Close()

	_, e := NewDrive(context.Background(), types.SM{
		"url":      server.URL,
		"username": "user",
		"password": "password",
		"request_headers": `[
			{"$key":"header","name":"User-Agent","value":"rclone/v1.68.0"},
			{"$key":"header","name":"X-Data-Space","value":"capsule"},
			{"$key":"header","name":"Depth","value":"9"},
			{"$key":"header","name":"Authorization","value":"Bearer ignored"}
		]`,
	}, driveutil.DriveEnv{})
	if e != nil {
		t.Fatalf("NewDrive: %v", e)
	}
}

func TestWrongPasswordIsNotSessionUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, e := NewDrive(context.Background(), types.SM{
		"url":      server.URL,
		"username": "user",
		"password": "wrong",
	}, driveutil.DriveEnv{})
	remote, ok := errors.AsType[err.RemoteAPIError](e)
	if !ok || remote.Code() != 500 || remote.Status() != http.StatusUnauthorized {
		t.Fatalf("NewDrive error = %v", e)
	}
}

func TestNativeTransfersReturnRemoteConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "COPY" || r.Method == "MOVE" {
			if r.Header.Get("Overwrite") != "F" {
				t.Errorf("Overwrite=%q", r.Header.Get("Overwrite"))
			}
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		if r.Method != "PROPFIND" {
			t.Errorf("unexpected request %s", r.Method)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:">
   <response><href>/</href><propstat><prop><resourcetype><collection/></resourcetype></prop></propstat></response>
   <response><href>/source.txt</href><propstat><prop><resourcetype/></prop></propstat></response>
   <response><href>/target.txt</href><propstat><prop><resourcetype/></prop></propstat></response>
  </multistatus>`))
	}))
	t.Cleanup(server.Close)
	d, e := NewDrive(context.Background(), types.SM{"url": server.URL}, driveutil.DriveEnv{})
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
		if e == nil {
			t.Fatalf("move=%v swallowed remote conflict", move)
		}
	}
}
