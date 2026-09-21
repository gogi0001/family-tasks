package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	WebDir           string
	DBPath           string
	UploadsDir       string
	MaxUploadMB      int
	NtfyURL          string
	NtfyTopic        string
	NtfyClick        string
	ReminderInterval string
	ReminderWindow   string
	LogLevel         string
	LogFormat        string
}

func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("family-tasks", flag.ContinueOnError)

	addr := fs.String("addr", envOr("ADDR", ":8787"), "адрес и порт, например :8787")
	webDir := fs.String("web-dir", envOr("WEB_DIR", ""), "путь к папке web/ (по умолчанию ищется рядом)")
	dbPath := fs.String("db", envOr("DB_PATH", filepath.Join("data", "tasks.db")), "путь к файлу SQLite")
	uploadsDir := fs.String("uploads-dir", envOr("UPLOADS_DIR", filepath.Join("data", "uploads")), "каталог для вложений")
	maxUploadMB := fs.Int("max-upload-mb", envOrInt("MAX_UPLOAD_MB", 20), "максимальный размер вложения, МБ")
	ntfyURL := fs.String("ntfy-url", envOr("NTFY_URL", ""), "базовый URL ntfy, например http://localhost:7070")
	ntfyTopic := fs.String("ntfy-topic", envOr("NTFY_TOPIC", ""), "topic ntfy, на который подписаны телефоны")
	ntfyClick := fs.String("ntfy-click", envOr("NTFY_CLICK", ""), "deep link для тапа по уведомлению, например familytasks://open")
	reminderInterval := fs.String("reminder-interval", envOr("REMINDER_INTERVAL", "5m"), "как часто проверять приближающиеся сроки (например, 5m)")
	reminderWindow := fs.String("reminder-window", envOr("REMINDER_WINDOW", "1h"), "за сколько до срока напоминать (например, 1h)")
	logLevel := fs.String("log-level", envOr("LOG_LEVEL", "info"), "уровень логов: debug|info|warn|error")
	logFormat := fs.String("log-format", envOr("LOG_FORMAT", "text"), "формат логов: text|json")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	c := &Config{
		Addr:             strings.TrimSpace(*addr),
		WebDir:           strings.TrimSpace(*webDir),
		DBPath:           strings.TrimSpace(*dbPath),
		UploadsDir:       strings.TrimSpace(*uploadsDir),
		MaxUploadMB:      *maxUploadMB,
		NtfyURL:          strings.TrimRight(strings.TrimSpace(*ntfyURL), "/"),
		NtfyTopic:        strings.TrimSpace(*ntfyTopic),
		NtfyClick:        strings.TrimSpace(*ntfyClick),
		ReminderInterval: strings.TrimSpace(*reminderInterval),
		ReminderWindow:   strings.TrimSpace(*reminderWindow),
		LogLevel:         strings.ToLower(strings.TrimSpace(*logLevel)),
		LogFormat:        strings.ToLower(strings.TrimSpace(*logFormat)),
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
	if c.UploadsDir == "" {
		return fmt.Errorf("uploads dir must not be empty")
	}
	if c.MaxUploadMB <= 0 {
		return fmt.Errorf("max-upload-mb must be positive")
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
	if (c.NtfyURL == "") != (c.NtfyTopic == "") {
		return fmt.Errorf("ntfy-url and ntfy-topic must be set together")
	}
	return nil
}

func (c *Config) NtfyEnabled() bool {
	return c.NtfyURL != "" && c.NtfyTopic != ""
}

// MaxUploadBytes возвращает лимит вложения в байтах.
func (c *Config) MaxUploadBytes() int64 {
	return int64(c.MaxUploadMB) << 20
}

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

// ReminderDurations возвращает распарсенные интервал и окно.
func (c *Config) ReminderDurations() (interval, window time.Duration, err error) {
	interval, err = time.ParseDuration(c.ReminderInterval)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid reminder-interval: %w", err)
	}
	window, err = time.ParseDuration(c.ReminderWindow)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid reminder-window: %w", err)
	}
	if interval <= 0 || window <= 0 {
		return 0, 0, fmt.Errorf("reminder durations must be positive")
	}
	return interval, window, nil
}

func Usage(fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(fs.Output(), "Flags (env var override in parentheses):\n")
		fs.PrintDefaults()
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Examples:")
		fmt.Fprintln(fs.Output(), "  family-tasks")
		fmt.Fprintln(fs.Output(), "  family-tasks -addr :9000 -db /var/lib/family-tasks/tasks.db")
		fmt.Fprintln(fs.Output(), "  family-tasks -max-upload-mb 50 -uploads-dir /var/lib/family-tasks/uploads")
		fmt.Fprintln(fs.Output(), "  family-tasks -ntfy-url http://localhost:7070 -ntfy-topic family-tasks-home -ntfy-click familytasks://open")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
