package mail_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	netmail "net/mail"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
)

var t0 = time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC)

func newSMTP(t *testing.T, cfg mail.SMTPConfig, roots *x509.CertPool) *mail.SMTP {
	t.Helper()
	if cfg.From == "" {
		cfg.From = "go-api-boilerplate <no-reply@go-api-boilerplate.localhost>"
	}
	s, err := mail.NewSMTP(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.TLS = &tls.Config{RootCAs: roots}
	s.Now = func() time.Time { return t0 }
	return s
}

var message = mail.Message{
	ID:      uuid.MustParse("0192d5a8-7c4e-7000-8000-00000000abcd"),
	To:      "ada@example.com",
	Subject: "Zaloguj się",
	Text:    "Open https://app.example.com/sign-in/complete?code=abc_DEF-123&next=%2Fprofile\n",
	HTML:    `<p><a href="https://app.example.com/sign-in/complete?code=abc_DEF-123">Sign in</a></p>`,
}

// parsed is a received message, decoded.
type parsed struct {
	header     netmail.Header
	subject    string
	text, html string
}

func parse(t *testing.T, data string) parsed {
	t.Helper()
	msg, err := netmail.ReadMessage(strings.NewReader(data + "\r\n"))
	if err != nil {
		t.Fatalf("read message: %v\n%s", err, data)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatal(err)
	}
	mt, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/alternative" {
		t.Fatalf("Content-Type = %q, %v", msg.Header.Get("Content-Type"), err)
	}
	p := parsed{header: msg.Header, subject: subject}
	parts := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(part) // NextPart undoes quoted-printable
		if err != nil {
			t.Fatal(err)
		}
		switch ct := part.Header.Get("Content-Type"); ct {
		case "text/plain; charset=utf-8":
			p.text = string(body)
		case "text/html; charset=utf-8":
			p.html = string(body)
		default:
			t.Errorf("unexpected part %q", ct)
		}
	}
	return p
}

func TestSMTPSendsOverSTARTTLS(t *testing.T) {
	srv, roots := newFakeSMTP(t, true)
	s := newSMTP(t, mail.SMTPConfig{Addr: srv.addr()}, roots)

	if err := s.Send(t.Context(), message); err != nil {
		t.Fatal(err)
	}
	got := srv.mails()
	if len(got) != 1 {
		t.Fatalf("server received %d mails, want 1", len(got))
	}
	m := got[0]
	if !m.tls {
		t.Error("the mail went in plain text")
	}
	if m.from != "FROM:<no-reply@go-api-boilerplate.localhost>" || m.to != "TO:<ada@example.com>" {
		t.Errorf("envelope = %s %s", m.from, m.to)
	}

	p := parse(t, m.data)
	wantHeaders := map[string]string{
		"From":         `"go-api-boilerplate" <no-reply@go-api-boilerplate.localhost>`,
		"To":           "ada@example.com",
		"Message-Id":   "<0192d5a8-7c4e-7000-8000-00000000abcd@go-api-boilerplate.localhost>",
		"Date":         "Sat, 24 Oct 2026 09:30:00 +0000",
		"Mime-Version": "1.0",
	}
	for k, want := range wantHeaders {
		if got := p.header.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if p.subject != "Zaloguj się" {
		t.Errorf("Subject = %q", p.subject)
	}
	// Mail lines end in CRLF.
	if p.text != strings.ReplaceAll(message.Text, "\n", "\r\n") || p.html != message.HTML {
		t.Errorf("parts = %q / %q", p.text, p.html)
	}
}

func TestSMTPRefusesPlainTextUnlessAllowed(t *testing.T) {
	srv, _ := newFakeSMTP(t, false)

	err := newSMTP(t, mail.SMTPConfig{Addr: srv.addr()}, nil).Send(t.Context(), message)
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("Send to a server without STARTTLS: err = %v, want a refusal", err)
	}
	if n := len(srv.mails()); n != 0 {
		t.Fatalf("server received %d mails", n)
	}

	if err := newSMTP(t, mail.SMTPConfig{Addr: srv.addr(), AllowInsecure: true}, nil).Send(t.Context(), message); err != nil {
		t.Fatalf("Send with SMTP_ALLOW_INSECURE: %v", err)
	}
	if got := srv.mails(); len(got) != 1 || got[0].tls {
		t.Fatalf("received %+v, want one plain-text mail", got)
	}
}

func TestSMTPLogsIn(t *testing.T) {
	srv, roots := newFakeSMTP(t, true)
	s := newSMTP(t, mail.SMTPConfig{Addr: srv.addr(), Username: "relay", Password: config.Secret("s3cret")}, roots)
	if err := s.Send(t.Context(), message); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.logins) != 1 || srv.logins[0] != "\x00relay\x00s3cret" {
		t.Errorf("logins = %q", srv.logins)
	}
}

func TestSMTPGivesUpWhenTheContextEnds(t *testing.T) {
	srv, _ := newFakeSMTP(t, false)
	srv.mu.Lock()
	srv.silent = true
	srv.mu.Unlock()
	s := newSMTP(t, mail.SMTPConfig{Addr: srv.addr(), AllowInsecure: true}, nil)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-srv.accepted
		cancel()
	}()
	err := s.Send(ctx, message)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Send = %v, want context.Canceled", err)
	}
}

func TestSMTPRejectsBadInput(t *testing.T) {
	if _, err := mail.NewSMTP(mail.SMTPConfig{Addr: "no-port", From: "a@example.com"}); err == nil {
		t.Error("NewSMTP accepted an address without a port")
	}
	if _, err := mail.NewSMTP(mail.SMTPConfig{Addr: "localhost:25", From: "not an address"}); err == nil {
		t.Error("NewSMTP accepted a bad sender")
	}

	srv, roots := newFakeSMTP(t, true)
	s := newSMTP(t, mail.SMTPConfig{Addr: srv.addr()}, roots)
	for _, m := range []mail.Message{
		{ID: message.ID, To: "ada@example.com\r\nBcc: eve@example.com", Subject: "x"},
		{ID: message.ID, To: "ada@example.com", Subject: "x\r\nBcc: eve@example.com"},
		{ID: message.ID, To: "not an address", Subject: "x"},
	} {
		if err := s.Send(t.Context(), m); err == nil {
			t.Errorf("sent %q / %q", m.To, m.Subject)
		}
	}
	if n := len(srv.mails()); n != 0 {
		t.Errorf("server received %d mails", n)
	}
}

// The relay retries a delivery the server refused, and the server ends up
// with one mail: the retry, carrying the same Message-ID.
func TestRelaySendsOnceWhenTheServerFailsOnce(t *testing.T) {
	srv, roots := newFakeSMTP(t, true)
	srv.mu.Lock()
	srv.failData = 1
	srv.mu.Unlock()
	s := newSMTP(t, mail.SMTPConfig{Addr: srv.addr()}, roots)

	db := memdb.New()
	box := memdb.NewOutbox(db)
	msg, err := outbox.NewMessage(mail.TopicLoginLink, mail.Link{
		To:        "ada@example.com",
		URL:       "https://app.example.com/sign-in/complete?code=abc",
		ExpiresAt: t0.Add(15 * time.Minute),
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Do(t.Context(), func(tx uow.Tx) error { return box.Add(t.Context(), tx, msg) }); err != nil {
		t.Fatal(err)
	}

	now := t0
	clock := func() time.Time { return now }
	relay := &outbox.Relay{Store: box, Handlers: mail.Handlers(s, clock), Now: clock, Logger: slog.New(slog.DiscardHandler)}
	for _, at := range []time.Time{t0, t0.Add(time.Second), t0.Add(time.Minute)} {
		now = at
		if _, err := relay.Drain(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	got := srv.mails()
	if len(got) != 1 {
		t.Fatalf("server received %d mails, want exactly 1", len(got))
	}
	if id := parse(t, got[0].data).header.Get("Message-Id"); id != "<"+msg.ID.String()+"@go-api-boilerplate.localhost>" {
		t.Errorf("Message-ID = %q, want the outbox message's ID", id)
	}
	if left, err := box.Claim(t.Context(), t0.Add(time.Hour), time.Minute, 10); err != nil || len(left) != 0 {
		t.Errorf("outbox still holds %d messages, %v", len(left), err)
	}
}
