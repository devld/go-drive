package script

import (
	"go-drive/common/driveutil"
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

// DetachReadCloser returns the underlying io.ReadCloser of a script reader
// value and detaches it from the VM's disposables. After detaching, the reader
// will NOT be closed when the VM is passivated/returned to the pool, so the
// returned reader stays valid even though the VM is reused. The caller takes
// ownership and MUST Close it. Returns nil if v is not a closable reader
// (ReadCloser or TempFile).
func DetachReadCloser(vm *VM, v any) io.ReadCloser {
	rc := GetReadCloser(v)
	if rc == nil {
		return nil
	}
	// v is the same ReadCloser/TempFile value that was registered via
	// PutDisposable, so removing it here prevents DisposeDisposables from
	// closing the resource out from under the caller.
	vm.RemoveDisposable(v)
	return rc
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
	limited := io.LimitReader(r.r, n)
	if f, ok := underlyingFile(r.r); ok {
		return NewReader(r.vm, limitedFileReader{r: limited, f: f, limit: n})
	}
	if k := readerKnownLength(r.r); k >= 0 {
		if k > n {
			k = n
		}
		return NewReader(r.vm, fixedLengthReader{r: limited, n: k})
	}
	return NewReader(r.vm, limited)
}

func (r Reader) ProgressReader(ctx any) Reader {
	return wrapPreservingLength(r.vm, driveutil.ProgressReader(r.r, GetTaskCtx(ctx)), r.r)
}

type contentLengthReader interface {
	ContentLength() int64
}

type fixedLengthReader struct {
	r io.Reader
	n int64
}

func (f fixedLengthReader) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f fixedLengthReader) ContentLength() int64 {
	return f.n
}

type sizedFileReader struct {
	r io.Reader
	f *os.File
}

func (s sizedFileReader) Read(p []byte) (int, error) {
	return s.r.Read(p)
}

func (s sizedFileReader) ContentLength() int64 {
	return remainingFileSize(s.f)
}

type limitedFileReader struct {
	r     io.Reader
	f     *os.File
	limit int64
}

func (l limitedFileReader) Read(p []byte) (int, error) {
	return l.r.Read(p)
}

func (l limitedFileReader) ContentLength() int64 {
	rem := remainingFileSize(l.f)
	if rem < 0 {
		return -1
	}
	if rem > l.limit {
		return l.limit
	}
	return rem
}

func wrapPreservingLength(vm *VM, wrapped, orig io.Reader) Reader {
	if f, ok := underlyingFile(orig); ok {
		return NewReader(vm, sizedFileReader{r: wrapped, f: f})
	}
	if n := readerKnownLength(orig); n >= 0 {
		return NewReader(vm, fixedLengthReader{r: wrapped, n: n})
	}
	return NewReader(vm, wrapped)
}

func underlyingFile(r io.Reader) (*os.File, bool) {
	if f, ok := r.(*os.File); ok {
		return f, true
	}
	if s, ok := r.(sizedFileReader); ok {
		return s.f, true
	}
	if l, ok := r.(limitedFileReader); ok {
		return l.f, true
	}
	return nil, false
}

func remainingFileSize(f *os.File) int64 {
	off, e := f.Seek(0, io.SeekCurrent)
	if e != nil {
		return -1
	}
	info, e := f.Stat()
	if e != nil {
		return -1
	}
	n := info.Size() - off
	if n < 0 {
		return 0
	}
	return n
}

func readerKnownLength(r io.Reader) int64 {
	if r == nil {
		return -1
	}
	if cl, ok := r.(contentLengthReader); ok {
		return cl.ContentLength()
	}
	if f, ok := r.(*os.File); ok {
		return remainingFileSize(f)
	}
	return -1
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
