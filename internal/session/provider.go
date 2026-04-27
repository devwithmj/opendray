package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderInfo is the resolved exec target for a session's provider_id.
type ProviderInfo struct {
	ID         string
	Executable string
	Args       []string
}

// ProviderResolver maps a provider_id to the executable + default args
// used when spawning a session. Defined as an interface so the manager
// can be tested without DB access.
type ProviderResolver interface {
	Resolve(ctx context.Context, id string) (ProviderInfo, error)
}

// DBProviderResolver reads from the providers table populated by the
// 0002 seed migration (and, post-M2, by the catalog subsystem).
type DBProviderResolver struct{ pool *pgxpool.Pool }

func NewDBProviderResolver(pool *pgxpool.Pool) *DBProviderResolver {
	return &DBProviderResolver{pool: pool}
}

func (r *DBProviderResolver) Resolve(ctx context.Context, id string) (ProviderInfo, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT config FROM providers WHERE id=$1 AND enabled=TRUE`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderInfo{}, fmt.Errorf("%w: %s", ErrUnknownProvider, id)
	}
	if err != nil {
		return ProviderInfo{}, fmt.Errorf("provider lookup: %w", err)
	}
	var cfg struct {
		Executable string   `json:"executable"`
		Args       []string `json:"args,omitempty"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ProviderInfo{}, fmt.Errorf("provider config decode: %w", err)
	}
	if cfg.Executable == "" {
		return ProviderInfo{}, fmt.Errorf("provider %s has no executable", id)
	}
	return ProviderInfo{ID: id, Executable: cfg.Executable, Args: cfg.Args}, nil
}
