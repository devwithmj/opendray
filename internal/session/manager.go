package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
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
)

// Manager owns the lifecycle of all live sessions in this process.
// Sessions are persisted in postgres for visibility / audit, but the
// authoritative state for a running session is the in-memory map here.
type Manager struct {
	log       *slog.Logger
	bus       *eventbus.Hub
	store     *sessionStore
	providers ProviderResolver

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

	cmd  *exec.Cmd
	pty  *os.File
	ring *RingBuffer

	subsMu sync.Mutex
	subs   map[chan []byte]struct{}

	endOnce sync.Once
	endedCh chan struct{}
}

func NewManager(pool *pgxpool.Pool, bus *eventbus.Hub, providers ProviderResolver, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{
		log:       log.With("component", "session"),
		bus:       bus,
		store:     newStore(pool),
		providers: providers,
		sessions:  make(map[string]*runningSession),
	}
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

	args := append([]string(nil), p.Args...)
	args = append(args, req.Args...)

	cmd := exec.Command(p.Executable, args...)
	cmd.Dir = req.Cwd
	cmd.Env = os.Environ()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return Session{}, fmt.Errorf("pty.Start: %w", err)
	}

	sess := Session{
		ID:         newID(),
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
		return Session{}, err
	}

	rs := &runningSession{
		sess:    sess,
		cmd:     cmd,
		pty:     ptmx,
		ring:    NewRing(DefaultRingSize),
		subs:    make(map[chan []byte]struct{}),
		endedCh: make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sess.ID] = rs
	m.mu.Unlock()

	m.wg.Add(2)
	go m.pumpStdout(rs)
	go m.waitExit(rs)

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

func (m *Manager) pumpStdout(rs *runningSession) {
	defer m.wg.Done()
	buf := make([]byte, pumpBufSize)
	for {
		n, err := rs.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			_, _ = rs.ring.Write(chunk)
			rs.fanout(chunk)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				m.log.Debug("pty read closed", "session", rs.sess.ID, "err", err)
			}
			return
		}
	}
}

func (m *Manager) waitExit(rs *runningSession) {
	defer m.wg.Done()
	err := rs.cmd.Wait()
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	rs.endOnce.Do(func() {
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.MarkEnded(dbCtx, rs.sess.ID, exitCode); err != nil {
			m.log.Error("mark session ended", "session", rs.sess.ID, "err", err)
		}

		now := time.Now().UTC()
		ec := exitCode
		rs.sessMu.Lock()
		rs.sess.State = StateEnded
		rs.sess.EndedAt = &now
		rs.sess.ExitCode = &ec
		rs.sessMu.Unlock()

		_ = rs.pty.Close()
		rs.closeSubs()
		close(rs.endedCh)

		m.bus.Publish(eventbus.Event{
			Topic: "session.ended",
			Data: map[string]any{
				"session_id": rs.sess.ID,
				"exit_code":  exitCode,
				"ended_at":   now,
			},
		})
	})
}

func (rs *runningSession) fanout(p []byte) {
	rs.subsMu.Lock()
	defer rs.subsMu.Unlock()
	for ch := range rs.subs {
		select {
		case ch <- p:
		default:
			// slow subscriber — drop frame; eventbus.subscribers handles
			// the same pattern.
		}
	}
}

func (rs *runningSession) closeSubs() {
	rs.subsMu.Lock()
	defer rs.subsMu.Unlock()
	for ch := range rs.subs {
		close(ch)
		delete(rs.subs, ch)
	}
}

func (m *Manager) lookup(id string) *runningSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

func (m *Manager) Get(ctx context.Context, id string) (Session, error) {
	if rs := m.lookup(id); rs != nil {
		rs.sessMu.RLock()
		defer rs.sessMu.RUnlock()
		return rs.sess, nil
	}
	return m.store.Get(ctx, id)
}

func (m *Manager) List(ctx context.Context) ([]Session, error) {
	return m.store.List(ctx)
}

// Terminate sends SIGTERM and waits up to terminateGrace for the
// process to exit; if it doesn't, SIGKILL.
func (m *Manager) Terminate(ctx context.Context, id string) error {
	rs := m.lookup(id)
	if rs == nil {
		sess, err := m.store.Get(ctx, id)
		if err != nil {
			return err
		}
		if sess.State == StateEnded {
			return ErrAlreadyEnded
		}
		return fmt.Errorf("session %s not in manager", id)
	}

	rs.sessMu.RLock()
	state := rs.sess.State
	pid := rs.sess.PID
	rs.sessMu.RUnlock()
	if state == StateEnded {
		return ErrAlreadyEnded
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
func (m *Manager) Subscribe(_ context.Context, id string) (<-chan []byte, func(), error) {
	rs := m.lookup(id)
	if rs == nil {
		return nil, nil, ErrNotFound
	}
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

func (m *Manager) Buffer(_ context.Context, id string) ([]byte, error) {
	rs := m.lookup(id)
	if rs == nil {
		return nil, ErrNotFound
	}
	return rs.ring.Snapshot(), nil
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
