package thumbnail

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"go-drive/common/drive_util"
	"go-drive/common/types"
	"io"
	"time"
)

func init() {
	RegisterTypeHandler("text", newTextTypeHandler)
}

type textTypeHandler struct {
	fontSize  int
	imageSize int
	maxRead   int64
	padding   int
}

func newTextTypeHandler(c types.SM) (TypeHandler, error) {
	return &textTypeHandler{
		fontSize:  c.GetInt("font-size", 12),
		imageSize: c.GetInt("size", 220),
		maxRead:   c.GetInt64("max-read", 8*1024),
		padding:   c.GetInt("padding", 10),
	}, nil
}

func (t *textTypeHandler) CreateThumbnail(ctx context.Context, entry ThumbnailEntry, dest io.Writer) error {
	reader, e := drive_util.GetIContentReader(ctx, entry, -1, -1)
	if e != nil {
		return e
	}
	defer func() {
		_ = reader.Close()
	}()

	rows := (t.imageSize - 2*t.padding) / t.fontSize

	w := bufio.NewWriter(dest)
	_, e = w.WriteString(fmt.Sprintf("<svg viewBox=\"0 0 %d %d\" xmlns=\"http://www.w3.org/2000/svg\" "+
		"style=\"background-color:#fff;padding: %dpx;\">"+
		"<style>text{font-size: 10px;white-space:pre;}</style>", t.imageSize, t.imageSize, t.padding))
	if e != nil {
		return e
	}

	r := bufio.NewReader(io.LimitReader(reader, t.maxRead))

	for i := 0; i < rows; i++ {
		line, e := r.ReadString('\n')
		if e != nil {
			if e == io.EOF {
				break
			}
			return e
		}

		_, e = w.WriteString(fmt.Sprintf("<text x=\"%d\" y=\"%d\">",
			0, (i+1)*t.fontSize))

		if e != nil {
			return e
		}
		if e := xml.EscapeText(w, []byte(line)); e != nil {
			return e
		}

		_, e = w.WriteString("</text>")
		if e != nil {
			return e
		}
	}

	_, e = w.WriteString("</svg>")
	if e != nil {
		return e
	}

	return w.Flush()
}

func (t *textTypeHandler) MimeType() string {
	return "image/svg+xml"
}

func (t *textTypeHandler) Timeout() time.Duration {
	return -1
}
