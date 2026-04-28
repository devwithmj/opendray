// Package app is opendray's composition root.
//
// All subsystems (config -> store -> eventbus -> session -> gateway) are
// constructed here. Subsystem packages must not import each other through
// globals; dependencies flow only via constructor parameters wired in
// this package.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/audit"
	"github.com/opendray/opendray-v2/internal/catalog"
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/gateway"
	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/store"
	"github.com/opendray/opendray-v2/internal/version"
)

type App struct {
	cfg      config.Config
	log      *slog.Logger
	store    *store.Store
	bus      *eventbus.Hub
	sessions *session.Manager
	audit    *audit.Sink
	server   *http.Server
}

// New wires the runtime dependencies but does not start any goroutines.
// Caller is responsible for calling Run or Close.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log := newLogger(cfg.Log)
	st, err := store.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	bus := eventbus.New(log)

	cat, err := catalog.New(st.Pool(), log)
	if err != nil {
		st.Close()
		return nil, err
	}
	if err := cat.Sync(ctx); err != nil {
		st.Close()
		return nil, err
	}
	catalogHandlers := catalog.NewHandlers(cat, log)

	var sessionOpts []session.ManagerOption
	if d := cfg.Session.Threshold(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleThreshold(d))
	}
	if d := cfg.Session.Interval(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleInterval(d))
	}
	sessionMgr := session.NewManager(
		st.Pool(),
		bus,
		catalog.NewSessionProvider(cat),
		log,
		sessionOpts...,
	)
	sessionHandlers := session.NewHandlers(sessionMgr, log)
	auditSink := audit.NewSink(st.Pool(), bus, log)

	gw := gateway.NewServer(gateway.Deps{
		Logger:    log,
		DB:        st,
		Version:   version.Current(),
		StartedAt: time.Now(),
		V1Routes: func(r chi.Router) {
			sessionHandlers.Mount(r)
			catalogHandlers.Mount(r)
		},
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           gw.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:      cfg,
		log:      log,
		store:    st,
		bus:      bus,
		sessions: sessionMgr,
		audit:    auditSink,
		server:   srv,
	}, nil
}

// Migrate applies pending DB migrations and returns. Used by `opendray migrate`.
func (a *App) Migrate(ctx context.Context) error {
	return a.store.Migrate(ctx, a.log)
}

// Run starts the HTTP server + audit sink and blocks until ctx is
// cancelled, then performs graceful shutdown in this order:
//
//	HTTP server -> session manager -> audit sink -> event bus -> store
func (a *App) Run(ctx context.Context) error {
	a.log.Info("opendray starting",
		"listen", a.cfg.Listen,
		"version", version.Version,
		"commit", version.Commit)

	auditDone := make(chan struct{})
	go func() {
		a.audit.Run(ctx)
		close(auditDone)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http shutdown", "err", err)
	}
	if err := a.sessions.Shutdown(shutdownCtx); err != nil {
		a.log.Error("session shutdown", "err", err)
	}

	select {
	case <-auditDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("audit shutdown timed out")
	}

	a.bus.Close()
	a.store.Close()
	a.log.Info("opendray stopped")
	return nil
}

func (a *App) Logger() *slog.Logger { return a.log }

// Close releases resources without waiting on the HTTP server. Use Run for
// the normal lifecycle; Close is for failure paths after New succeeded.
func (a *App) Close() {
	if a.sessions != nil {
		_ = a.sessions.Shutdown(context.Background())
	}
	if a.bus != nil {
		a.bus.Close()
	}
	if a.store != nil {
		a.store.Close()
	}
}
