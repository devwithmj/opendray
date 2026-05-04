package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Service wires one Embedder + one Store together. Handlers and the
// MCP server hold a *Service; they don't reach into the underlying
// pieces directly.
//
// Lifecycle: built once at app startup, lives until shutdown. Safe
// for concurrent use.
type Service struct {
	emb       Embedder
	store     Store
	threshold float32
	topK      int
	scope     ScopeDefaults
	log       *slog.Logger
}

// ScopeDefaults captures the operator's per-scope policy. These
// only set defaults — every API call can override.
type ScopeDefaults struct {
	Default       Scope
	GlobalReaders []string
}

// Options is the constructor argument bag. Everything is optional
// except Embedder, Store and Logger.
type Options struct {
	Embedder            Embedder
	Store               Store
	SimilarityThreshold float32
	DefaultTopK         int
	Scope               ScopeDefaults
	Logger              *slog.Logger
}

// New builds a Service. Sensible defaults fill in zero-valued
// options so the caller can pass a near-empty struct in tests.
func New(opts Options) (*Service, error) {
	if opts.Embedder == nil {
		return nil, errors.New("memory: Service requires an Embedder")
	}
	if opts.Store == nil {
		return nil, errors.New("memory: Service requires a Store")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.SimilarityThreshold <= 0 {
		// 0.1 is a permissive default — BM25 hash-bucket vectors
		// rarely score above ~0.3 even for clearly related text,
		// so we lean on Top-K for filtering and only use the
		// threshold to cut hits with literally zero overlap.
		// When operators wire in a dense embedder (phase 2: bge-m3)
		// they can tighten this in [memory] to 0.6+ for stricter
		// recall.
		opts.SimilarityThreshold = 0.1
	}
	if opts.DefaultTopK <= 0 {
		opts.DefaultTopK = 5
	}
	if opts.Scope.Default == "" {
		opts.Scope.Default = ScopeProject
	}
	return &Service{
		emb:       opts.Embedder,
		store:     opts.Store,
		threshold: opts.SimilarityThreshold,
		topK:      opts.DefaultTopK,
		scope:     opts.Scope,
		log:       opts.Logger.With("component", "memory"),
	}, nil
}

// Close releases the store's resources.
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// EmbedderName + StoreName are exposed for the Settings UI's
// "what's currently active?" status pane.
func (s *Service) EmbedderName() string { return s.emb.Name() }
func (s *Service) Dimensions() int      { return s.emb.Dimensions() }

// StoreRequest is the public shape callers (MCP, HTTP debug API)
// pass to Store. It mirrors InsertRequest minus the embedding
// (we compute that here).
type StoreRequest struct {
	Text     string                 `json:"text"`
	Scope    Scope                  `json:"scope"`
	ScopeKey string                 `json:"scope_key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchRequest mirrors a /memory.search tool call — text query
// plus the scope filter, no vector (we embed here).
//
// MinSimilarity overrides the service-level threshold. 0 = use
// default; explicit -1 keeps every hit regardless of score (handy
// for debugging). UI's "show all matches" toggle sends -1.
type SearchRequest struct {
	Query         string  `json:"query"`
	Scope         Scope   `json:"scope,omitempty"`
	ScopeKey      string  `json:"scope_key,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	MinSimilarity float32 `json:"min_similarity,omitempty"`
}

// Store embeds + persists a fact. Returns the stored memory id and
// the dedupe outcome ("inserted" | "merged" — current implementation
// always inserts; consolidation lands in phase 2).
func (s *Service) Store(ctx context.Context, req StoreRequest) (string, error) {
	if strings.TrimSpace(req.Text) == "" {
		return "", errors.New("memory: empty text")
	}
	if req.Scope == "" {
		req.Scope = s.scope.Default
	}
	if err := req.Scope.Validate(); err != nil {
		return "", err
	}
	if req.Scope != ScopeGlobal && strings.TrimSpace(req.ScopeKey) == "" {
		return "", fmt.Errorf("memory: scope %q requires a scope_key", req.Scope)
	}

	emb, err := s.emb.Embed(ctx, []string{req.Text})
	if err != nil {
		return "", fmt.Errorf("memory: embed for store: %w", err)
	}
	if len(emb) != 1 {
		return "", fmt.Errorf("memory: embedder returned %d vectors", len(emb))
	}

	id, err := s.store.Insert(ctx, InsertRequest{
		Scope:     req.Scope,
		ScopeKey:  req.ScopeKey,
		Text:      req.Text,
		Embedder:  s.emb.Name(),
		Embedding: emb[0],
		Metadata:  req.Metadata,
	})
	if err != nil {
		return "", err
	}
	s.log.Info("memory.store", "id", id, "scope", req.Scope, "scope_key", req.ScopeKey, "len", len(req.Text))
	return id, nil
}

// Search embeds the query and asks the store for top-K similar
// memories, then drops anything below threshold so callers see only
// likely matches.
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]SearchHit, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("memory: empty query")
	}
	if req.Scope == "" {
		req.Scope = s.scope.Default
	}
	if err := req.Scope.Validate(); err != nil {
		return nil, err
	}
	if req.Scope != ScopeGlobal && strings.TrimSpace(req.ScopeKey) == "" {
		return nil, fmt.Errorf("memory: scope %q requires a scope_key", req.Scope)
	}
	topK := req.TopK
	if topK <= 0 {
		topK = s.topK
	}

	t0 := time.Now()
	emb, err := s.emb.Embed(ctx, []string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("memory: embed for search: %w", err)
	}

	hits, err := s.store.Search(ctx, SearchQuery{
		Vector:   emb[0],
		Embedder: s.emb.Name(),
		Scope:    req.Scope,
		ScopeKey: req.ScopeKey,
		TopK:     topK,
	})
	if err != nil {
		return nil, err
	}

	threshold := s.threshold
	switch {
	case req.MinSimilarity == -1:
		threshold = -2 // never filter
	case req.MinSimilarity > 0:
		threshold = req.MinSimilarity
	}
	out := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Similarity >= threshold {
			out = append(out, h)
		}
	}
	s.log.Debug("memory.search", "query_len", len(req.Query), "hits", len(out), "kept_of", len(hits), "dur", time.Since(t0))
	return out, nil
}

// List proxies straight through. Used by the admin debug page —
// agents shouldn't need raw listing.
func (s *Service) List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error) {
	if scope == "" {
		scope = s.scope.Default
	}
	return s.store.List(ctx, scope, scopeKey, limit)
}

// Delete proxies straight through.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
