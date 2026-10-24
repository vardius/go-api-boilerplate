// Package mail sends the user service's emails: sign-in links and
// email-change confirmations.
//
// The use cases don't send mail themselves. They add an outbox message on
// topic TopicLoginLink or TopicEmailChange, with a Link as its payload, in
// the transaction that creates the code, and the relay hands it to the
// Handlers here, which render the templates and send it with a Mailer. A
// message can be delivered more than once, so every send of it carries the
// same Message-ID, made from the outbox message ID: mail systems that see
// the same Message-ID twice can tell it is one mail.
package mail

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"math"
	"text/template"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
)

// Message is one email.
type Message struct {
	// ID is the same for every send of one outbox message; it becomes the
	// Message-ID header.
	ID      uuid.UUID
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer sends email.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// The outbox topics this package handles.
const (
	TopicLoginLink   = "mail.login_link"
	TopicEmailChange = "mail.email_change"
)

// Link is the payload of both topics: where to send a one-time link, and
// when it stops working. It is stored in the outbox until delivered, so its
// JSON shape changes only in ways old messages still decode.
type Link struct {
	To        string    `json:"to"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

//go:embed templates
var templates embed.FS

var (
	textTemplates = template.Must(template.ParseFS(templates, "templates/*.txt.tmpl"))
	htmlTemplates = htmltemplate.Must(htmltemplate.ParseFS(templates, "templates/*.html.tmpl"))
)

type kind struct {
	subject  string
	template string
}

var kinds = map[string]kind{
	TopicLoginLink:   {subject: "Your sign-in link", template: "login_link"},
	TopicEmailChange: {subject: "Confirm your new email address", template: "email_change"},
}

// Handlers returns the outbox handlers for both topics, sending with m.
// now is the clock; a link that has expired by the time its message is
// delivered is dropped rather than sent, since it would only fail.
func Handlers(m Mailer, now func() time.Time) map[string]outbox.Handler {
	handlers := make(map[string]outbox.Handler, len(kinds))
	for topic, k := range kinds {
		handlers[topic] = func(ctx context.Context, msg outbox.Message) error {
			var link Link
			if err := json.Unmarshal(msg.Payload, &link); err != nil {
				return fmt.Errorf("mail: decode %s %s: %w", topic, msg.ID, err)
			}
			left := link.ExpiresAt.Sub(now())
			if left <= 0 {
				return nil
			}
			email, err := render(k, msg.ID, link, left)
			if err != nil {
				return err
			}
			if err := m.Send(ctx, email); err != nil {
				return fmt.Errorf("mail: send %s %s: %w", topic, msg.ID, err)
			}
			return nil
		}
	}
	return handlers
}

type view struct {
	URL string
	// Minutes is how long the link has left, rounded up.
	Minutes int
}

func render(k kind, id uuid.UUID, link Link, left time.Duration) (Message, error) {
	v := view{URL: link.URL, Minutes: int(math.Ceil(left.Minutes()))}
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, k.template+".txt.tmpl", v); err != nil {
		return Message{}, fmt.Errorf("mail: render %s text: %w", k.template, err)
	}
	if err := htmlTemplates.ExecuteTemplate(&html, k.template+".html.tmpl", v); err != nil {
		return Message{}, fmt.Errorf("mail: render %s html: %w", k.template, err)
	}
	return Message{ID: id, To: link.To, Subject: k.subject, Text: text.String(), HTML: html.String()}, nil
}
