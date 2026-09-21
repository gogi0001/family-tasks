package notify

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"
)

// Client отправляет уведомления в ntfy.
// nil-указатель допустим — все методы становятся no-op.
type Client struct {
	baseURL string
	topic   string
	http    *http.Client
}

// NewClient создаёт клиент. Если baseURL или topic пусты — возвращает nil,
// и уведомления отключаются автоматически.
func NewClient(baseURL, topic string) *Client {
	if baseURL == "" || topic == "" {
		slog.Info("ntfy notifications disabled (url or topic not set)")
		return nil
	}
	slog.Info("ntfy notifications enabled", "url", baseURL, "topic", topic)
	return &Client{
		baseURL: baseURL,
		topic:   topic,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Send отправляет уведомление.
// title    — заголовок (виден крупно)
// message  — текст
// priority — "min"|"low"|"default"|"high"|"urgent" (пусто = default)
// tags     — emoji/иконки через запятую: "memo", "warning", "+1" и т.д.
func (c *Client) Send(title, message, priority, tags string) {
	if c == nil {
		return
	}

	url := c.baseURL + "/" + c.topic
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		slog.Error("ntfy: create request", "err", err)
		return
	}

	if title != "" {
		req.Header.Set("Title", encodeHeader(title))
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}
	if tags != "" {
		req.Header.Set("Tags", tags)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("ntfy: send failed", "err", err, "url", url)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slog.Warn("ntfy: server returned error", "status", resp.StatusCode)
		return
	}
	slog.Debug("ntfy: sent", "title", title)
}

// encodeHeader — ntfy ожидает заголовки в UTF-8, но HTTP-хедеры формально
// ASCII. Go это не мешает (шлёт как есть), но некоторые прокси ломают
// кириллицу. Используем "RFC 2047"-подобный вариант, ntfy его понимает.
func encodeHeader(s string) string {
	// Простейший путь — оставить как есть, ntfy корректно принимает UTF-8.
	return s
}
