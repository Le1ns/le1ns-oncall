package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

type Telegram struct {
	token     string
	defaultID string
	mapping   map[string]string
	client    *http.Client
}

type TelegramConfig struct {
	BotToken    string
	ChatID      string
	ChatMapping string
}

type Message struct {
	ChatID string
	Text   string
}

func NewTelegram(config TelegramConfig) *Telegram {
	return &Telegram{
		token:     strings.TrimSpace(config.BotToken),
		defaultID: strings.TrimSpace(config.ChatID),
		mapping:   parseMapping(config.ChatMapping),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *Telegram) Enabled() bool {
	return t != nil && t.token != ""
}

func (t *Telegram) ChatIDFor(login string) string {
	if t == nil {
		return ""
	}
	normalized := normalizeHandle(login)
	if normalized != "" {
		if chatID := t.mapping[normalized]; chatID != "" {
			return chatID
		}
	}
	return t.defaultID
}

func (t *Telegram) Send(ctx context.Context, message Message) error {
	if !t.Enabled() {
		return nil
	}
	chatID := strings.TrimSpace(message.ChatID)
	if chatID == "" {
		chatID = t.defaultID
	}
	if chatID == "" || strings.TrimSpace(message.Text) == "" {
		return nil
	}

	body, err := json.Marshal(map[string]any{
		"chat_id":                  chatID,
		"text":                     message.Text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage returned %s", response.Status)
	}
	return nil
}

func AlertMessage(status, severity, name, service, summary, assignee string) string {
	lines := []string{
		"<b>New critical alert</b>",
		"Status: " + escape(firstNonEmpty(status, "unknown")),
		"Severity: " + escape(firstNonEmpty(severity, "unknown")),
		"Alert: " + escape(firstNonEmpty(name, "unknown")),
	}
	if service != "" {
		lines = append(lines, "Service: "+escape(service))
	}
	if assignee != "" {
		lines = append(lines, "On-call: "+escape(assignee))
	}
	if summary != "" {
		lines = append(lines, "", escape(summary))
	}
	return strings.Join(lines, "\n")
}

func ShiftMessage(userName, userLogin, startsAt, endsAt string) string {
	person := firstNonEmpty(userName, userLogin, "unknown")
	return strings.Join([]string{
		"<b>On-call shift updated</b>",
		"Person: " + escape(person),
		"Starts: " + escape(startsAt),
		"Ends: " + escape(endsAt),
	}, "\n")
}

func parseMapping(value string) map[string]string {
	result := map[string]string{}
	for _, item := range strings.Split(value, ",") {
		key, chatID, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = normalizeHandle(key)
		chatID = strings.TrimSpace(chatID)
		if key != "" && chatID != "" {
			result[key] = chatID
		}
	}
	return result
}

func normalizeHandle(value string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(value), "@"))
}

func escape(value string) string {
	return html.EscapeString(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
