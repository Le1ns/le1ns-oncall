package config

import "os"

type Config struct {
	HTTPAddr            string
	PublicBaseURL       string
	GrafanaURL          string
	GrafanaServiceToken string
	GrafanaBasicAuth    string
	AlertmanagerToken   string
	JiraBaseURL         string
	JiraEmail           string
	JiraAPIToken        string
	JiraProjectKey      string
	JiraIssueType       string
	TelegramBotToken    string
	LarkWebhookURL      string
}

func Load() Config {
	return Config{
		HTTPAddr:            env("ONCALL_HTTP_ADDR", ":8080"),
		PublicBaseURL:       env("ONCALL_PUBLIC_BASE_URL", "http://localhost:8080"),
		GrafanaURL:          env("ONCALL_GRAFANA_URL", "http://localhost:3000"),
		GrafanaServiceToken: os.Getenv("ONCALL_GRAFANA_SERVICE_TOKEN"),
		GrafanaBasicAuth:    os.Getenv("ONCALL_GRAFANA_BASIC_AUTH"),
		AlertmanagerToken:   env("ONCALL_ALERTMANAGER_TOKEN", "dev-alertmanager-token"),
		JiraBaseURL:         os.Getenv("ONCALL_JIRA_BASE_URL"),
		JiraEmail:           os.Getenv("ONCALL_JIRA_EMAIL"),
		JiraAPIToken:        os.Getenv("ONCALL_JIRA_API_TOKEN"),
		JiraProjectKey:      env("ONCALL_JIRA_PROJECT_KEY", "OPS"),
		JiraIssueType:       env("ONCALL_JIRA_ISSUE_TYPE", "Task"),
		TelegramBotToken:    os.Getenv("ONCALL_TELEGRAM_BOT_TOKEN"),
		LarkWebhookURL:      os.Getenv("ONCALL_LARK_WEBHOOK_URL"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
