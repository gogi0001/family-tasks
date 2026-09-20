package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Config — все настройки сервера.
// Собирается из флагов, переменных окружения и дефолтов (в этом приоритете).
type Config struct {
	Addr      string // ":8787"
	WebDir    string // путь к web/, если пусто — ищется автоматически
	DBPath    string // "data/tasks.db"
	NtfyURL   string // "http://localhost:8080", пусто = уведомления выключены
	NtfyTopic string // "family-tasks-home"
	LogLevel  string // debug|info|warn|error
	LogFormat string // text|json
}

// Parse читает os.Args, переменные окружения и дефолты.
// Возвращает конфиг и error (например, при кривых флагах).
func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("family-tasks", flag.ContinueOnError)

	addr := fs.String("addr", envOr("ADDR", ":8787"), "адрес и порт, например :8787")
	webDir := fs.String("web-dir", envOr("WEB_DIR", ""), "путь к папке web/ (по умолчанию ищется рядом)")
	dbPath := fs.String("db", envOr("DB_PATH", filepath.Join("data", "tasks.db")), "путь к файлу SQLite")
	ntfyURL := fs.String("ntfy-url", envOr("NTFY_URL", ""), "базовый URL ntfy, например http://localhost:8080")
	ntfyTopic := fs.String("ntfy-topic", envOr("NTFY_TOPIC", ""), "topic ntfy, на который подписаны телефоны")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "уровень логов: debug|info|warn|error")
	logFormat := fs.String("log-format", envOr("LOG_FORMAT", "text"), "формат логов: text|json")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	c := &Config{
		Addr:      strings.TrimSpace(*addr),
		WebDir:    strings.TrimSpace(*webDir),
		DBPath:    strings.TrimSpace(*dbPath),
		NtfyURL:   strings.TrimRight(strings.TrimSpace(*ntfyURL), "/"),
		NtfyTopic: strings.TrimSpace(*ntfyTopic),
		LogLevel:  strings.ToLower(strings.TrimSpace(*logLevel)),
		LogFormat: strings.ToLower(strings.TrimSpace(*logFormat)),
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	if c.Addr == "" {
		return fmt.Errorf("addr must not be empty")
	}
	if c.DBPath == "" {
		return fmt.Errorf("db path must not be empty")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log level %q (want debug|info|warn|error)", c.LogLevel)
	}
	switch c.LogFormat {
	case "text", "json":
	default:
		return fmt.Errorf("invalid log format %q (want text|json)", c.LogFormat)
	}
	// ntfy: либо оба поля, либо ни одного.
	if (c.NtfyURL == "") != (c.NtfyTopic == "") {
		return fmt.Errorf("ntfy-url and ntfy-topic must be set together")
	}
	return nil
}

// NtfyEnabled — включены ли уведомления.
func (c *Config) NtfyEnabled() bool {
	return c.NtfyURL != "" && c.NtfyTopic != ""
}

// SlogLevel возвращает slog.Level для настройки логгера.
func (c *Config) SlogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Usage — текст справки. Используется для флага --help.
func Usage(fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(fs.Output(), "Flags (env var override in parentheses):\n")
		fs.PrintDefaults()
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Examples:")
		fmt.Fprintln(fs.Output(), "  family-tasks")
		fmt.Fprintln(fs.Output(), "  family-tasks -addr :9000 -db /var/lib/family-tasks/tasks.db")
		fmt.Fprintln(fs.Output(), "  family-tasks -ntfy-url http://localhost:8080 -ntfy-topic family-tasks-home")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
