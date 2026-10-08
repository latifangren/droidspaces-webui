package terminal

import (
	"bytes"
	"testing"
)

func TestRingBuffer_WriteAndBytes(t *testing.T) {
	rb := NewRingBuffer(10)

	_, _ = rb.Write([]byte("hello"))
	if !bytes.Equal(rb.Bytes(), []byte("hello")) {
		t.Fatalf("expected 'hello', got '%s'", string(rb.Bytes()))
	}

	// Append more bytes without exceeding capacity
	_, _ = rb.Write([]byte("123"))
	if !bytes.Equal(rb.Bytes(), []byte("hello123")) {
		t.Fatalf("expected 'hello123', got '%s'", string(rb.Bytes()))
	}

	// Write overflow: total length 8 + 4 = 12 > 10, should drop first 2 bytes ("he")
	_, _ = rb.Write([]byte("abcd"))
	if !bytes.Equal(rb.Bytes(), []byte("llo123abcd")) {
		t.Fatalf("expected 'llo123abcd', got '%s'", string(rb.Bytes()))
	}
}

func TestRingBuffer_LargeWrite(t *testing.T) {
	rb := NewRingBuffer(5)
	_, _ = rb.Write([]byte("0123456789"))
	if !bytes.Equal(rb.Bytes(), []byte("56789")) {
		t.Fatalf("expected '56789', got '%s'", string(rb.Bytes()))
	}
}
