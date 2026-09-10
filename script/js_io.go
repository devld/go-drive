package script

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

var jsClassBytes = JSClass{
	Name:   "Bytes",
	Handle: jsObjBytes{},
	Construct: func(vm *VM, args Values) any {
		arg := args.Get(0)
		if !arg.IsNumber() {
			vm.ThrowTypeError("Bytes requires a length")
		}
		n := arg.Integer()
		if n < 0 {
			vm.ThrowTypeError("Bytes length must be non-negative")
		}
		return newEmptyBytes(vm, n)
	},
	Statics: map[string]any{
		"fromString": NativeFunction(func(vm *VM, args Values) any {
			arg := args.Get(0)
			if !arg.IsString() {
				vm.ThrowTypeError("Bytes.fromString requires a string")
			}
			return newBytes(vm, arg.String())
		}),
		"fromHex": NativeFunction(func(vm *VM, args Values) any {
			b, e := hex.DecodeString(args.Get(0).String())
			if e != nil {
				vm.ThrowError(e)
			}
			return b
		}),
		"fromBase64": NativeFunction(func(vm *VM, args Values) any {
			return decodeBase64(vm, args.Get(0).String(), false, paddedFromValue(args.Get(1)))
		}),
		"fromBase64Url": NativeFunction(func(vm *VM, args Values) any {
			return decodeBase64(vm, args.Get(0).String(), true, paddedFromValue(args.Get(1)))
		}),
		"random": NativeFunction(func(vm *VM, args Values) any {
			n := args.Get(0).Integer()
			if n < 0 || n > maxRandomBytes {
				vm.ThrowError(errors.New("Bytes.random: size must be between 0 and 1MiB"))
			}
			b := make([]byte, n)
			if n > 0 {
				if _, e := rand.Read(b); e != nil {
					vm.ThrowError(e)
				}
			}
			return b
		}),
	},
	Accepts: func(v any) bool {
		_, ok := v.([]byte)
		return ok
	},
	Wrap: func(vm *VM, v any) any { return newBytes(vm, v.([]byte)) },
	Getters: map[string]ClassMethod{
		"length": func(vm *VM, this *Value, _ Values) any {
			return This[jsBytes](vm, this, "Bytes.length").Len()
		},
	},
	Methods: map[string]ClassMethod{
		"slice": func(vm *VM, this *Value, args Values) any {
			b := This[jsBytes](vm, this, "Bytes.slice")
			start := int(args.Get(0).Integer())
			end := len(b.NativeBytes())
			if args.Len() > 1 && !args.Get(1).IsUndefined() {
				end = int(args.Get(1).Integer())
			}
			return b.Slice(start, end)
		},
		"toString": func(vm *VM, this *Value, args Values) any {
			return This[jsBytes](vm, this, "Bytes.toString").ToString(args.Get(0), args.Get(1))
		},
	},
}

var jsClassReader = JSClass{
	Name:      "Reader",
	Handle:    jsObjReader{},
	Construct: ConstructorHostOnly("Reader"),
	Accepts:   AcceptsNonNil[io.Reader](),
	Wrap:      func(vm *VM, v any) any { return newReader(vm, v.(io.Reader)) },
	Methods: map[string]ClassMethod{
		"read": func(vm *VM, this *Value, args Values) any {
			return This[jsReader](vm, this, "Reader.read").Read(args.Get(0).Raw())
		},
		"readAsString": func(vm *VM, this *Value, _ Values) any {
			return This[jsReader](vm, this, "Reader.readAsString").ReadAsString()
		},
		"limitReader": func(vm *VM, this *Value, args Values) any {
			return This[jsReader](vm, this, "Reader.limitReader").LimitReader(args.Get(0).Integer())
		},
		"withProgress": func(vm *VM, this *Value, args Values) any {
			return This[jsReader](vm, this, "Reader.withProgress").WithProgress(args.Get(0).Raw())
		},
	},
}

var jsClassReadCloser = JSClass{
	Name:      "ReadCloser",
	Parent:    "Reader",
	Handle:    jsObjReadCloser{},
	Construct: ConstructorHostOnly("ReadCloser"),
	Accepts:   AcceptsNonNil[io.ReadCloser](),
	Wrap:      func(vm *VM, v any) any { return newReadCloser(vm, v.(io.ReadCloser)) },
	Methods: map[string]ClassMethod{
		"close": func(vm *VM, this *Value, _ Values) any {
			This[jsCloser](vm, this, "ReadCloser.close").Close()
			return nil
		},
	},
}

var jsClassTempFile = JSClass{
	Name:   "TempFile",
	Parent: "ReadCloser",
	Handle: jsObjTempFile{},
	Construct: func(vm *VM, _ Values) any {
		return newTempFile(vm)
	},
	Methods: map[string]ClassMethod{
		"write": func(vm *VM, this *Value, args Values) any {
			This[jsObjTempFile](vm, this, "TempFile.write").Write(args.Get(0).Raw())
			return nil
		},
		"copyFrom": func(vm *VM, this *Value, args Values) any {
			This[jsObjTempFile](vm, this, "TempFile.copyFrom").CopyFrom(args.Get(0).Raw())
			return nil
		},
		"seekTo": func(vm *VM, this *Value, args Values) any {
			return This[jsObjTempFile](vm, this, "TempFile.seekTo").SeekTo(args.Get(0).Integer(), int(args.Get(1).Integer()))
		},
		"size": func(vm *VM, this *Value, _ Values) any {
			return This[jsObjTempFile](vm, this, "TempFile.size").Size()
		},
		"close": func(vm *VM, this *Value, _ Values) any {
			This[jsCloser](vm, this, "TempFile.close").Close()
			return nil
		},
	},
}

// Capability interfaces used by prototype methods instead of concrete handles.
type jsBytes interface {
	NativeBytes() []byte
	Len() int
	Slice(s, e int) *Value
	ToString(encoding, options *Value) string
}

type jsReader interface {
	NativeReader() io.Reader
	Read(dest any) int
	ReadAsString() string
	LimitReader(n int64) *Value
	WithProgress(reporter any) *Value
}

type jsCloser interface {
	Close()
}

type jsReadCloser interface {
	jsReader
	NativeReadCloser() io.ReadCloser
}

func GetReader(vm *VM, v any, required string) io.Reader {
	if r, ok := HostAs[jsReader](v); ok {
		if nr := r.NativeReader(); nr != nil {
			return nr
		}
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

func GetReadCloser(vm *VM, v any, required string) io.ReadCloser {
	if r, ok := HostAs[jsReadCloser](v); ok {
		if rc := r.NativeReadCloser(); rc != nil {
			return rc
		}
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

// DetachReader transfers the stream and its owning resource to the caller.
// Limited views keep the original owner, so returning one does not let the VM
// close the underlying file or response body while the caller is reading it.
func DetachReader(vm *VM, v any, required string) io.ReadCloser {
	handle := unwrapHost(v)
	r := GetReader(vm, handle, required)
	if r == nil {
		return nil
	}
	if owned, ok := handle.(interface{ readerOwner() any }); ok {
		if owner := owned.readerOwner(); owner != nil {
			handle = owner
		}
	}
	if closer := GetReadCloser(vm, handle, ""); closer != nil {
		vm.RemoveDisposable(handle)
		return &detachedReader{Reader: r, Closer: closer}
	}
	if closer, ok := r.(io.ReadCloser); ok {
		return closer
	}
	return io.NopCloser(r)
}

type detachedReader struct {
	io.Reader
	io.Closer
}

func GetBytes(vm *VM, v any, required string) []byte {
	if b, ok := HostAs[jsBytes](v); ok {
		return b.NativeBytes()
	}
	vm.throwTypeErrorRequired(required)
	return nil
}

var (
	_ jsBytes      = jsObjBytes{}
	_ jsReader     = jsObjReader{}
	_ jsReader     = jsObjReadCloser{}
	_ jsReader     = jsObjTempFile{}
	_ jsReadCloser = jsObjReadCloser{}
	_ jsReadCloser = jsObjTempFile{}
	_ jsCloser     = (*jsObjReadCloser)(nil)
	_ jsCloser     = (*jsObjTempFile)(nil)
)

type jsObjBytes struct {
	ClassHost
	b []byte
}

func (b jsObjBytes) NativeBytes() []byte {
	return b.b
}

func (b jsObjBytes) Len() int {
	return len(b.b)
}

func (b jsObjBytes) Slice(s, e int) *Value {
	n := len(b.b)
	s = clampSliceIndex(s, n)
	e = clampSliceIndex(e, n)
	if s > e {
		s = e
	}
	return b.vm.NewInstance("Bytes", b.b[s:e])
}

func clampSliceIndex(i, n int) int {
	if i < 0 {
		i += n
		if i < 0 {
			return 0
		}
	}
	if i > n {
		return n
	}
	return i
}

func (b jsObjBytes) ToString(encoding *Value, options *Value) string {
	return formatBytes(b.vm, b.b, encodingString(encoding), paddedFromJS(options))
}

func (b jsObjBytes) ConsoleString() string {
	return formatGoInspect("Bytes", []string{fmt.Sprintf("Len: %d", len(b.b))}, false)
}

type jsObjReader struct {
	ClassHost
	r     io.Reader
	owner any
}

func (r jsObjReader) readerOwner() any { return r.owner }

func (r jsObjReader) NativeReader() io.Reader {
	return r.r
}

func (r jsObjReader) Read(dest any) int {
	buf := GetBytes(r.vm, dest, "Read requires Bytes")
	n, e := r.r.Read(buf)
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

func (r jsObjReader) ReadAsString() string {
	bytes, e := io.ReadAll(r.r)
	if e != nil {
		r.vm.ThrowError(e)
	}
	return string(bytes)
}

func (r jsObjReader) LimitReader(n int64) *Value {
	view := newReader(r.vm, limitedReader{&io.LimitedReader{R: r.r, N: max(n, 0)}})
	view.owner = r.owner
	return newValue(r.vm, r.vm.instantiate(r.vm.classSet().classByName("Reader"), view))
}

func (r jsObjReader) WithProgress(reporter any) *Value {
	p := GetProgressReporter(r.vm, reporter, "Reader.withProgress requires a ProgressReporter")
	if !p.allowLoaded {
		r.vm.ThrowTypeError("Reader.withProgress requires loaded permission")
	}
	if readerHasProgress(r.r) {
		r.vm.ThrowTypeError("Reader already reports progress")
	}
	view := newReader(r.vm, progressReportingReader{r: r.r, p: p})
	view.owner = r.owner
	return newValue(r.vm, r.vm.instantiate(r.vm.classSet().classByName("Reader"), view))
}

func (r jsObjReader) ConsoleString() string {
	return formatGoInspect("Reader", nil, true)
}

type contentLengthReader interface {
	ContentLength() int64
}

type progressReportingReader struct {
	r io.Reader
	p *ProgressReporter
}

func (r progressReportingReader) Read(b []byte) (int, error) {
	n, e := r.r.Read(b)
	r.p.addLoadedFromIO(int64(n))
	return n, e
}

func (r progressReportingReader) ContentLength() int64 {
	return readerKnownLength(r.r)
}

func readerHasProgress(r io.Reader) bool {
	switch r := r.(type) {
	case progressReportingReader:
		return true
	case *progressReportingReader:
		return true
	case limitedReader:
		return readerHasProgress(r.R)
	case *limitedReader:
		return readerHasProgress(r.R)
	default:
		return false
	}
}

// limitedReader queries both the remaining limit and the underlying stream.
// Keeping the original reader preserves limits when readers are nested.
type limitedReader struct {
	*io.LimitedReader
}

func (l limitedReader) ContentLength() int64 {
	if l.N <= 0 {
		return 0
	}
	if n := readerKnownLength(l.R); n >= 0 {
		return min(n, l.N)
	}
	return -1
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

type jsObjReadCloser struct {
	jsObjReader
	r io.ReadCloser
}

func (r jsObjReadCloser) NativeReadCloser() io.ReadCloser {
	return r.r
}

func (r *jsObjReadCloser) Close() {
	r.vm.RemoveDisposable(r)
	if e := r.r.Close(); e != nil {
		r.vm.ThrowError(e)
	}
}

func (r jsObjReadCloser) ConsoleString() string {
	return formatGoInspect("ReadCloser", nil, true)
}

type jsObjTempFile struct {
	jsObjReadCloser
	f *os.File
}

func (tf jsObjTempFile) NativeReadCloser() io.ReadCloser {
	if tf.f == nil {
		return nil
	}
	return &tempFileCloser{tf.f}
}

func (tf jsObjTempFile) Write(b any) {
	_, e := tf.f.Write(GetBytes(tf.vm, b, "Write requires Bytes"))
	if e != nil {
		tf.vm.ThrowError(e)
	}
}

func (tf jsObjTempFile) CopyFrom(r any) {
	reader := GetReader(tf.vm, r, "CopyFrom requires a Reader")
	if _, e := io.Copy(tf.f, reader); e != nil {
		tf.vm.ThrowError(e)
	}
}

func (tf jsObjTempFile) SeekTo(offset int64, whence int) int64 {
	ret, e := tf.f.Seek(offset, whence)
	if e != nil {
		tf.vm.ThrowError(e)
	}
	return ret
}

func (tf jsObjTempFile) Size() int64 {
	info, e := tf.f.Stat()
	if e != nil {
		tf.vm.ThrowError(e)
	}
	return info.Size()
}

func (tf jsObjTempFile) ConsoleString() string {
	n := int64(-1)
	if info, e := tf.f.Stat(); e == nil {
		n = info.Size()
	}
	return formatGoInspect("TempFile", []string{fmt.Sprintf("Size: %d", n)}, true)
}

func (tf jsObjTempFile) close() error {
	return (&tempFileCloser{tf.f}).Close()
}

func (tf *jsObjTempFile) Close() {
	tf.vm.RemoveDisposable(tf)
	if e := tf.close(); e != nil {
		tf.vm.ThrowError(e)
	}
}

type tempFileCloser struct {
	*os.File
}

func (tfc *tempFileCloser) Close() error {
	_ = tfc.File.Close()
	return os.Remove(tfc.File.Name())
}

func newBytes(vm *VM, s any) jsObjBytes {
	switch s := s.(type) {
	case string:
		return jsObjBytes{ClassHost: NewClassHost(vm), b: []byte(s)}
	case []byte:
		return jsObjBytes{ClassHost: NewClassHost(vm), b: s}
	}
	vm.ThrowTypeError("invalid Bytes source")
	return jsObjBytes{}
}

func newEmptyBytes(vm *VM, n int64) jsObjBytes {
	if n < 0 || n > maxEmptyBytes {
		vm.ThrowTypeError("Bytes length is invalid")
	}
	return jsObjBytes{ClassHost: NewClassHost(vm), b: make([]byte, int(n))}
}

func newReader(vm *VM, r io.Reader) jsObjReader {
	return jsObjReader{ClassHost: NewClassHost(vm), r: r}
}

func newReadCloser(vm *VM, r io.ReadCloser) *jsObjReadCloser {
	rc := &jsObjReadCloser{newReader(vm, r), r}
	rc.owner = rc
	vm.PutDisposable(rc)
	return rc
}

func newTempFile(vm *VM) *jsObjTempFile {
	f, e := os.CreateTemp("", "go-drive-script-temp-")
	if e != nil {
		vm.ThrowError(e)
	}
	tf := &jsObjTempFile{jsObjReadCloser: jsObjReadCloser{newReader(vm, f), f}, f: f}
	tf.owner = tf
	vm.PutDisposable(tf)
	return tf
}
