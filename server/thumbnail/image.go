package thumbnail

import (
	"context"
	"github.com/nfnt/resize"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
)

const (
	maxSize      = 32 * 1024 * 1024 // 32MB
	maxPixels    = 6000 * 6000
	imageSize    = 220
	imageQuality = 50
)

func init() {
	h := TypeHandler{
		Create:   imageThumbnail,
		Name:     "image.jpg",
		MimeType: "image/jpeg",
	}

	Register("jpg", h)
	Register("jpeg", h)
	Register("png", h)
	Register("gif", h)
}

func imageThumbnail(ctx context.Context, entry types.IEntry, dest io.Writer) error {
	content, ok := entry.(types.IContent)
	if !ok {
		return err.NewNotFoundError()
	}
	if content.Size() > maxSize {
		return err.NewNotFoundMessageError(i18n.T("api.thumbnail.file_too_large"))
	}
	tempFile, e := drive_util.CopyIContentToTempFile(task.NewContextWrapper(ctx), content, "")
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
	if imgConf.Width*imgConf.Height > maxPixels {
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
	resizedImg := resize.Thumbnail(imageSize, imageSize, img, resize.NearestNeighbor)
	return jpeg.Encode(dest, resizedImg, &jpeg.Options{Quality: imageQuality})
}
