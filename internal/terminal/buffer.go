package terminal

import (
	"sync"
)

// DefaultRingBufferCap is 512 KB, matching the Termix scrollback replay cache.
const DefaultRingBufferCap = 512 * 1024

// RingBuffer is a thread-safe circular memory buffer for terminal output replay.
type RingBuffer struct {
	mu       sync.RWMutex
	buf      []byte
	maxBytes int
}

// NewRingBuffer allocates a ring buffer with a maximum capacity.
func NewRingBuffer(maxBytes int) *RingBuffer {
	if maxBytes <= 0 {
		maxBytes = DefaultRingBufferCap
	}
	return &RingBuffer{
		buf:      make([]byte, 0, 4096),
		maxBytes: maxBytes,
	}
}

// Write appends incoming terminal output, trimming from the front if capacity is exceeded.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// If incoming chunk itself is larger than max capacity, only keep its tail
	if len(p) >= r.maxBytes {
		r.buf = append(r.buf[:0], p[len(p)-r.maxBytes:]...)
		return len(p), nil
	}

	// Calculate excess bytes
	overflow := (len(r.buf) + len(p)) - r.maxBytes
	if overflow > 0 {
		// Shift buffer left by dropping the oldest overflow bytes
		copy(r.buf, r.buf[overflow:])
		r.buf = r.buf[:len(r.buf)-overflow]
	}

	r.buf = append(r.buf, p...)
	return len(p), nil
}

// Bytes returns a thread-safe snapshot copy of current buffer content for replay.
func (r *RingBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.buf) == 0 {
		return nil
	}
	out := make([]byte, len(r.buf))
	copy(out, r.buf)
	return out
}

// Len returns the current buffered byte count.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.buf)
}

// Reset clears the buffer content.
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = r.buf[:0]
}
