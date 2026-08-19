package script

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"io"
)

func optionalBool(arg *Value, defaultVal bool) bool {
	if arg.IsNil() {
		return defaultVal
	}
	return arg.Bool()
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

func decodeBase64(vm *VM, s string, url, padded bool) Bytes {
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
	return NewBytes(vm, r)
}

// vm_base64Encode: (b Bytes, padded bool = true) string
func vm_base64Encode(vm *VM, args Values) any {
	return encodeBase64(GetBytes(args.Get(0).Raw()), false, optionalBool(args.Get(1), true))
}

// vm_base64Decode: (s string, padded bool = true) Bytes
func vm_base64Decode(vm *VM, args Values) any {
	return decodeBase64(vm, args.Get(0).String(), false, optionalBool(args.Get(1), true))
}

// vm_urlBase64Encode: (s Bytes, padded bool = true) string
func vm_urlBase64Encode(vm *VM, args Values) any {
	return encodeBase64(GetBytes(args.Get(0).Raw()), true, optionalBool(args.Get(1), true))
}

// vm_urlBase64Decode: (s string, padded bool = true) Bytes
func vm_urlBase64Decode(vm *VM, args Values) any {
	return decodeBase64(vm, args.Get(0).String(), true, optionalBool(args.Get(1), true))
}

const maxRandomBytes = 1 << 20

// vm_randomBytes: (n int) Bytes
func vm_randomBytes(vm *VM, args Values) any {
	n := args.Get(0).Integer()
	if n < 0 || n > maxRandomBytes {
		vm.ThrowError(errors.New("randomBytes: size must be between 0 and 1MiB"))
	}
	b := make([]byte, n)
	if n > 0 {
		if _, e := rand.Read(b); e != nil {
			vm.ThrowError(e)
		}
	}
	return NewBytes(vm, b)
}

func vm_toHex(vm *VM, args Values) any {
	return hex.EncodeToString(GetBytes(args.Get(0).Raw()))
}

func vm_fromHex(vm *VM, args Values) any {
	b, e := hex.DecodeString(args.Get(0).String())
	if e != nil {
		vm.ThrowError(e)
	}
	return NewBytes(vm, b)
}

var hashFns = map[uint8]func() hash.Hash{
	1: md5.New,
	2: sha1.New,
	3: sha256.New,
	4: sha512.New,
}

func hashFn(vm *VM, t int) func() hash.Hash {
	fn, ok := hashFns[uint8(t)]
	if !ok {
		vm.ThrowError(errors.New("unknown hash type"))
	}
	return fn
}

type Hasher struct {
	vm *VM
	s  hash.Hash
}

func (h Hasher) Write(b any) Hasher {
	_, _ = h.s.Write(bytesArg(h.vm, b, "Write requires Bytes"))
	return h
}

// WriteReader hashes from the current offset to EOF. Seekable readers (TempFile)
// are restored to that offset; they are not rewound to the start.
func (h Hasher) WriteReader(r any) Hasher {
	reader := GetReader(r)
	if reader == nil {
		h.vm.ThrowTypeError("WriteReader requires a Reader")
	}
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
	return h
}

func (h Hasher) Sum() Bytes {
	r := h.s.Sum(nil)
	return NewBytes(h.vm, r)
}

func vm_newHash(vm *VM, args Values) any {
	return Hasher{vm, hashFn(vm, int(args.Get(0).Integer()))()}
}

func vm_newHmac(vm *VM, args Values) any {
	mac := hmac.New(hashFn(vm, int(args.Get(0).Integer())), GetBytes(args.Get(1).Raw()))
	return Hasher{vm, mac}
}
