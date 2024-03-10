package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"time"
)

type Signer struct {
	secret []byte
}

func sha256mac(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func NewSigner() *Signer {
	return &Signer{RandSecret(64)}
}

func (s *Signer) sign(v string, notAfter int64, r uint32) string {
	vByte := []byte(v)
	buf := make([]byte, 4+8+len(vByte))
	binary.LittleEndian.PutUint32(buf, r)
	binary.LittleEndian.PutUint64(buf[4:], uint64(notAfter))
	copy(buf[4+8:], vByte)
	signature := sha256mac(s.secret, buf)

	result := make([]byte, 4+8+32)
	copy(result[:], buf[:12])
	copy(result[12:], signature)

	return Base64URLEncode(result)
}

func (s *Signer) Sign(v string, notAfter time.Time) string {
	r := rand.Uint32()
	return s.sign(v, notAfter.Unix(), r)
}

func (s *Signer) Validate(v string, signature string) bool {
	buf, e := Base64URLDecode(signature)
	if e != nil || len(buf) != (4+8+32) {
		return false
	}
	r := binary.LittleEndian.Uint32(buf)
	notAfter := int64(binary.LittleEndian.Uint64(buf[4:]))

	actualSignature := s.sign(v, notAfter, r)
	if actualSignature != signature {
		return false
	}

	return notAfter > time.Now().Unix()
}
