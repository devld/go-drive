package ftp

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestDeadlineConnAppliesReadTimeout(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	c := &deadlineConn{Conn: client, timeout: 10 * time.Millisecond}
	started := time.Now()
	_, e := c.Read(make([]byte, 1))
	if e == nil {
		t.Fatal("expected read to time out")
	}
	var netErr net.Error
	if !errors.As(e, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", e)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("read timeout took too long: %v", elapsed)
	}
}

func TestDeadlineConnAppliesWriteTimeout(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	c := &deadlineConn{Conn: client, timeout: 10 * time.Millisecond}
	started := time.Now()
	_, e := c.Write([]byte("blocked"))
	if e == nil {
		t.Fatal("expected write to time out")
	}
	var netErr net.Error
	if !errors.As(e, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", e)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("write timeout took too long: %v", elapsed)
	}
}
