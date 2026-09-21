package notify

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client отправляет уведомления в ntfy.
// nil-указатель допустим — все методы становятся no-op.
type Client struct {
	baseURL string
	topic   string
	click   string // базовый deep link, например "familytasks://open"
	http    *http.Client
}

// NewClient создаёт клиент. Если baseURL или topic пусты — возвращает nil,
// и уведомления отключаются автоматически.
func NewClient(baseURL, topic, click string) *Client {
	if baseURL == "" || topic == "" {
		slog.Info("ntfy notifications disabled (url or topic not set)")
		return nil
	}
	slog.Info("ntfy notifications enabled",
		"url", baseURL,
		"topic", topic,
		"click", click,
	)
	return &Client{
		baseURL: baseURL,
		topic:   topic,
		click:   click,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Send отправляет уведомление.
// title    — заголовок (виден крупно)
// message  — текст
// priority — "min"|"low"|"default"|"high"|"urgent" (пусто = default)
// tags     — emoji/иконки через запятую: "memo", "warning", "+1"
// click    — полный URL для тапа; если пусто, используется базовый из конфига
func (c *Client) Send(title, message, priority, tags, click string) {
	if c == nil {
		return
	}
	if click == "" {
		click = c.click
	}

	url := c.baseURL + "/" + c.topic
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		slog.Error("ntfy: create request", "err", err)
		return
	}

	if title != "" {
		req.Header.Set("Title", title)
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}
	if tags != "" {
		req.Header.Set("Tags", tags)
	}
	if click != "" {
		req.Header.Set("Click", click)
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
	slog.Debug("ntfy: sent", "title", title, "click", click)
}

// TaskClick собирает URL на конкретную задачу из базового click.
// Если базовый click пуст или taskID пуст — вернёт пустую строку,
// и Send подставит базовый click как есть.
func (c *Client) TaskClick(taskID string) string {
	if c == nil || c.click == "" || taskID == "" {
		return ""
	}
	sep := "?"
	if strings.Contains(c.click, "?") {
		sep = "&"
	}
	return c.click + sep + "task=" + url.QueryEscape(taskID)
}
