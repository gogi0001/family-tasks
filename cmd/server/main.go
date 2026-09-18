package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gogi0001/family-tasks/internal/api"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	addr := envOr("ADDR", ":8787")

	webDir, err := resolveWebDir(envOr("WEB_DIR", ""))
	if err != nil {
		slog.Error("cannot locate web dir", "err", err)
		os.Exit(1)
	}
	slog.Info("starting server", "addr", addr, "web_dir", webDir)

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(api.Config{WebDir: webDir}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// resolveWebDir:
//  1. Если задан WEB_DIR — используем его как есть (с абсолютным путём).
//  2. Иначе пробуем по очереди: ./web, ./../../web (запуск из cmd/server),
//     рядом с исполняемым файлом.
//
// Первая существующая директория выигрывает.
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
	return "", errors.New("no web dir found; checked: " + joinPaths(candidates))
}

func joinPaths(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
