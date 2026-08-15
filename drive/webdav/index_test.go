package webdav

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-drive/common/driveutil"
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
	}, driveutil.DriveUtils{})
	if e != nil {
		t.Fatalf("NewDrive: %v", e)
	}
}
