package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

	"golang.org/x/crypto/bcrypt"

	"github.com/gogi0001/family-tasks/internal/api"
	"github.com/gogi0001/family-tasks/internal/config"
	"github.com/gogi0001/family-tasks/internal/email"
	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/notify"
	"github.com/gogi0001/family-tasks/internal/reminder"
	"github.com/gogi0001/family-tasks/internal/scheduler"
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

	// --- CLI-режим: сброс пароля вручную ---
	// Если задан флаг -reset-password <email>, сервер ничего не запускает,
	// а сразу сбрасывает пароль пользователю и выходит.
	if cfg.ResetPassword != "" {
		runPasswordResetCLI(cfg)
		return
	}

	webDir, err := resolveWebDir(cfg.WebDir)
	if err != nil {
		slog.Error("cannot locate web dir", "err", err, "web_dir_flag", cfg.WebDir)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		slog.Error("create db dir", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}

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

	files, err := storage.NewFileStorage(cfg.UploadsDir)
	if err != nil {
		slog.Error("open file storage", "err", err, "path", cfg.UploadsDir)
		os.Exit(1)
	}

	users := storage.NewUsersRepo(store)
	sessions := storage.NewSessionsRepo(store)
	invites := storage.NewInvitesRepo(store)
	atts := storage.NewAttachmentRepo(store)
	templates := storage.NewTemplatesRepo(store)
	resets := storage.NewResetsRepo(store)

	mailer := email.New(email.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPass,
		From:     cfg.SMTPFrom,
	})

	ntfyClient := notify.NewClient(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyClick)
	hub := events.NewHub()

	reminderInterval, reminderWindow, err := cfg.ReminderDurations()
	if err != nil {
		slog.Error("invalid reminder config", "err", err)
		os.Exit(1)
	}
	schedulerInterval, err := cfg.SchedulerDuration()
	if err != nil {
		slog.Error("invalid scheduler config", "err", err)
		os.Exit(1)
	}

	slog.Info("starting server",
		"addr", cfg.Addr,
		"web_dir", webDir,
		"db_path", cfg.DBPath,
		"uploads_dir", cfg.UploadsDir,
		"max_upload_mb", cfg.MaxUploadMB,
		"ntfy_enabled", cfg.NtfyEnabled(),
		"email_enabled", cfg.EmailEnabled(),
		"app_url", cfg.AppURL,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reminderRunner := reminder.New(store, ntfyClient, reminderInterval, reminderWindow)
	go reminderRunner.Run(ctx)

	taskScheduler := scheduler.New(templates, store, hub, schedulerInterval)
	go taskScheduler.Run(ctx)

	// Чистка протухших сессий и reset-токенов — раз в час.
	go func() {
		t := time.NewTicker(1 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := sessions.DeleteExpiredSessions(ctx); err != nil {
					slog.Warn("cleanup sessions", "err", err)
				}
				if err := resets.DeleteExpired(ctx); err != nil {
					slog.Warn("cleanup resets", "err", err)
				}
			}
		}
	}()

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.NewRouter(api.Config{
			WebDir:         webDir,
			Tasks:          store,
			Users:          users,
			Families:       store,
			Atts:           atts,
			Files:          files,
			Sessions:       sessions,
			Invites:        invites,
			Templates:      templates,
			Resets:         resets,
			Mailer:         mailer,
			AppURL:         cfg.AppURL,
			MaxUploadBytes: cfg.MaxUploadBytes(),
			Ntfy:           ntfyClient,
			Events:         hub,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

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

// --- CLI: сброс пароля вручную ---

func runPasswordResetCLI(cfg *config.Config) {
	// Тихо открываем БД, ничего не запускаем.
	store, err := storage.OpenSQLite(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer store.Close()

	users := storage.NewUsersRepo(store)

	u, _, err := users.GetUserByEmail(context.Background(), cfg.ResetPassword)
	if err != nil {
		fmt.Fprintln(os.Stderr, "user not found:", err)
		os.Exit(1)
	}

	pwd := cfg.ResetPasswordValue
	generated := false
	if pwd == "" {
		pwd = generatePassword(16)
		generated = true
	}
	if len(pwd) < 8 {
		fmt.Fprintln(os.Stderr, "password must be at least 8 characters")
		os.Exit(2)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash:", err)
		os.Exit(1)
	}
	if err := users.UpdatePassword(context.Background(), u.ID, string(hash)); err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		os.Exit(1)
	}

	// Убиваем все сессии пользователя.
	sessions := storage.NewSessionsRepo(store)
	_ = sessions.DeleteAllUserSessions(context.Background(), u.ID)

	fmt.Printf("OK: password updated for %s\n", u.Email)
	if generated {
		fmt.Printf("New password: %s\n", pwd)
	} else {
		fmt.Println("New password: (как задано в -new-password)")
	}
}

func generatePassword(n int) string {
	// Алфавит без похожих символов.
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// Крайне маловероятно; возвращаем что-то осмысленное.
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().String()))[:n]
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

// --- helpers (без изменений) ---

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
