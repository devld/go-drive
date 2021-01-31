package thumbnail

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/types"
	"io"
	"strings"
)

const (
	fontSize         = 12
	textImageSize    = 220
	textMaxRead      = 8 * 1024 * 1024
	textImagePadding = 10
)

const textExt = "txt,md,xml,html,css,scss,js,json,jsx,properties,yml,yaml,ini,c,h,cpp,go,java,kt,gradle,ps1"

func init() {
	h := TypeHandler{
		Create:   textThumbnail,
		Name:     "text.svg",
		MimeType: "image/svg+xml",
	}
	for _, e := range strings.Split(textExt, ",") {
		Register(e, h)
	}
}

func textThumbnail(ctx context.Context, entry types.IEntry, dest io.Writer) error {
	content, ok := entry.(types.IContent)
	if !ok {
		return err.NewNotFoundError()
	}
	reader, e := drive_util.GetIContentReader(ctx, content)
	if e != nil {
		return e
	}
	defer func() {
		_ = reader.Close()
	}()

	rows := (textImageSize - 2*textImagePadding) / fontSize

	w := bufio.NewWriter(dest)
	_, e = w.WriteString(fmt.Sprintf("<svg viewBox=\"0 0 %d %d\" xmlns=\"http://www.w3.org/2000/svg\" "+
		"style=\"background-color:#fff;padding: %dpx;\">"+
		"<style>text{font-size: 10px;white-space:pre;}</style>", textImageSize, textImageSize, textImagePadding))
	if e != nil {
		return e
	}

	r := bufio.NewReader(io.LimitReader(reader, textMaxRead))

	for i := 0; i < rows; i++ {
		line, e := r.ReadString('\n')
		if e != nil {
			if e == io.EOF {
				break
			}
			return e
		}

		_, e = w.WriteString(fmt.Sprintf("<text x=\"%d\" y=\"%d\">",
			0, (i+1)*fontSize))

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
