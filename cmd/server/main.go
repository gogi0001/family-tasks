package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gogi0001/family-tasks/internal/api"
	"github.com/gogi0001/family-tasks/internal/config"
	"github.com/gogi0001/family-tasks/internal/storage"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(2)
	}

	setupLogger(cfg)

	// --- Поиск web/ ---
	webDir, err := resolveWebDir(cfg.WebDir)
	if err != nil {
		slog.Error("cannot locate web dir", "err", err, "web_dir_flag", cfg.WebDir)
		os.Exit(1)
	}

	// --- Подготовка каталога для БД ---
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		slog.Error("create db dir", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}

	// --- Хранилище ---
	store, err := storage.OpenSQLite(cfg.DBPath)
	if err != nil {
		slog.Error("open store", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("close store", "err", err)
		}
	}()

	slog.Info("starting server",
		"addr", cfg.Addr,
		"web_dir", webDir,
		"db_path", cfg.DBPath,
		"ntfy_enabled", cfg.NtfyEnabled(),
		"ntfy_url", cfg.NtfyURL,
		"ntfy_topic", cfg.NtfyTopic,
	)

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.NewRouter(api.Config{
			WebDir:   webDir,
			Tasks:    store,
			Users:    store,
			Families: store,
		}),
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

// --- helpers ---

func setupLogger(cfg *config.Config) {
	opts := &slog.HandlerOptions{Level: cfg.SlogLevel()}

	var h slog.Handler
	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

// resolveWebDir: если задан явно — берём как есть.
// Иначе пробуем ./web, ../../web (запуск из cmd/server), web рядом с бинарником.
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
		st, err := os.Stat(abs)
		if err == nil && st.IsDir() {
			return abs, nil
		}
	}
	return "", errors.New("no web dir found; checked: " + joinStrings(candidates, ", "))
}

func joinStrings(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
