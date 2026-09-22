package email

import (
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string // "Name <addr@example.com>" или просто "addr@example.com"
}

type Sender struct {
	cfg Config
}

func New(cfg Config) *Sender {
	if cfg.Host == "" {
		slog.Info("email: disabled (SMTP host not configured)")
		return nil
	}
	slog.Info("email: enabled", "host", cfg.Host, "port", cfg.Port, "from", cfg.From)
	return &Sender{cfg: cfg}
}

// Send отправляет простое текстовое письмо.
func (s *Sender) Send(to, subject, body string) error {
	if s == nil {
		return fmt.Errorf("email not configured")
	}

	fromAddr, fromName := splitFrom(s.cfg.From)
	if fromAddr == "" {
		return fmt.Errorf("invalid 'from' address")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(fromAddr, fromName))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetHeader("Date", time.Now().Format(time.RFC1123Z))
	m.SetBody("text/plain; charset=UTF-8", body)

	d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password)
	// d.Timeout = 10 * time.Second

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func splitFrom(s string) (addr, name string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address, a.Name
	}
	// Просто email без имени
	return s, ""
}
