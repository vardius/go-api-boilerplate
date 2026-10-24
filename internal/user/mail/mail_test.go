package mail_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/mail/mailtest"
)

func outboxMessage(t *testing.T, topic string, link mail.Link) outbox.Message {
	t.Helper()
	m, err := outbox.NewMessage(topic, link, t0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHandlersRenderEachTopic(t *testing.T) {
	const url = "https://app.example.com/sign-in/complete?code=abc&next=%2Fprofile"
	tests := []struct {
		topic, subject, text string
	}{
		{mail.TopicLoginLink, "Your sign-in link", "Open this link to sign in:\n\n" + url + "\n\nIt works once, for the next 15 minutes."},
		{mail.TopicEmailChange, "Confirm your new email address", "make this your account's email address:\n\n" + url + "\n\nIt works once, for the next 15 minutes,"},
	}
	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			capture := &mailtest.Capture{}
			handle := mail.Handlers(capture, func() time.Time { return t0 })[tt.topic]
			msg := outboxMessage(t, tt.topic, mail.Link{To: "ada@example.com", URL: url, ExpiresAt: t0.Add(14*time.Minute + time.Second)})

			if err := handle(t.Context(), msg); err != nil {
				t.Fatal(err)
			}
			sent := capture.Sent()
			if len(sent) != 1 {
				t.Fatalf("sent %d mails, want 1", len(sent))
			}
			m := sent[0]
			if m.ID != msg.ID || m.To != "ada@example.com" || m.Subject != tt.subject {
				t.Errorf("mail = %v to %s, %q", m.ID, m.To, m.Subject)
			}
			if !strings.Contains(m.Text, tt.text) {
				t.Errorf("text = %q, want it to contain %q", m.Text, tt.text)
			}
			// html/template escapes the & in the link.
			if !strings.Contains(m.HTML, `href="https://app.example.com/sign-in/complete?code=abc&amp;next=%2Fprofile"`) {
				t.Errorf("html = %s", m.HTML)
			}
		})
	}
}

func TestHandlersEscapeTheLinkInHTML(t *testing.T) {
	capture := &mailtest.Capture{}
	handle := mail.Handlers(capture, func() time.Time { return t0 })[mail.TopicLoginLink]
	msg := outboxMessage(t, mail.TopicLoginLink, mail.Link{To: "ada@example.com", URL: `javascript:alert("x")`, ExpiresAt: t0.Add(time.Minute)})
	if err := handle(t.Context(), msg); err != nil {
		t.Fatal(err)
	}
	if html := capture.Sent()[0].HTML; strings.Contains(html, `href="javascript:`) {
		t.Errorf("html carries a script link: %s", html)
	}
}

func TestHandlersDropAnExpiredLink(t *testing.T) {
	capture := &mailtest.Capture{}
	handle := mail.Handlers(capture, func() time.Time { return t0 })[mail.TopicLoginLink]
	msg := outboxMessage(t, mail.TopicLoginLink, mail.Link{To: "ada@example.com", URL: "https://app.example.com/x", ExpiresAt: t0})
	if err := handle(t.Context(), msg); err != nil {
		t.Fatal(err)
	}
	if n := len(capture.Sent()); n != 0 {
		t.Errorf("sent %d mails for an expired link", n)
	}
}

func TestHandlersReportFailures(t *testing.T) {
	capture := &mailtest.Capture{}
	handle := mail.Handlers(capture, func() time.Time { return t0 })[mail.TopicLoginLink]

	bad := outboxMessage(t, mail.TopicLoginLink, mail.Link{})
	bad.Payload = []byte(`{"to": 42}`)
	if err := handle(t.Context(), bad); err == nil {
		t.Error("a malformed payload was handled")
	}

	down := errors.New("connection refused")
	capture.FailNext(down)
	msg := outboxMessage(t, mail.TopicLoginLink, mail.Link{To: "ada@example.com", URL: "https://app.example.com/x", ExpiresAt: t0.Add(time.Minute)})
	if err := handle(t.Context(), msg); !errors.Is(err, down) {
		t.Errorf("handle = %v, want the mailer's error", err)
	}
	if err := handle(t.Context(), msg); err != nil {
		t.Errorf("retry: %v", err)
	}
	if n := len(capture.Sent()); n != 1 {
		t.Errorf("sent %d mails, want 1", n)
	}
}
