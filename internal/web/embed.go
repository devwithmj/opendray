// Package web embeds the Vite-built admin SPA (app/web/) and serves
// it from the Go binary at /admin/*.
//
// Build flow:
//
//	cd app/web
//	pnpm build      # writes ../../internal/web/dist
//	cd ../..
//	go build ./cmd/opendray
//
// If the dist tree is missing (fresh checkout that hasn't run
// pnpm build), Handler returns a 503 with build instructions —
// useful so a dev-mode binary still boots, with the dev front-end
// served by `pnpm dev` at :5173 instead.
package web

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var distFS embed.FS

// Handler returns an http.Handler for the SPA. It serves static
// assets out of dist/ when the path matches a file, and falls back
// to dist/index.html for every other path so the client-side router
// (TanStack Router) can take over. Mount under /admin/ — chi.Mount
// strips the prefix before this handler runs.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return notBuiltHandler()
	}
	indexBytes, indexErr := fs.ReadFile(sub, "index.html")
	if indexErr != nil {
		return notBuiltHandler()
	}
	fileSrv := http.FileServerFS(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// chi.Mount has already stripped the mount prefix; r.URL.Path
		// here is something like "/", "/login", "/assets/index-XX.js".
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			serveIndex(w, indexBytes)
			return
		}
		// Try as a real file in dist.
		info, err := fs.Stat(sub, path)
		if err != nil || info.IsDir() {
			// SPA fallback for any unknown path.
			serveIndex(w, indexBytes)
			return
		}
		fileSrv.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func notBuiltHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(
			"opendray web bundle not built.\n" +
				"Run:\n  cd app/web && pnpm build\nthen rebuild the Go binary.\n",
		))
	})
}

// distExists reports whether the embed.FS actually carries assets.
// Useful for callers that want to log a warning at startup.
func DistExists() bool {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return false
	}
	_, err = fs.Stat(sub, "index.html")
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}
