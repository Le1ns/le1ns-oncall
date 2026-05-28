package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Lark struct {
	webhookURL string
	client     *http.Client
}

func NewLark(webhookURL string) *Lark {
	return &Lark{
		webhookURL: strings.TrimSpace(webhookURL),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (l *Lark) Enabled() bool {
	return l != nil && l.webhookURL != ""
}

func (l *Lark) Send(ctx context.Context, text string) error {
	if !l.Enabled() || strings.TrimSpace(text) == "" {
		return nil
	}

	body, err := json.Marshal(map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": stripHTML(text),
		},
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, l.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := l.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("lark webhook returned %s", response.Status)
	}
	return nil
}

func stripHTML(value string) string {
	replacer := strings.NewReplacer("<b>", "", "</b>", "")
	return replacer.Replace(value)
}
