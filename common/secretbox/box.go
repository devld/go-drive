// Package secretbox encrypts drive secrets before they are stored.
// The key is a deployment secret, not a per-field key. Callers pass plaintext
// in and receive plaintext back; stored values carry a versioned marker.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"go-drive/common/logging"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvKey is the deployment secret. When set, it overrides the key file.
	// The value is the secret itself; the AES key is derived from it.
	EnvKey = "GO_DRIVE_ENCRYPTION_KEY"

	keyFileName = "encryption.key"
	marker      = "$go-drive-secret-v1$"

	nonceSize = 12
	tagSize   = 16
)

// minEncodedLen is the standard-base64 length of nonce + GCM tag, which is
// the shortest payload this version can produce.
const minEncodedLen = 40

// Box encrypts and decrypts individual string values.
type Box struct {
	key []byte
}

// New derives a box from an existing deployment secret.
func New(secret string) (*Box, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("encryption secret is empty")
	}
	sum := sha256.Sum256([]byte(secret))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return &Box{key: key}, nil
}

// Open loads the deployment secret. An environment variable is used when set.
// Otherwise the key file in dataDir is used, and created when it does not exist.
// The process refuses to start when both are present and different.
func Open(dataDir string) (*Box, error) {
	env := strings.TrimSpace(os.Getenv(EnvKey))
	path := filepath.Join(dataDir, keyFileName)
	fileSecret, fileErr := readSecretFile(path)
	if fileErr != nil && !errors.Is(fileErr, os.ErrNotExist) {
		return nil, fileErr
	}
	fileExists := fileErr == nil

	var (
		box *Box
		e   error
	)
	switch {
	case env != "" && fileExists && env != fileSecret:
		return nil, fmt.Errorf("encryption key in %s does not match %s", keyFileName, EnvKey)
	case env != "":
		logging.For("secretbox").Debugf("encryption key loaded from environment")
		box, e = New(env)
	case fileExists:
		logging.For("secretbox").Debugf("encryption key loaded from file")
		box, e = New(fileSecret)
	default:
		secret, writeErr := writeSecretFile(path)
		if writeErr != nil {
			return nil, writeErr
		}
		logging.For("secretbox").Infof("generated encryption key at %s", path)
		box, e = New(secret)
	}
	return box, e
}

func readSecretFile(path string) (string, error) {
	raw, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", fmt.Errorf("encryption key file %s is empty", keyFileName)
	}
	return secret, nil
}

func writeSecretFile(path string) (string, error) {
	buf := make([]byte, 32)
	if _, e := rand.Read(buf); e != nil {
		return "", e
	}
	secret := base64.StdEncoding.EncodeToString(buf)
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(e, os.ErrExist) {
		return readSecretFile(path)
	}
	if e != nil {
		return "", e
	}
	_, writeErr := io.WriteString(file, secret+"\n")
	closeErr := file.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return secret, nil
}

// IsSealed reports whether value is a complete ciphertext produced by Box.
// A short or partial marker is not enough: the whole string must match.
func IsSealed(value string) bool {
	payload, ok := strings.CutPrefix(value, marker)
	if !ok || len(payload) < minEncodedLen {
		return false
	}
	raw, e := base64.StdEncoding.DecodeString(payload)
	return e == nil && len(raw) >= nonceSize+tagSize
}

// Encrypt seals plaintext. Empty plaintext is returned unchanged.
func (b *Box) Encrypt(plaintext string) (string, error) {
	if plaintext == "" || IsSealed(plaintext) {
		return plaintext, nil
	}
	block, e := aes.NewCipher(b.key)
	if e != nil {
		return "", e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, nonceSize)
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return marker + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt returns plaintext. Values without the seal marker are returned unchanged.
// A value that matches the marker but cannot be decrypted is an error.
func (b *Box) Decrypt(value string) (string, error) {
	if !IsSealed(value) {
		return value, nil
	}
	payload, _ := strings.CutPrefix(value, marker)
	raw, e := base64.StdEncoding.DecodeString(payload)
	if e != nil || len(raw) < nonceSize+tagSize {
		return "", errors.New("invalid sealed value")
	}
	block, e := aes.NewCipher(b.key)
	if e != nil {
		return "", e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	plaintext, e := gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if e != nil {
		return "", errors.New("failed to decrypt sealed value")
	}
	return string(plaintext), nil
}
