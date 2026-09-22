package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gogi0001/family-tasks/internal/email"
)

func main() {
	host := flag.String("host", envOr("SMTP_HOST", ""), "SMTP-хост (например, smtp.gmail.com)")
	port := flag.Int("port", envOrInt("SMTP_PORT", 587), "SMTP-порт (587 STARTTLS, 465 SMTPS, 1025 для Mailhog)")
	user := flag.String("user", envOr("SMTP_USER", ""), "SMTP-логин")
	pass := flag.String("pass", envOr("SMTP_PASS", ""), "SMTP-пароль")
	from := flag.String("from", envOr("SMTP_FROM", ""), "отправитель: «Имя <addr@example.com>» или просто addr")
	to := flag.String("to", envOr("MAIL_TO", ""), "адрес получателя (обязательно)")
	subj := flag.String("subject", "Family Tasks: проверка почты", "тема письма")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"mailtest — отправка тестового письма для проверки SMTP.\n\n"+
				"Usage: %s -to <addr> [-host ...] [-port ...] [-user ...] [-pass ...] [-from ...]\n\n"+
				"Flags (env var override in parentheses):\n",
			os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Examples:")
		fmt.Fprintln(flag.CommandLine.Output(), "  mailtest -to test@example.com \\")
		fmt.Fprintln(flag.CommandLine.Output(), "    -host smtp.gmail.com -port 587 \\")
		fmt.Fprintln(flag.CommandLine.Output(), "    -user you@gmail.com -pass <app-password> \\")
		fmt.Fprintln(flag.CommandLine.Output(), "    -from '\"Family Tasks\" <you@gmail.com>'")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "  # Mailhog для локальной проверки:")
		fmt.Fprintln(flag.CommandLine.Output(), "  mailtest -to test@local -host 127.0.0.1 -port 1025 -from test@local")
	}
	flag.Parse()

	if *to == "" {
		fmt.Fprintln(os.Stderr, "error: -to is required")
		flag.Usage()
		os.Exit(2)
	}
	if *host == "" {
		fmt.Fprintln(os.Stderr, "error: -host is required")
		os.Exit(2)
	}
	if *from == "" {
		fmt.Fprintln(os.Stderr, "error: -from is required")
		os.Exit(2)
	}

	sender := email.New(email.Config{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *pass,
		From:     *from,
	})
	if sender == nil {
		fmt.Fprintln(os.Stderr, "error: sender not initialized (empty host?)")
		os.Exit(2)
	}

	body := fmt.Sprintf(
		"Привет!\n\n"+
			"Это тестовое письмо от Family Tasks.\n"+
			"Время отправки: %s\n"+
			"SMTP-хост:     %s:%d\n"+
			"Отправитель:   %s\n"+
			"Получатель:    %s\n\n"+
			"Если вы получили это письмо — SMTP настроен корректно.\n",
		time.Now().Format("2006-01-02 15:04:05"),
		*host, *port, *from, *to,
	)

	fmt.Printf("→ %s:%d, from=%q, to=%q\n", *host, *port, *from, *to)
	if err := sender.Send(*to, *subj, body); err != nil {
		fmt.Fprintln(os.Stderr, "✗ Ошибка:", err)
		os.Exit(1)
	}
	fmt.Println("✓ OK: письмо отправлено.")
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
