package common

import (
	"crypto"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"time"
)

type Signer struct {
	secret []byte
}

func sha256(v []byte) []byte {
	sha256 := crypto.SHA256.New()
	sha256.Write(v)
	return sha256.Sum(nil)
}

func NewSigner() *Signer {
	return &Signer{[]byte(RandString(16))}
}

func (s *Signer) sign(v string, notAfter int64, r uint32) string {
	vByte := []byte(v)
	buf := make([]byte, 4+8+len(vByte)+len(s.secret))
	binary.LittleEndian.PutUint32(buf, r)
	binary.LittleEndian.PutUint64(buf[4:], uint64(notAfter))
	copy(buf[4+8:], vByte)
	copy(buf[4+8+len(vByte):], s.secret)
	signature := sha256(buf)

	result := make([]byte, 4+8+32)
	copy(result[:], buf[:12])
	copy(result[12:], signature)

	return base64.URLEncoding.EncodeToString(result)
}

func (s *Signer) Sign(v string, notAfter time.Time) string {
	r := rand.Uint32()
	return s.sign(v, notAfter.Unix(), r)
}

func (s *Signer) Validate(v string, signature string) bool {
	buf, e := base64.URLEncoding.DecodeString(signature)
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
