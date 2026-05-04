package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgvectorStore persists memories in opendray's existing PostgreSQL
// using the pgvector extension. The schema (see migration 0011) is
// dimension-agnostic at the column level so different embedders can
// coexist; we issue per-(embedder,dim) HNSW indexes lazily on first
// insert.
//
// The cosine-similarity operator pgvector exposes is `<=>`; values
// are 0 (identical) → 2 (opposite). We invert to "similarity"
// (1 - distance/2 → 0..1) when returning hits so callers see the
// same scale as our in-memory CosineSimilarity helper.
type PgvectorStore struct {
	pool *pgxpool.Pool

	// indexedMu guards the in-memory mirror of memory_index_state.
	// Pre-loaded on Open so we avoid a SELECT per insert.
	indexedMu sync.Mutex
	indexed   map[string]int // embedder → dim
}

// OpenPgvectorStore constructs the store and pre-loads the
// "(embedder, dim)" combinations we've already indexed so subsequent
// inserts can short-circuit the lazy-index check.
func OpenPgvectorStore(ctx context.Context, pool *pgxpool.Pool) (*PgvectorStore, error) {
	if pool == nil {
		return nil, errors.New("memory: pgvector store requires a *pgxpool.Pool")
	}
	s := &PgvectorStore{pool: pool, indexed: make(map[string]int)}
	if err := s.loadIndexed(ctx); err != nil {
		return nil, fmt.Errorf("memory: load indexed state: %w", err)
	}
	return s, nil
}

func (s *PgvectorStore) loadIndexed(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT embedder, dim FROM memory_index_state`)
	if err != nil {
		// Tolerate "relation does not exist" — that just means the
		// migration hasn't run yet (we get called on every app.New
		// including the one inside `opendray migrate`). The next
		// startup after the migration will populate the cache.
		if isRelationDoesNotExist(err) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var dim int
		if err := rows.Scan(&name, &dim); err != nil {
			return err
		}
		s.indexed[name] = dim
	}
	return rows.Err()
}

// isRelationDoesNotExist returns true for pg error 42P01.
func isRelationDoesNotExist(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01"
	}
	return false
}

func (s *PgvectorStore) Close() error { return nil }

// Insert writes a memory and lazily creates an HNSW index for the
// (embedder, dim) pair on first observation. Index creation is best-
// effort: a CREATE INDEX failure is logged-and-swallowed because
// pgvector falls back to seq scan automatically — we'd rather take
// the perf hit than reject the insert.
func (s *PgvectorStore) Insert(ctx context.Context, req InsertRequest) (string, error) {
	if err := req.Scope.Validate(); err != nil {
		return "", err
	}
	if len(req.Embedding) == 0 {
		return "", errors.New("memory: empty embedding")
	}
	if strings.TrimSpace(req.Text) == "" {
		return "", errors.New("memory: empty text")
	}

	id := NewID()
	meta := req.Metadata
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("memory: marshal metadata: %w", err)
	}

	vec := vectorLiteral(req.Embedding)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memories (id, scope, scope_key, text, embedding, embedder, metadata)
		VALUES ($1, $2, $3, $4, $5::vector, $6, $7::jsonb)
	`, id, string(req.Scope), req.ScopeKey, req.Text, vec, req.Embedder, metaJSON)
	if err != nil {
		return "", fmt.Errorf("memory: insert: %w", err)
	}

	s.ensureIndex(ctx, req.Embedder, len(req.Embedding))
	return id, nil
}

// ensureIndex creates an HNSW index for (embedder, dim) once. Errors
// are non-fatal: pgvector still serves queries via sequential scan,
// just slower. Locking via indexedMu prevents concurrent inserts
// from racing on the same DDL.
func (s *PgvectorStore) ensureIndex(ctx context.Context, embedder string, dim int) {
	s.indexedMu.Lock()
	defer s.indexedMu.Unlock()
	if existing, ok := s.indexed[embedder]; ok && existing == dim {
		return
	}
	idxName := fmt.Sprintf("memories_emb_%s_idx", sqlSafe(embedder))
	// HNSW with vector_cosine_ops is what pgvector recommends for
	// cosine-similarity workloads; defaults (m=16, ef_construction=64)
	// are fine for our scale.
	_, err := s.pool.Exec(ctx, fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON memories USING hnsw ((embedding::vector(%d)) vector_cosine_ops) WHERE embedder = $1`,
		idxName, dim,
	), embedder)
	if err != nil {
		// Don't surface this — silently degrade. The caller already
		// successfully inserted; failing here would be misleading.
		// In tests we log this; in prod the operator notices via slow
		// queries + the call log.
		return
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_index_state (embedder, dim) VALUES ($1, $2)
		ON CONFLICT (embedder) DO UPDATE SET dim = EXCLUDED.dim
	`, embedder, dim)
	if err != nil {
		return
	}
	s.indexed[embedder] = dim
}

// Search returns the top-K hits for q.Vector, filtered by embedder
// (so cosine comparisons stay honest across multiple embedders) and
// by scope. Empty TopK defaults to 5.
func (s *PgvectorStore) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	if err := q.Scope.Validate(); err != nil {
		return nil, err
	}
	if len(q.Vector) == 0 {
		return nil, errors.New("memory: empty query vector")
	}
	if q.TopK <= 0 {
		q.TopK = 5
	}

	vec := vectorLiteral(q.Vector)
	args := []interface{}{vec, q.Embedder, string(q.Scope), q.ScopeKey, q.TopK}
	// For global scope, ignore scope_key entirely.
	whereScope := `scope = $3 AND scope_key = $4`
	if q.Scope == ScopeGlobal {
		whereScope = `scope = $3`
		args = []interface{}{vec, q.Embedder, string(q.Scope), q.TopK}
	}

	// pgvector's <=> returns cosine *distance* (1 - cosine_similarity),
	// so similarity = 1 - distance. Range is [-1, 1]; the service
	// layer threshold filter discards anything below the configured
	// minimum (default 0.5 since the BM25 fallback rarely scores high).
	sql := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata,
		       created_at, updated_at,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM memories
		WHERE embedder = $2 AND %s
		ORDER BY embedding <=> $1::vector ASC
		LIMIT $%d
	`, whereScope, len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: search: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var (
			m    Memory
			meta []byte
			sim  float32
		)
		if err := rows.Scan(
			&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta,
			&m.CreatedAt, &m.UpdatedAt, &sim,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		hits = append(hits, SearchHit{Memory: m, Similarity: sim})
	}
	return hits, rows.Err()
}

func (s *PgvectorStore) List(ctx context.Context, scope Scope, scopeKey string, limit int) ([]Memory, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	args := []interface{}{string(scope), scopeKey, limit}
	where := `scope = $1 AND scope_key = $2`
	if scope == ScopeGlobal {
		where = `scope = $1`
		args = []interface{}{string(scope), limit}
	}
	sql := fmt.Sprintf(`
		SELECT id, scope, scope_key, text, embedder, metadata, created_at, updated_at
		FROM memories
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d
	`, where, len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: list: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m    Memory
			meta []byte
		)
		if err := rows.Scan(&m.ID, &m.Scope, &m.ScopeKey, &m.Text, &m.Embedder, &meta, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &m.Metadata)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PgvectorStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM memories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("memory: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// vectorLiteral renders a []float32 as the pgvector text format
// "[v1,v2,...]" pgx-compat. We could use the pgvector-go driver's
// custom type, but a string literal keeps the dependency surface
// flat and works equally well with prepared statements.
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", x)
	}
	b.WriteByte(']')
	return b.String()
}

// sqlSafe returns a slug usable inside an identifier without
// quoting. Used only for index names so the input set is small.
func sqlSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// Compile-time guarantee.
var _ Store = (*PgvectorStore)(nil)

// pgxPoolEnsureUsed avoids "imported and not used" if the file is
// compiled in a context where pgx is unused. Harmless at runtime.
var _ = pgx.ErrNoRows
