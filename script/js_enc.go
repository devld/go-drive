package script

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"

	"github.com/dop251/goja"
)

var jsClassHash = JSClass{
	Name:   "Hash",
	Handle: jsObjHash{},
	Construct: func(vm *VM, args Values) any {
		return jsObjHash{ClassHost: NewClassHost(vm), s: hashFn(vm, args.Get(0).String())()}
	},
	Methods: map[string]ClassMethod{
		"write": func(vm *VM, this *Value, args Values) any {
			This[jsHash](vm, this, "Hash.write").Write(args.Get(0).Raw())
			return this
		},
		"writeFrom": func(vm *VM, this *Value, args Values) any {
			This[jsHash](vm, this, "Hash.writeFrom").WriteFrom(args.Get(0).Raw())
			return this
		},
		"sum": func(vm *VM, this *Value, _ Values) any {
			return This[jsHash](vm, this, "Hash.sum").Sum()
		},
	},
}

var jsClassHmac = JSClass{
	Name:   "Hmac",
	Parent: "Hash",
	Handle: jsObjHmac{},
	Construct: func(vm *VM, args Values) any {
		mac := hmac.New(hashFn(vm, args.Get(0).String()), GetBytes(vm, args.Get(1).Raw(), "Hmac requires Bytes"))
		return jsObjHmac{jsObjHash{ClassHost: NewClassHost(vm), s: mac}}
	},
}

const (
	bytesEncUTF8      = "utf8"
	bytesEncHex       = "hex"
	bytesEncBase64    = "base64"
	bytesEncBase64URL = "base64url"

	maxRandomBytes = 1 << 20
	maxEmptyBytes  = 32 << 20
)

func encodingString(v *Value) string {
	if v == nil || v.IsNil() {
		return ""
	}
	return v.String()
}

func paddedFromJS(v *Value) bool {
	if v == nil || v.IsNil() {
		return true
	}
	obj, ok := v.v.(*goja.Object)
	if !ok {
		return true
	}
	p := obj.Get("padded")
	if p == nil || goja.IsUndefined(p) || goja.IsNull(p) {
		return true
	}
	return p.ToBoolean()
}

func paddedFromValue(v *Value) bool {
	if v == nil || v.IsNil() {
		return true
	}
	return paddedFromJS(v)
}

func encodeBase64(b []byte, url, padded bool) string {
	switch {
	case url && padded:
		return base64.URLEncoding.EncodeToString(b)
	case url && !padded:
		return base64.RawURLEncoding.EncodeToString(b)
	case padded:
		return base64.StdEncoding.EncodeToString(b)
	default:
		return base64.RawStdEncoding.EncodeToString(b)
	}
}

func decodeBase64(vm *VM, s string, url, padded bool) []byte {
	var (
		r   []byte
		err error
	)
	switch {
	case url && padded:
		r, err = base64.URLEncoding.DecodeString(s)
	case url && !padded:
		r, err = base64.RawURLEncoding.DecodeString(s)
	case padded:
		r, err = base64.StdEncoding.DecodeString(s)
	default:
		r, err = base64.RawStdEncoding.DecodeString(s)
	}
	if err != nil {
		vm.ThrowError(err)
	}
	return r
}

func formatBytes(vm *VM, b []byte, encoding string, padded bool) string {
	switch encoding {
	case "", bytesEncUTF8, "utf-8":
		return string(b)
	case bytesEncHex:
		return hex.EncodeToString(b)
	case bytesEncBase64:
		return encodeBase64(b, false, padded)
	case bytesEncBase64URL:
		return encodeBase64(b, true, padded)
	default:
		vm.ThrowTypeError("unknown Bytes encoding: " + encoding)
		return ""
	}
}

var hashFns = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha1":   sha1.New,
	"sha256": sha256.New,
	"sha512": sha512.New,
}

func hashFn(vm *VM, name string) func() hash.Hash {
	fn, ok := hashFns[strings.ToLower(name)]
	if !ok {
		vm.ThrowTypeError("unknown hash algorithm: " + name)
	}
	return fn
}

type jsHash interface {
	Write(any)
	WriteFrom(any)
	Sum() *Value
}

type jsObjHash struct {
	ClassHost
	s hash.Hash
}

type jsObjHmac struct {
	jsObjHash
}

var (
	_ jsHash = jsObjHash{}
	_ jsHash = jsObjHmac{}
)

func (h jsObjHash) Write(b any) {
	_, _ = h.s.Write(GetBytes(h.vm, b, "Write requires Bytes"))
}

// WriteFrom hashes from the current offset to EOF. Seekable readers are
// restored to that offset, not rewound to the start.
func (h jsObjHash) WriteFrom(r any) {
	reader := GetReader(h.vm, r, "WriteFrom requires a Reader")
	var (
		seeker io.Seeker
		pos    int64
	)
	if s, ok := reader.(io.Seeker); ok {
		off, e := s.Seek(0, io.SeekCurrent)
		if e != nil {
			h.vm.ThrowError(e)
		}
		seeker, pos = s, off
	}
	if _, e := io.Copy(h.s, reader); e != nil {
		h.vm.ThrowError(e)
	}
	if seeker != nil {
		if _, e := seeker.Seek(pos, io.SeekStart); e != nil {
			h.vm.ThrowError(e)
		}
	}
}

func (h jsObjHash) Sum() *Value {
	return h.vm.NewInstance("Bytes", h.s.Sum(nil))
}

func (h jsObjHash) ConsoleString() string {
	if h.s == nil {
		return "Hash {}"
	}
	return formatGoInspect("Hash", []string{fmt.Sprintf("Size: %d", h.s.Size())}, true)
}

func (h jsObjHmac) ConsoleString() string {
	if h.s == nil {
		return "Hmac {}"
	}
	return formatGoInspect("Hmac", []string{fmt.Sprintf("Size: %d", h.s.Size())}, true)
}
