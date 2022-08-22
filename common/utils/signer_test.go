package utils

import (
	"testing"
	"time"
)

func TestSigner(t *testing.T) {
	s := NewSigner()

	signature := s.Sign("hello world", time.Now().Add(3*time.Second))

	t.Logf("signature: %s\n", signature)

	ok := s.Validate("hello world", signature)
	if !ok {
		t.Errorf("test failed")
	}

	ok = s.Validate("HELLO WORLD", signature)
	if ok {
		t.Errorf("test failed")
	}

	time.Sleep(3 * time.Second)
	ok = s.Validate("hello world", signature)
	if ok {
		t.Errorf("test failed")
	}
}
