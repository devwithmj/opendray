package session

import "sync"

// RingBuffer is a fixed-capacity circular byte buffer used to hold a
// session's recent stdout for replay. Writes never block; once full,
// each new byte overwrites the oldest. The buffer is safe for
// concurrent Write/Snapshot use.
//
// `written` is a monotonic counter of total bytes ever written, used by
// M1β resume to express buffer offsets across reconnects.
type RingBuffer struct {
	mu      sync.Mutex
	buf     []byte
	cap     int
	pos     int
	full    bool
	written int64
}

func NewRing(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer{buf: make([]byte, capacity), cap: capacity}
}

// Write implements io.Writer. Always returns len(p), nil — never errors.
func (r *RingBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(p)
	r.written += int64(n)
	if n == 0 {
		return 0, nil
	}
	if n >= r.cap {
		copy(r.buf, p[n-r.cap:])
		r.pos = 0
		r.full = true
		return n, nil
	}
	end := r.pos + n
	if end <= r.cap {
		copy(r.buf[r.pos:], p)
		r.pos = end
		if r.pos == r.cap {
			r.pos = 0
			r.full = true
		}
	} else {
		first := r.cap - r.pos
		copy(r.buf[r.pos:], p[:first])
		copy(r.buf, p[first:])
		r.pos = n - first
		r.full = true
	}
	return n, nil
}

// Snapshot returns the current buffer contents in chronological order.
// The returned slice is a fresh copy — safe for the caller to retain.
func (r *RingBuffer) Snapshot() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		out := make([]byte, r.pos)
		copy(out, r.buf[:r.pos])
		return out
	}
	out := make([]byte, r.cap)
	copy(out, r.buf[r.pos:])
	copy(out[r.cap-r.pos:], r.buf[:r.pos])
	return out
}

func (r *RingBuffer) BytesWritten() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.written
}
