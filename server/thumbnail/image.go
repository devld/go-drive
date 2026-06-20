package thumbnail

import (
	"context"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"time"

	xdraw "golang.org/x/image/draw"
)

func init() {
	RegisterTypeHandler("image", newImageTypeHandler)
}

type imageTypeHandler struct {
	maxSize      int64
	maxPixels    int
	imageSize    uint
	imageQuality int
}

func newImageTypeHandler(c types.SM) (TypeHandler, error) {
	return &imageTypeHandler{
		maxSize:      c.GetInt64("max-size", 32*1024*1024), // 32MB
		maxPixels:    c.GetInt("max-pixels", 6000*6000),
		imageSize:    c.GetUint("size", 220),
		imageQuality: c.GetInt("quality", 50),
	}, nil
}

func (i *imageTypeHandler) CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error {
	if entry.Size() > i.maxSize {
		return err.NewNotFoundMessageError(i18n.T("api.thumbnail.file_too_large"))
	}
	tempFile, e := drive_util.CopyIContentToTempFile(task.NewContextWrapper(ctx), entry, "")
	if e != nil {
		return e
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	imgConf, _, e := image.DecodeConfig(tempFile)
	if e != nil {
		return e
	}
	if imgConf.Width*imgConf.Height > i.maxPixels {
		return err.NewNotFoundMessageError(i18n.T("api.thumbnail.image_too_large"))
	}
	_, e = tempFile.Seek(0, 0)
	if e != nil {
		return e
	}
	img, _, e := image.Decode(tempFile)
	if e != nil {
		return e
	}
	resizedImg := resizeThumbnail(img, int(i.imageSize), int(i.imageSize))
	return jpeg.Encode(dest, resizedImg, &jpeg.Options{Quality: i.imageQuality})
}

func resizeThumbnail(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxWidth && height <= maxHeight {
		return src
	}

	newWidth, newHeight := width, height
	if newWidth > maxWidth {
		newHeight = max(1, newHeight*maxWidth/newWidth)
		newWidth = maxWidth
	}
	if newHeight > maxHeight {
		newWidth = max(1, newWidth*maxHeight/newHeight)
		newHeight = maxHeight
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)
	return dst
}

func (i *imageTypeHandler) MimeType() string {
	return "image/jpeg"
}

func (i *imageTypeHandler) Timeout() time.Duration {
	return -1
}
