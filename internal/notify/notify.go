package notify

import (
	"bytes"
	"fmt"
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
// Возвращает ошибку, если запрос не удался или ntfy ответил не 2xx.
// nil-клиент → сразу nil (уведомления отключены).
func (c *Client) Send(title, message, priority, tags, click string) error {
	if c == nil {
		return nil
	}
	if click == "" {
		click = c.click
	}

	url := c.baseURL + "/" + c.topic
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
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
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: status %d", resp.StatusCode)
	}
	slog.Debug("ntfy: sent", "title", title, "click", click)
	return nil
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
