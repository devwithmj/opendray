package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sessionStore is the package-private DB layer for sessions. Manager owns
// it; subsystem callers use Manager's API instead.
type sessionStore struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *sessionStore { return &sessionStore{pool: pool} }

func (s *sessionStore) Insert(ctx context.Context, sess Session) error {
	argsJSON, err := json.Marshal(sess.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	if argsJSON == nil {
		argsJSON = []byte("[]")
	}
	_, err = s.pool.Exec(ctx, `
        INSERT INTO sessions (id, name, provider_id, cwd, args, state, pid, started_at)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)`,
		sess.ID, nullableString(sess.Name), sess.ProviderID, sess.Cwd,
		argsJSON, string(sess.State), nullableInt(sess.PID), sess.StartedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *sessionStore) Get(ctx context.Context, id string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT id, COALESCE(name, ''), provider_id, cwd, args, state,
               COALESCE(pid, 0), started_at, ended_at, exit_code
        FROM sessions WHERE id=$1`, id)
	return scanSession(row)
}

func (s *sessionStore) List(ctx context.Context) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT id, COALESCE(name, ''), provider_id, cwd, args, state,
               COALESCE(pid, 0), started_at, ended_at, exit_code
        FROM sessions ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// MarkEnded sets state=ended + ended_at + exit_code, but only if the row
// is not already ended. Idempotent against repeat exit-detector wakeups.
func (s *sessionStore) MarkEnded(ctx context.Context, id string, exitCode int) error {
	_, err := s.pool.Exec(ctx, `
        UPDATE sessions
        SET state='ended', ended_at=NOW(), exit_code=$1
        WHERE id=$2 AND state != 'ended'`, exitCode, id)
	if err != nil {
		return fmt.Errorf("mark ended: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (Session, error) {
	var (
		s        Session
		argsJSON []byte
		endedAt  sql.NullTime
		exitCode sql.NullInt32
		stateStr string
	)
	err := row.Scan(&s.ID, &s.Name, &s.ProviderID, &s.Cwd, &argsJSON,
		&stateStr, &s.PID, &s.StartedAt, &endedAt, &exitCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	s.State = State(stateStr)
	_ = json.Unmarshal(argsJSON, &s.Args)
	if endedAt.Valid {
		t := endedAt.Time
		s.EndedAt = &t
	}
	if exitCode.Valid {
		c := int(exitCode.Int32)
		s.ExitCode = &c
	}
	return s, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}
