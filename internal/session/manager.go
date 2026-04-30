package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	// DefaultRingSize is the per-session stdout ring buffer capacity.
	DefaultRingSize = 1 << 20 // 1 MiB
	fanoutBuffer    = 64
	pumpBufSize     = 4096
	terminateGrace  = 3 * time.Second

	defaultIdleThreshold = 30 * time.Second
	defaultIdleInterval  = 5 * time.Second
)

// ManagerOption mutates Manager defaults; pass to NewManager.
type ManagerOption func(*Manager)

// WithIdleThreshold sets how long a session must be silent before
// session.idle fires. Pass 0 to disable idle detection.
func WithIdleThreshold(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleThreshold = d }
}

// WithIdleInterval sets the idle-detector poll cadence. Lower values
// improve latency; higher values reduce wakeups.
func WithIdleInterval(d time.Duration) ManagerOption {
	return func(m *Manager) { m.idleInterval = d }
}

// Manager owns the lifecycle of all live sessions in this process.
// Sessions are persisted in postgres for visibility / audit, but the
// authoritative state for a running session is the in-memory map here.
type Manager struct {
	log       *slog.Logger
	bus       *eventbus.Hub
	store     *sessionStore
	providers ProviderResolver

	idleThreshold time.Duration
	idleInterval  time.Duration

	mu       sync.RWMutex
	closed   bool
	sessions map[string]*runningSession
	wg       sync.WaitGroup
}

// runningSession holds the runtime state for one active PTY-backed
// session. The exported view (Manager.Get returns Session) snapshots
// `sess` under sessMu.
type runningSession struct {
	sessMu sync.RWMutex
	sess   Session

	cmd     *exec.Cmd
	pty     *os.File
	ring    *RingBuffer
	tempDir string // per-session scratch dir, removed on session.ended

	subsMu sync.Mutex
	subs   map[chan []byte]struct{}

	activityMu   sync.Mutex
	lastActivity time.Time
	isIdle       bool

	endOnce sync.Once
	endedCh chan struct{}
}

// markActive records new activity and reports whether the session was
// previously idle (so the caller can flip state back to running).
func (rs *runningSession) markActive(t time.Time) bool {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	rs.lastActivity = t
	wasIdle := rs.isIdle
	rs.isIdle = false
	return wasIdle
}

// checkIdle returns true if the session has just transitioned from
// active to idle (silent for >= threshold). Returns false if already
// idle (so callers fire session.idle once per idle window) or still
// active.
func (rs *runningSession) checkIdle(now time.Time, threshold time.Duration) bool {
	rs.activityMu.Lock()
	defer rs.activityMu.Unlock()
	if rs.isIdle {
		return false
	}
	if now.Sub(rs.lastActivity) >= threshold {
		rs.isIdle = true
		return true
	}
	return false
}

func NewManager(pool *pgxpool.Pool, bus *eventbus.Hub, providers ProviderResolver, log *slog.Logger, opts ...ManagerOption) *Manager {
	if log == nil {
		log = slog.Default()
	}
	m := &Manager{
		log:           log.With("component", "session"),
		bus:           bus,
		store:         newStore(pool),
		providers:     providers,
		sessions:      make(map[string]*runningSession),
		idleThreshold: defaultIdleThreshold,
		idleInterval:  defaultIdleInterval,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Create resolves the provider, spawns a PTY, persists the row, and
// starts the stdout pump + exit detector goroutines. Returns the
// persisted Session view.
func (m *Manager) Create(ctx context.Context, req CreateRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return Session{}, err
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return Session{}, errors.New("session manager closed")
	}
	m.mu.RUnlock()

	if info, err := os.Stat(req.Cwd); err != nil {
		return Session{}, fmt.Errorf("cwd: %w", err)
	} else if !info.IsDir() {
		return Session{}, fmt.Errorf("cwd is not a directory: %s", req.Cwd)
	}

	p, err := m.providers.Resolve(ctx, req.ProviderID)
	if err != nil {
		return Session{}, err
	}

	sessID := newID()
	tempDir := filepath.Join(os.TempDir(), "opendray-sess-"+sessID)
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return Session{}, fmt.Errorf("session tempdir: %w", err)
	}

	var (
		extraArgs []string
		extraEnv  map[string]string
	)
	if p.Prepare != nil {
		out, err := p.Prepare(ctx, sessID, tempDir)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return Session{}, fmt.Errorf("provider prepare: %w", err)
		}
		extraArgs = out.Args
		extraEnv = out.Env
	}

	args := append([]string(nil), p.Args...)
	args = append(args, extraArgs...)
	args = append(args, req.Args...)

	cmd := exec.Command(p.Executable, args...)
	cmd.Dir = req.Cwd
	cmd.Env = mergeEnv(os.Environ(), extraEnv)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return Session{}, fmt.Errorf("pty.Start: %w", err)
	}

	sess := Session{
		ID:         sessID,
		Name:       req.Name,
		ProviderID: req.ProviderID,
		Cwd:        req.Cwd,
		Args:       req.Args,
		State:      StateRunning,
		PID:        cmd.Process.Pid,
		StartedAt:  time.Now().UTC(),
	}
	if sess.Args == nil {
		sess.Args = []string{}
	}

	if err := m.store.Insert(ctx, sess); err != nil {
		_ = cmd.Process.Kill()
		_ = ptmx.Close()
		_ = os.RemoveAll(tempDir)
		return Session{}, err
	}

	rs := &runningSession{
		sess:         sess,
		cmd:          cmd,
		pty:          ptmx,
		ring:         NewRing(DefaultRingSize),
		tempDir:      tempDir,
		subs:         make(map[chan []byte]struct{}),
		lastActivity: sess.StartedAt,
		endedCh:      make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sess.ID] = rs
	m.mu.Unlock()

	m.wg.Add(2)
	go m.pumpStdout(rs)
	go m.waitExit(rs)
	if m.idleThreshold > 0 {
		m.wg.Add(1)
		go m.idleWatcher(rs)
	}

	m.bus.Publish(eventbus.Event{
		Topic: "session.started",
		Data: map[string]any{
			"session_id":  sess.ID,
			"provider_id": sess.ProviderID,
			"name":        sess.Name,
		},
	})
	return sess, nil
}

func (m *Manager) lookup(id string) *runningSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// mergeEnv overlays `overrides` onto a base "K=V" slice. Keys present
// in both win for `overrides`. Used so PrepareFunc can inject env vars
// like CODEX_HOME without losing the inherited environment.
func mergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	seen := make(map[string]bool, len(overrides))
	out := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		if eq := strings.IndexByte(kv, '='); eq > 0 {
			key := kv[:eq]
			if v, ok := overrides[key]; ok {
				out = append(out, key+"="+v)
				seen[key] = true
				continue
			}
		}
		out = append(out, kv)
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	if rs := m.lookup(id); rs != nil {
		rs.sessMu.RLock()
		defer rs.sessMu.RUnlock()
		return rs.sess, nil
	}
	return m.store.Get(ctx, id)
}

// List returns persisted sessions overlaid with in-memory state for
// any session still managed in this process. Useful so /sessions
// reports `idle` even though we don't write that to the DB on every
// transition.
func (m *Manager) List(ctx context.Context) ([]Session, error) {
	list, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	inflight := make(map[string]State, len(m.sessions))
	for id, rs := range m.sessions {
		rs.sessMu.RLock()
		inflight[id] = rs.sess.State
		rs.sessMu.RUnlock()
	}
	m.mu.RUnlock()
	for i, s := range list {
		if state, ok := inflight[s.ID]; ok {
			list[i].State = state
		}
	}
	return list, nil
}

// Terminate sends SIGTERM and waits up to terminateGrace for the
// process to exit; if it doesn't, SIGKILL. For an already-ended
// session it succeeds as a no-op so callers (e.g. handler.terminate)
// can chain Delete.
func (m *Manager) Terminate(ctx context.Context, id string) error {
	rs := m.lookup(id)
	if rs == nil {
		sess, err := m.store.Get(ctx, id)
		if err != nil {
			return err
		}
		if sess.State == StateEnded {
			return nil // already ended; nothing to terminate
		}
		return fmt.Errorf("session %s not in manager", id)
	}

	rs.sessMu.RLock()
	state := rs.sess.State
	pid := rs.sess.PID
	rs.sessMu.RUnlock()
	if state == StateEnded {
		return nil
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("sigterm: %w", err)
	}

	select {
	case <-rs.endedCh:
		return nil
	case <-time.After(terminateGrace):
		_ = syscall.Kill(pid, syscall.SIGKILL)
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-rs.endedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Delete permanently removes a session row from the database and the
// in-memory map. The caller must ensure the process is no longer
// running (Terminate first); Delete on a live session refuses.
func (m *Manager) Delete(ctx context.Context, id string) error {
	if rs := m.lookup(id); rs != nil {
		rs.sessMu.RLock()
		ended := rs.sess.State == StateEnded
		rs.sessMu.RUnlock()
		if !ended {
			return fmt.Errorf("cannot delete a running session — terminate first")
		}
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
	}
	return m.store.Delete(ctx, id)
}

func (m *Manager) Input(_ context.Context, id string, data []byte) error {
	rs := m.lookup(id)
	if rs == nil {
		return ErrNotFound
	}
	rs.sessMu.RLock()
	ended := rs.sess.State == StateEnded
	rs.sessMu.RUnlock()
	if ended {
		return ErrAlreadyEnded
	}
	if _, err := rs.pty.Write(data); err != nil {
		return fmt.Errorf("pty write: %w", err)
	}
	if rs.markActive(time.Now()) {
		m.flipBackToRunning(rs)
	}
	return nil
}

func (m *Manager) Resize(_ context.Context, id string, cols, rows uint16) error {
	rs := m.lookup(id)
	if rs == nil {
		return ErrNotFound
	}
	return pty.Setsize(rs.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

// Subscribe registers a channel that receives every chunk of stdout
// written after registration. The unsub function is idempotent.
//
// Returns ErrAlreadyEnded if the session has already exited — the
// pump goroutine is gone, so a fresh subscriber would never receive
// data. Callers should fall back to Buffer() to read the ring
// snapshot instead of opening a stream.
func (m *Manager) Subscribe(_ context.Context, id string) (<-chan []byte, func(), error) {
	rs := m.lookup(id)
	if rs == nil {
		return nil, nil, ErrNotFound
	}
	select {
	case <-rs.endedCh:
		return nil, nil, ErrAlreadyEnded
	default:
	}
	rs.sessMu.RLock()
	if rs.sess.State == StateEnded {
		rs.sessMu.RUnlock()
		return nil, nil, ErrAlreadyEnded
	}
	rs.sessMu.RUnlock()

	ch := make(chan []byte, fanoutBuffer)
	rs.subsMu.Lock()
	rs.subs[ch] = struct{}{}
	rs.subsMu.Unlock()
	unsub := func() {
		rs.subsMu.Lock()
		if _, ok := rs.subs[ch]; ok {
			delete(rs.subs, ch)
			close(ch)
		}
		rs.subsMu.Unlock()
	}
	return ch, unsub, nil
}

// Buffer returns ring-buffer bytes since the caller's cursor. Pass
// since=0 to receive whatever is currently in the ring.
func (m *Manager) Buffer(_ context.Context, id string, since int64) (Replay, error) {
	rs := m.lookup(id)
	if rs == nil {
		return Replay{}, ErrNotFound
	}
	return rs.ring.SnapshotSince(since), nil
}

// Shutdown signals SIGTERM to all live sessions, waits up to 5s, then
// SIGKILL stragglers. Idempotent.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	rss := make([]*runningSession, 0, len(m.sessions))
	for _, rs := range m.sessions {
		rss = append(rss, rs)
	}
	m.mu.Unlock()

	for _, rs := range rss {
		rs.sessMu.RLock()
		pid := rs.sess.PID
		ended := rs.sess.State == StateEnded
		rs.sessMu.RUnlock()
		if !ended {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		m.log.Warn("session shutdown timed out, sending SIGKILL")
		for _, rs := range rss {
			rs.sessMu.RLock()
			pid := rs.sess.PID
			ended := rs.sess.State == StateEnded
			rs.sessMu.RUnlock()
			if !ended {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
		select {
		case <-done:
		case <-ctx.Done():
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
