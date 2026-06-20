package thumbnail

import (
	"bytes"
	"encoding/base64"
	"image"
	"testing"
)

// A small lossless WebP fixture from golang.org/x/image/testdata.
const webpFixture = "UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA=="

func TestWebPDecoderRegistered(t *testing.T) {
	data, e := base64.StdEncoding.DecodeString(webpFixture)
	if e != nil {
		t.Fatal(e)
	}
	img, format, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		t.Fatalf("decode WebP: %v", e)
	}
	if format != "webp" {
		t.Fatalf("unexpected format %q", format)
	}
	if img.Bounds().Empty() {
		t.Fatal("decoded WebP has empty bounds")
	}
}

func TestResizeThumbnail(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{name: "landscape", width: 4000, height: 3000, wantWidth: 220, wantHeight: 165},
		{name: "portrait", width: 3000, height: 4000, wantWidth: 165, wantHeight: 220},
		{name: "wide", width: 1000, height: 1, wantWidth: 220, wantHeight: 1},
		{name: "small", width: 100, height: 80, wantWidth: 100, wantHeight: 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := image.NewRGBA(image.Rect(10, 20, 10+tt.width, 20+tt.height))
			got := resizeThumbnail(src, 220, 220)
			if got.Bounds().Dx() != tt.wantWidth || got.Bounds().Dy() != tt.wantHeight {
				t.Fatalf("unexpected size: got %dx%d, want %dx%d", got.Bounds().Dx(), got.Bounds().Dy(), tt.wantWidth, tt.wantHeight)
			}
			if tt.width <= 220 && tt.height <= 220 && got != src {
				t.Fatal("small image should be returned without resizing")
			}
		})
	}
}
