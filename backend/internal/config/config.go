package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr            string
	PublicBaseURL       string
	DatabaseURL         string
	GrafanaURL          string
	GrafanaServiceToken string
	GrafanaBasicAuth    string
	AlertmanagerToken   string
	AlertSeverities     []string
	AlertMaxPerGroup    int
	AlertRateWindow     time.Duration
	AlertMaxPerWebhook  int
	JiraBaseURL         string
	JiraEmail           string
	JiraAPIToken        string
	JiraProjectKey      string
	JiraIssueType       string
	TelegramBotToken    string
	TelegramChatID      string
	TelegramChatMapping string
	LarkWebhookURL      string
}

func Load() Config {
	return Config{
		HTTPAddr:            env("ONCALL_HTTP_ADDR", ":8080"),
		PublicBaseURL:       env("ONCALL_PUBLIC_BASE_URL", "http://localhost:8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		GrafanaURL:          env("ONCALL_GRAFANA_URL", "http://localhost:3000"),
		GrafanaServiceToken: os.Getenv("ONCALL_GRAFANA_SERVICE_TOKEN"),
		GrafanaBasicAuth:    os.Getenv("ONCALL_GRAFANA_BASIC_AUTH"),
		AlertmanagerToken:   env("ONCALL_ALERTMANAGER_TOKEN", "dev-alertmanager-token"),
		AlertSeverities:     listEnv("ONCALL_ALERTMANAGER_ALLOWED_SEVERITIES", []string{"*"}),
		AlertMaxPerGroup:    intEnv("ONCALL_ALERTMANAGER_MAX_ALERTS_PER_GROUP", 20),
		AlertRateWindow:     durationEnv("ONCALL_ALERTMANAGER_RATE_WINDOW", 10*time.Minute),
		AlertMaxPerWebhook:  intEnv("ONCALL_ALERTMANAGER_MAX_ALERTS_PER_WEBHOOK", 100),
		JiraBaseURL:         os.Getenv("ONCALL_JIRA_BASE_URL"),
		JiraEmail:           os.Getenv("ONCALL_JIRA_EMAIL"),
		JiraAPIToken:        os.Getenv("ONCALL_JIRA_API_TOKEN"),
		JiraProjectKey:      env("ONCALL_JIRA_PROJECT_KEY", "OPS"),
		JiraIssueType:       env("ONCALL_JIRA_ISSUE_TYPE", "Task"),
		TelegramBotToken:    os.Getenv("ONCALL_TELEGRAM_BOT_TOKEN"),
		TelegramChatID:      os.Getenv("ONCALL_TELEGRAM_CHAT_ID"),
		TelegramChatMapping: os.Getenv("ONCALL_TELEGRAM_CHAT_MAPPING"),
		LarkWebhookURL:      os.Getenv("ONCALL_LARK_WEBHOOK_URL"),
	}
}

func listEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func intEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
