package store

import (
	"fmt"
	"testing"
	"time"
)

func TestSaveAlertmanagerPayloadFiltersSeverity(t *testing.T) {
	st := newMemoryStore()
	payload := alertmanagerPayload(
		alertPayload("critical", "critical-alert", "critical-fp"),
		alertPayload("warning", "warning-alert", "warning-fp"),
		alertPayload("fatal", "fatal-alert", "fatal-fp"),
	)

	result, err := st.SaveAlertmanagerPayload(payload, AlertIntakeConfig{
		AllowedSeverities: []string{"critical", "fatal"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Received != 3 || result.Saved != 2 || result.Filtered != 1 || result.RateLimited != 0 {
		t.Fatalf("unexpected intake result: %+v", result)
	}

	alerts := st.Alerts(10)
	if len(alerts) != 2 {
		t.Fatalf("expected 2 stored alerts, got %d", len(alerts))
	}
	for _, alert := range alerts {
		if alert.Severity != "critical" && alert.Severity != "fatal" {
			t.Fatalf("unexpected stored severity %q", alert.Severity)
		}
	}
}

func TestSaveAlertmanagerPayloadRateLimitsGroup(t *testing.T) {
	st := newMemoryStore()
	items := make([]map[string]any, 0, 3)
	for index := 0; index < 3; index++ {
		items = append(items, alertPayload("critical", fmt.Sprintf("DatabaseDown%d", index), fmt.Sprintf("fp-%d", index)))
	}

	result, err := st.SaveAlertmanagerPayload(alertmanagerPayload(items...), AlertIntakeConfig{
		AllowedSeverities: []string{"critical", "fatal"},
		MaxPerGroup:       2,
		RateWindow:        time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Received != 3 || result.Saved != 2 || result.Filtered != 0 || result.RateLimited != 1 {
		t.Fatalf("unexpected intake result: %+v", result)
	}
	if alerts := st.Alerts(10); len(alerts) != 2 {
		t.Fatalf("expected 2 stored alerts, got %d", len(alerts))
	}
}

func TestAlertNotificationChannelsMatchesPersonAndGroupSeverity(t *testing.T) {
	st := newMemoryStore()
	receiver, err := st.CreateNotificationReceiver(CreateNotificationReceiver{
		Kind:     "telegram",
		Name:     "Ops Telegram",
		BotToken: "token",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	person, err := st.CreateNotificationChannel(CreateNotificationChannel{
		ReceiverID:    receiver.ID,
		TargetType:    "person",
		Name:          "Primary",
		GrafanaUserID: 7,
		UserLogin:     "le1ns",
		UserName:      "Le1ns",
		ChatID:        "579908488",
		Severities:    []string{"critical"},
		Enabled:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	group, err := st.CreateNotificationChannel(CreateNotificationChannel{
		ReceiverID: receiver.ID,
		TargetType: "group",
		Name:       "SRE",
		ChatID:     "-100",
		Severities: []string{"warning", "critical"},
		Enabled:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	channels := st.AlertNotificationChannels(Alert{
		Severity:          "critical",
		AssignedUserID:    7,
		AssignedUserLogin: "le1ns",
	})

	if len(channels) != 2 {
		t.Fatalf("expected 2 matching channels, got %d", len(channels))
	}
	if channels[0].ID != group.ID || channels[1].ID != person.ID {
		t.Fatalf("unexpected channel order or ids: %+v", channels)
	}
}

func alertmanagerPayload(alerts ...map[string]any) map[string]any {
	items := make([]any, 0, len(alerts))
	for _, alert := range alerts {
		items = append(items, alert)
	}
	return map[string]any{
		"groupKey": "{}:{alertname=\"DatabaseDown\"}",
		"alerts":   items,
	}
}

func alertPayload(severity string, alertName string, fingerprint string) map[string]any {
	return map[string]any{
		"status":      "firing",
		"fingerprint": fingerprint,
		"labels": map[string]any{
			"alertname": alertName,
			"severity":  severity,
			"service":   "postgres",
		},
		"annotations": map[string]any{
			"summary": alertName,
		},
	}
}
