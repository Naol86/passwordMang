package mail

import (
	"errors"
	"net/mail"
	"strings"
	"testing"
	"time"

	"passwordmanage/server/internal/app/notify"
)

var (
	testFrom = &mail.Address{Name: "Vault", Address: "vault@example.com"}
	testNow  = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
)

func TestBuildMessage(t *testing.T) {
	msg, err := buildMessage(testFrom, notify.Email{
		To:      "ada@example.com",
		Subject: "Willkommen bei Acme",
		Text:    "Hello Ada",
	}, testNow)
	if err != nil {
		t.Fatal(err)
	}
	got := string(msg)
	for _, want := range []string{
		"From: \"Vault\" <vault@example.com>\r\n",
		"To: <ada@example.com>\r\n",
		"Subject: Willkommen bei Acme\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nHello Ada",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q:\n%s", want, got)
		}
	}
}

func TestBuildMessageRejectsHeaderInjection(t *testing.T) {
	tests := []notify.Email{
		{To: "ada@example.com\r\nBcc: eve@example.com", Subject: "Hi"},
		{To: "ada@example.com", Subject: "Hi\r\nBcc: eve@example.com"},
	}
	for _, email := range tests {
		if _, err := buildMessage(testFrom, email, testNow); !errors.Is(err, errHeaderInjection) {
			t.Errorf("expected header injection error for %+v, got %v", email, err)
		}
	}
}

func TestBuildMessageRejectsInvalidRecipient(t *testing.T) {
	if _, err := buildMessage(testFrom, notify.Email{To: "not an address", Subject: "Hi"}, testNow); err == nil {
		t.Fatal("expected invalid recipient error")
	}
}
