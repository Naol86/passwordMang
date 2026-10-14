package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"passwordmanage/server/internal/app/notify"
	"passwordmanage/server/internal/platform/config"
)

const mailpitSMTPPort = 1025

// TestSendDeliversThroughSMTP sends a real message to Mailpit and finds it through Mailpit's API.
// It runs when PM_TEST_SMTP_HOST and PM_TEST_MAILPIT_URL are set (make test does this).
func TestSendDeliversThroughSMTP(t *testing.T) {
	host, apiURL := os.Getenv("PM_TEST_SMTP_HOST"), os.Getenv("PM_TEST_MAILPIT_URL")
	if host == "" || apiURL == "" {
		t.Skip("set PM_TEST_SMTP_HOST and PM_TEST_MAILPIT_URL to run the SMTP integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := "integration " + uuid.NewString()
	sender := NewSMTP(config.SMTP{Host: host, Port: mailpitSMTPPort, From: "Vault <vault@pm.localhost>", TLS: config.SMTPTLSNone})
	if err := sender.Send(ctx, notify.Email{To: "ada@example.com", Subject: subject, Text: "Hello Ada"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	query := url.Values{"query": {fmt.Sprintf("subject:%q", subject)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/api/v1/search?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("query mailpit: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		MessagesCount int `json:"messages_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.MessagesCount != 1 {
		t.Fatalf("found %d messages with subject %q, want 1", result.MessagesCount, subject)
	}
}
