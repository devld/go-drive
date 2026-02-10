package script

import (
	"go-drive/common/drive_util"
	"io"
	"os"
)

func NewBytes(vm *VM, s any) Bytes {
	switch s := s.(type) {
	case string:
		return Bytes{vm, []byte(s)}
	case []byte:
		return Bytes{vm, s}
	}
	panic("invalid type for NewBytes")
}

func NewEmptyBytes(vm *VM, n int64) Bytes {
	return Bytes{vm, make([]byte, n)}
}

func NewReader(vm *VM, r io.Reader) Reader {
	return Reader{vm, r}
}

func NewReadCloser(vm *VM, r io.ReadCloser) ReadCloser {
	rc := ReadCloser{NewReader(vm, r), r}
	vm.PutDisposable(rc)
	return rc
}

func NewTempFile(vm *VM) TempFile {
	f, e := os.CreateTemp("", "go-drive-script-temp-")
	if e != nil {
		vm.ThrowError(e)
	}
	tf := TempFile{NewReader(vm, f), f}
	vm.PutDisposable(tf)
	return tf
}

func GetReader(v any) io.Reader {
	switch v := v.(type) {
	case Reader:
		return v.r
	case ReadCloser:
		return v.r
	case TempFile:
		return v.r
	}
	return nil
}

func GetReadCloser(v any) io.ReadCloser {
	switch v := v.(type) {
	case ReadCloser:
		return v.r
	case TempFile:
		return &tempFileCloser{v.f, v}
	}
	return nil
}

func GetBytes(v any) []byte {
	switch v := v.(type) {
	case Bytes:
		return v.b
	}
	return nil
}

type Bytes struct {
	vm *VM
	b  []byte
}

func (b Bytes) Len() int {
	return len(b.b)
}

func (b Bytes) Slice(s, e int) Bytes {
	return NewBytes(b.vm, b.b[s:e])
}

func (b Bytes) String() string {
	return string(b.b)
}

type Reader struct {
	vm *VM
	r  io.Reader
}

func (r Reader) Read(dest Bytes) int {
	n, e := r.r.Read(dest.b)
	if e != nil {
		if e == io.EOF {
			if n > 0 {
				return n
			}
			return -1
		}
		r.vm.ThrowError(e)
	}
	return n
}

func (r Reader) ReadAsString() string {
	bytes, e := io.ReadAll(r.r)
	if e != nil {
		r.vm.ThrowError(e)
	}
	return string(bytes)
}

func (r Reader) LimitReader(n int64) Reader {
	return NewReader(r.vm, io.LimitReader(r.r, n))
}

func (r Reader) ProgressReader(ctx any) Reader {
	return NewReader(r.vm, drive_util.ProgressReader(r.r, GetTaskCtx(ctx)))
}

type ReadCloser struct {
	Reader
	r io.ReadCloser
}

func (r ReadCloser) Close() {
	r.vm.RemoveDisposable(r)
	if e := r.r.Close(); e != nil {
		r.vm.ThrowError(e)
	}
}

type TempFile struct {
	Reader
	f *os.File
}

func (tf TempFile) Write(b Bytes) {
	_, e := tf.f.Write(b.b)
	if e != nil {
		tf.vm.ThrowError(e)
	}
}

func (tf TempFile) CopyFrom(r any) {
	reader := GetReader(r)
	if reader == nil {
		tf.vm.ThrowTypeError("CopyFrom required a Reader")
	}
	if closer, ok := reader.(io.ReadCloser); ok {
		defer func() {
			_ = closer.Close()
		}()
	}
	if _, e := io.Copy(tf.f, reader); e != nil {
		tf.vm.ThrowError(e)
	}
}

func (tf TempFile) SeekTo(offset int64, whence int) int64 {
	ret, e := tf.f.Seek(offset, whence)
	if e != nil {
		tf.vm.ThrowError(e)
	}
	return ret
}

func (tf TempFile) Size() int64 {
	info, e := tf.f.Stat()
	if e != nil {
		tf.vm.ThrowError(e)
	}
	return info.Size()
}

func (tf TempFile) close() error {
	_ = tf.f.Close()
	return os.Remove(tf.f.Name())
}

func (tf TempFile) Close() {
	tf.vm.RemoveDisposable(tf)
	if e := tf.close(); e != nil {
		tf.vm.ThrowError(e)
	}
}

type tempFileCloser struct {
	io.Reader
	tf TempFile
}

func (tfc *tempFileCloser) Close() error {
	return tfc.tf.close()
}
