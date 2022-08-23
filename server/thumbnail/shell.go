package thumbnail

import (
	"bytes"
	"context"
	"errors"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"io"
	"log"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterTypeHandler("shell", newShellThumbnailTypeHandler)
}

// shellThumbnailTypeHandler generating thumbnails by executing external command.
//
// The file content(if the entry is readable) will be written to stdin.
//
// And the thumbnail should be written to stdout.
//
// The generating failed if the command exit with non-zero code.
//
// There are some environment variables will be supplied.
//
// GO_DRIVE_ENTRY_TYPE: file|dir
//
// GO_DRIVE_ENTRY_PATH: the quoted entry path
//
// GO_DRIVE_ENTRY_NAME: the quoted entry name
//
// GO_DRIVE_ENTRY_SIZE: the entry file size
//
// GO_DRIVE_ENTRY_MOD_TIME: timestamp, modTime of this entry
type shellThumbnailTypeHandler struct {
	command string
	args    []string

	// writeContent indicates whether the file content will be supplied to stdin
	writeContent bool
	// maxSize is the maximum supported file size
	maxSize int64

	mimeType string
	timeout  time.Duration
}

func newShellThumbnailTypeHandler(c types.SM) (TypeHandler, error) {
	shell := c["shell"]
	mimeType := c["mime-type"]
	writeContent := c.GetBool("write-content")
	args := make([]string, 0)
	for _, s := range strings.Split(shell, " ") {
		s = strings.TrimSpace(s)
		if s != "" {
			args = append(args, s)
		}
	}
	if len(args) == 0 {
		return nil, errors.New("invalid command, you must specify valid 'shell'")
	}
	if mimeType == "" {
		return nil, errors.New("mime-type must be specified")
	}

	return &shellThumbnailTypeHandler{
		command:      args[0],
		args:         args[1:],
		writeContent: writeContent,
		maxSize:      c.GetInt64("max-size", -1),
		mimeType:     mimeType,
		timeout:      c.GetDuration("timeout", -1),
	}, nil
}

func (s *shellThumbnailTypeHandler) CreateThumbnail(ctx context.Context, entry types.IEntry, dest io.Writer) error {
	if s.maxSize > 0 && entry.Size() > s.maxSize {
		return err.NewNotFoundMessageError(i18n.T("api.thumbnail.file_too_large"))
	}
	cmd := exec.Command(s.command, s.args...)

	cmd.Env = append(cmd.Env, "GO_DRIVE_ENTRY_TYPE="+string(entry.Type()))
	cmd.Env = append(cmd.Env, "GO_DRIVE_ENTRY_PATH=\""+entry.Path()+"\"")
	cmd.Env = append(cmd.Env, "GO_DRIVE_ENTRY_NAME=\""+path.Base(entry.Path())+"\"")
	cmd.Env = append(cmd.Env, "GO_DRIVE_ENTRY_SIZE="+strconv.FormatInt(entry.Size(), 10))
	cmd.Env = append(cmd.Env, "GO_DRIVE_ENTRY_MOD_TIME="+strconv.FormatInt(entry.ModTime(), 10))

	if entry.Type().IsFile() && s.writeContent {
		reader, e := entry.GetReader(ctx)
		if e != nil {
			return e
		}
		defer func() { _ = reader.Close() }()
		cmd.Stdin = reader
	}

	stdErr := &bytes.Buffer{}

	cmd.Stdout = dest
	cmd.Stderr = stdErr

	c := make(chan error)
	go func() {
		c <- cmd.Run()
	}()

	var e error

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ctx.Err()
	case e = <-c:
	}

	if e != nil {
		log.Printf("shell thumbnail handler error: %v. stderr: %s", e, stdErr.String())
	}

	return e
}

func (s *shellThumbnailTypeHandler) MimeType() string {
	return s.mimeType
}

func (s *shellThumbnailTypeHandler) Timeout() time.Duration {
	return s.timeout
}
