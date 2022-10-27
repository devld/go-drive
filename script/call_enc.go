package script

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
)

// vm_base64Encode: (b Bytes) string
func vm_base64Encode(vm *VM, args []*Value) interface{} {
	return base64.StdEncoding.EncodeToString(args[0].Raw().(Bytes).b)
}

// vm_base64Decode: (s string) Bytes
func vm_base64Decode(vm *VM, args []*Value) interface{} {
	r, e := base64.StdEncoding.DecodeString(args[0].String())
	if e != nil {
		vm.ThrowError(e)
	}
	return NewBytes(vm, r)
}

// vm_urlBase64Encode: (s Bytes) string
func vm_urlBase64Encode(vm *VM, args []*Value) interface{} {
	return base64.URLEncoding.EncodeToString(args[0].Raw().(Bytes).b)
}

// vm_urlBase64Decode: (s string) Bytes
func vm_urlBase64Decode(vm *VM, args []*Value) interface{} {
	r, e := base64.URLEncoding.DecodeString(args[0].String())
	if e != nil {
		vm.ThrowError(e)
	}
	return NewBytes(vm, r)
}

// vm_urlBase64Encode: (s Bytes) string
func vm_toHex(vm *VM, args []*Value) interface{} {
	return hex.EncodeToString(args[0].Raw().(Bytes).b)
}

// vm_urlBase64Decode: (s string) Bytes
func vm_fromHex(vm *VM, args []*Value) interface{} {
	b, e := hex.DecodeString(args[0].String())
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

type hasher struct {
	vm *VM
	s  hash.Hash
}

func (h hasher) Write(b Bytes) hasher {
	_, _ = h.s.Write(b.b)
	return h
}

func (h hasher) Sum() Bytes {
	r := h.s.Sum(nil)
	return NewBytes(h.vm, r)
}

func vm_newHash(vm *VM, args []*Value) interface{} {
	return hasher{vm, hashFn(vm, int(args[0].Integer()))()}
}

// vm_hmac: (typ int, payload, key Bytes) Bytes
func vm_hmac(vm *VM, args []*Value) interface{} {
	mac := hmac.New(hashFn(vm, int(args[0].Integer())), args[2].Raw().(Bytes).b)
	_, _ = mac.Write(args[1].Raw().(Bytes).b)
	return NewBytes(vm, mac.Sum(nil))
}
