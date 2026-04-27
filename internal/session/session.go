package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

// State enumerates the lifecycle of a session. Persisted as TEXT in
// sessions.state.
type State string

const (
	StatePending State = "pending"
	StateRunning State = "running"
	StateIdle    State = "idle"
	StateEnded   State = "ended"
)

// Session is the public view of a PTY-backed CLI session. Runtime
// resources (PTY fd, ring buffer, subscribers) live on the Manager's
// internal struct, not here.
type Session struct {
	ID         string     `json:"id"`
	Name       string     `json:"name,omitempty"`
	ProviderID string     `json:"provider_id"`
	Cwd        string     `json:"cwd"`
	Args       []string   `json:"args"`
	State      State      `json:"state"`
	PID        int        `json:"pid,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	ExitCode   *int       `json:"exit_code,omitempty"`
}

// CreateRequest is the JSON body for POST /api/v1/sessions.
type CreateRequest struct {
	Name       string   `json:"name"`
	ProviderID string   `json:"provider_id"`
	Cwd        string   `json:"cwd"`
	Args       []string `json:"args"`
}

func (r CreateRequest) Validate() error {
	if r.ProviderID == "" {
		return errors.New("provider_id is required")
	}
	if r.Cwd == "" {
		return errors.New("cwd is required")
	}
	return nil
}

// InputRequest is the JSON body for POST /api/v1/sessions/{id}/input.
// `Data` is treated as raw bytes (not base64-decoded) and sent verbatim
// to the PTY's stdin.
type InputRequest struct {
	Data string `json:"data"`
}

// ResizeRequest is the JSON body for POST /api/v1/sessions/{id}/resize.
type ResizeRequest struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// Errors used by the manager and surfaced as HTTP status codes by the
// handler layer.
var (
	ErrNotFound       = errors.New("session not found")
	ErrAlreadyEnded   = errors.New("session already ended")
	ErrUnknownProvider = errors.New("unknown provider")
)

func newID() string {
	var b [9]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand is not expected to fail; fall back to time-based
		// id to keep the system functional rather than panicking.
		t := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(t >> (i * 8))
		}
	}
	return "ses_" + base64.RawURLEncoding.EncodeToString(b[:])
}
