package gomega

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const numReplications = 262144
const tokenSlotSize = 48
const doneCtxCheckWhenNthIteration = 1000

// Base64ToBytes decodes a base64url-encoded string to a byte slice
func Base64ToBytes(s string) ([]byte, error) {
	if strings.ContainsAny(s, "+/=") {
		return nil, fmt.Errorf("invalid base64url format")
	}

	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// PadToAESBlockSize ensures a byte slice is padded to AES block size (16 bytes)
func PadToAESBlockSize(data []byte) []byte {
	if rem := len(data) % 16; rem != 0 {
		padding := make([]byte, 16-rem)
		return append(data, padding...)
	}
	return data
}

// parseHashcash parses the X-Hashcash header value and returns the components
func parseHashcash(header string) (easiness int, token string, valid bool) {
	parts := strings.Split(header, ":")
	if len(parts) != 4 {
		return 0, "", false
	}

	v, err := strconv.Atoi(parts[0])
	if err != nil || v != 1 {
		return 0, "", false
	}

	e, err := strconv.Atoi(parts[1])
	if err != nil || e < 0 || e > 255 {
		return 0, "", false
	}

	return e, parts[3], true
}

// gencash generates a hashcash value based on the token and easiness
func gencash(ctx context.Context, token string, easiness int) string {
	threshold := uint32((((easiness & 63) << 1) + 1) << ((easiness>>6)*7 + 3))
	tokenBytes, err := Base64ToBytes(token)
	if err != nil {
		return ""
	}

	tokenBytes = PadToAESBlockSize(tokenBytes)
	buffer := make([]byte, 4+numReplications*tokenSlotSize) // 12 MB!

	// Replicate token data across the buffer
	for i := 0; i < numReplications; i++ {
		copy(buffer[4+i*tokenSlotSize:], tokenBytes)
	}

	prefix := make([]byte, 4)

	// Try different prefixes until we find one that satisfies the threshold
	iterations := 0
	for {
		// Check context every doneCtxCheckWhenNthIteration iterations
		if iterations++; iterations%doneCtxCheckWhenNthIteration == 0 {
			select {
			case <-ctx.Done():
				return ""
			default:
			}
		}

		// Increment prefix
		prefixSize := 4
		for j := 0; j < prefixSize; j++ {
			buffer[j]++
			if buffer[j] != 0 {
				break
			}
			// last byte overflowed to zero
			if j == prefixSize-1 {
				return ""
			}
		}

		// Save prefix for later
		copy(prefix, buffer[:4])

		hash := sha256.Sum256(buffer)
		hashValue := binary.BigEndian.Uint32(hash[:4])
		if hashValue <= threshold {
			return base64.RawURLEncoding.EncodeToString(prefix)
		}
	}
}

func solveHashCashChallenge(token string, easiness int, timeout time.Duration, workers int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan string, workers)

	workerFunc := func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result := gencash(ctx, token, easiness)
				if result != "" {
					select {
					case resultChan <- result:
						return
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}

	for i := 0; i < workers; i++ {
		go workerFunc()
	}

	select {
	case result := <-resultChan:
		return result, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
