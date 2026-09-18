package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gogi0001/family-tasks/internal/api"
	"github.com/gogi0001/family-tasks/internal/storage"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	addr := envOr("ADDR", ":8787")
	dbPath := envOr("DB_PATH", filepath.Join("data", "tasks.db"))

	webDir, err := resolveWebDir(envOr("WEB_DIR", ""))
	if err != nil {
		slog.Error("cannot locate web dir", "err", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		slog.Error("create db dir", "err", err, "path", dbPath)
		os.Exit(1)
	}

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		slog.Error("open store", "err", err, "path", dbPath)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("close store", "err", err)
		}
	}()

	slog.Info("starting server", "addr", addr, "web_dir", webDir, "db_path", dbPath)

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(api.Config{WebDir: webDir, Store: store}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutdown signal received")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
	slog.Info("stopped")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// resolveWebDir — как в прошлом шаге, не менялся.
func resolveWebDir(explicit string) (string, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	} else {
		candidates = append(candidates, "web", filepath.Join("..", "..", "web"))
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "web"))
		}
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			return abs, nil
		}
	}
	return "", errors.New("no web dir found")
}
