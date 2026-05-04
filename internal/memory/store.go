package memory

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

// Scope is the visibility band a memory belongs to. The MCP
// `memory.search` tool defaults to retrieving across the calling
// session's project but a caller can override per-query.
type Scope string

const (
	ScopeSession Scope = "session"
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// Validate normalises and rejects unknown scopes. Empty input
// returns ErrInvalidScope so callers default explicitly.
func (s Scope) Validate() error {
	switch s {
	case ScopeSession, ScopeProject, ScopeGlobal:
		return nil
	}
	return ErrInvalidScope
}

var (
	// ErrInvalidScope means the caller passed something other than
	// session / project / global.
	ErrInvalidScope = errors.New("memory: invalid scope")
	// ErrNotFound means a Get or Delete by id missed.
	ErrNotFound = errors.New("memory: not found")
	// ErrDimensionMismatch means the embedder's Dimensions() doesn't
	// agree with the stored vector at insert time. Real-world cause:
	// operator changed embedder without flushing memories.
	ErrDimensionMismatch = errors.New("memory: vector dimension mismatch")
)

// Memory is a stored fact the agent decided was worth remembering.
// Public fields are JSON-tagged for the admin debug API.
type Memory struct {
	ID        string                 `json:"id"`
	Scope     Scope                  `json:"scope"`
	ScopeKey  string                 `json:"scope_key"`
	Text      string                 `json:"text"`
	Embedder  string                 `json:"embedder"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`

	// Embedding is omitted from JSON by default — vectors are noisy
	// and rarely useful in admin views. Inspector dumps include it
	// via a separate endpoint.
	Embedding []float32 `json:"-"`
}

// SearchHit is one match returned by Store.Search, paired with its
// cosine similarity score for caller-side cutoff / display.
type SearchHit struct {
	Memory     Memory  `json:"memory"`
	Similarity float32 `json:"similarity"`
}

// SearchQuery describes a similarity search. ScopeKey is required
// for session and project scope; ignored for global.
type SearchQuery struct {
	Vector   []float32
	Embedder string
	Scope    Scope
	ScopeKey string
	TopK     int
}

// InsertRequest is the shape passed to Store.Insert.
type InsertRequest struct {
	Scope     Scope
	ScopeKey  string
	Text      string
	Embedder  string
	Embedding []float32
	Metadata  map[string]interface{}
}

// Store persists memories and answers similarity queries. Both
// pgvector and chromem-go implementations satisfy this interface;
// callers (the MCP server, the admin debug handler) work against
// it without knowing which is wired up.
type Store interface {
	// Insert persists a new memory and returns the assigned id.
	Insert(ctx context.Context, req InsertRequest) (string, error)
	// Search returns the top-K most similar memories, descending
	// by cosine similarity, filtered by scope.
	Search(ctx context.Context, q SearchQuery) ([]SearchHit, error)
	// List returns memories for a scope without similarity
	// ranking — used by the admin debug UI.
	List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error)
	// Delete removes a memory by id; returns ErrNotFound when the
	// id wasn't there.
	Delete(ctx context.Context, id string) error
	// Close releases store-level resources (DB conns, files).
	// Safe to call multiple times.
	Close() error
}

// NewID returns a fresh memory id with a stable prefix. The 12-byte
// random suffix keeps collisions astronomically unlikely while
// staying short enough to surface in the admin UI.
func NewID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "mem_" + base64.RawURLEncoding.EncodeToString(b[:])
}
