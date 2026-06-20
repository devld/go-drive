package thumbnail

import (
	"image"
	"testing"
)

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
