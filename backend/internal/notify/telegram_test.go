package notify

import "testing"

func TestTelegramChatIDForUsesMappingBeforeDefault(t *testing.T) {
	tg := NewTelegram(TelegramConfig{
		BotToken:    "token",
		ChatID:      "default-chat",
		ChatMapping: "Le1ns=12345, admin=67890",
	})

	if got := tg.ChatIDFor("@Le1ns"); got != "12345" {
		t.Fatalf("expected mapped chat id, got %q", got)
	}
	if got := tg.ChatIDFor("unknown"); got != "default-chat" {
		t.Fatalf("expected default chat id, got %q", got)
	}
}

func TestTelegramDisabledWithoutChatTarget(t *testing.T) {
	tg := NewTelegram(TelegramConfig{})

	if tg.Enabled() {
		t.Fatal("telegram should be disabled without a bot token")
	}
}
