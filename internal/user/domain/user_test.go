package domain_test

import (
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/es/estest"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
)

var (
	id    = uuid.MustParse("0192d5a8-7c4e-7000-8000-000000000001")
	ada   = mustEmail("ada@example.com")
	grace = mustEmail("grace@example.com")
)

func mustEmail(s string) domain.Email {
	e, err := domain.ParseEmail(s)
	if err != nil {
		panic(err)
	}
	return e
}

func registered() []any { return []any{domain.Registered{UserID: id, Email: ada}} }

func TestRegister(t *testing.T) {
	u := estest.Given(t, domain.New()).
		When(func(u *domain.User) error { return u.Register(id, ada) }).
		Then(domain.Registered{UserID: id, Email: ada})
	if u.ID() != id || u.Email() != ada || u.GoogleSubject() != "" {
		t.Errorf("registered user = %v %v %q", u.ID(), u.Email(), u.GoogleSubject())
	}
}

func TestRegisterRules(t *testing.T) {
	tests := []struct {
		name    string
		history []any
		id      uuid.UUID
		email   domain.Email
		want    error
	}{
		{"twice", registered(), uuid.NewV7(), grace, apperr.ErrConflict},
		{"without an ID", nil, uuid.Nil(), ada, apperr.ErrInvalid},
		{"without an email", nil, id, domain.Email{}, apperr.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estest.Given(t, domain.New(), tt.history...).
				When(func(u *domain.User) error { return u.Register(tt.id, tt.email) }).
				ThenError(tt.want)
		})
	}
}

func TestChangeEmail(t *testing.T) {
	u := estest.Given(t, domain.New(), registered()...).
		When(func(u *domain.User) error { return u.ChangeEmail(grace) }).
		Then(domain.EmailChanged{Old: ada, New: grace})
	if u.Email() != grace {
		t.Errorf("email = %v, want %v", u.Email(), grace)
	}
}

func TestChangeEmailRules(t *testing.T) {
	tests := []struct {
		name    string
		history []any
		to      domain.Email
		want    error
	}{
		{"to the same address", registered(), ada, domain.ErrSameEmail},
		{"to the same address in another case", registered(), mustEmail("ADA@example.COM"), domain.ErrSameEmail},
		{"to no address", registered(), domain.Email{}, apperr.ErrInvalid},
		{"before registering", nil, grace, apperr.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estest.Given(t, domain.New(), tt.history...).
				When(func(u *domain.User) error { return u.ChangeEmail(tt.to) }).
				ThenError(tt.want)
		})
	}
}

func TestLinkGoogle(t *testing.T) {
	u := estest.Given(t, domain.New(), registered()...).
		When(func(u *domain.User) error { return u.LinkGoogle("1084") }).
		Then(domain.GoogleLinked{Subject: "1084"})
	if u.GoogleSubject() != "1084" {
		t.Errorf("subject = %q", u.GoogleSubject())
	}
}

func TestLinkGoogleAgainChangesNothing(t *testing.T) {
	estest.Given(t, domain.New(), append(registered(), domain.GoogleLinked{Subject: "1084"})...).
		When(func(u *domain.User) error { return u.LinkGoogle("1084") }).
		Then()
}

func TestLinkGoogleRules(t *testing.T) {
	linked := append(registered(), domain.GoogleLinked{Subject: "1084"})
	tests := []struct {
		name    string
		history []any
		subject string
		want    error
	}{
		{"a second account", linked, "2048", domain.ErrGoogleAlreadyLinked},
		{"no subject", registered(), "", apperr.ErrInvalid},
		{"an oversize subject", registered(), strings.Repeat("9", 256), apperr.ErrInvalid},
		{"before registering", nil, "1084", apperr.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estest.Given(t, domain.New(), tt.history...).
				When(func(u *domain.User) error { return u.LinkGoogle(tt.subject) }).
				ThenError(tt.want)
		})
	}
	if !errors.Is(domain.ErrGoogleAlreadyLinked, apperr.ErrConflict) {
		t.Error("a second Google account isn't a conflict")
	}
}

func TestReplayRebuildsTheUser(t *testing.T) {
	u := estest.Given(t, domain.New(),
		domain.Registered{UserID: id, Email: ada},
		domain.GoogleLinked{Subject: "1084"},
		domain.EmailChanged{Old: ada, New: grace},
	).When(func(*domain.User) error { return nil }).Then()
	if u.ID() != id || u.Email() != grace || u.GoogleSubject() != "1084" || u.Version() != 3 {
		t.Errorf("rebuilt user = %v %v %q version %d", u.ID(), u.Email(), u.GoogleSubject(), u.Version())
	}
}

func TestApplyRejectsAnUnknownEvent(t *testing.T) {
	if err := domain.New().Apply(struct{}{}); err == nil {
		t.Error("Apply accepted an event it doesn't know")
	}
}

// The stored shape of each event is a contract with every event already
// written, so these literals change only with a new .vN type.
func TestEventsKeepTheirStoredShape(t *testing.T) {
	r := es.NewRegistry()
	domain.RegisterEvents(r)
	tests := []struct {
		event any
		name  string
		json  string
	}{
		{domain.Registered{UserID: id, Email: ada}, "user.registered.v1", `{"user_id":"0192d5a8-7c4e-7000-8000-000000000001","email":"ada@example.com"}`},
		{domain.EmailChanged{Old: ada, New: grace}, "user.email_changed.v1", `{"old":"ada@example.com","new":"grace@example.com"}`},
		{domain.GoogleLinked{Subject: "1084"}, "user.google_linked.v1", `{"subject":"1084"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, data, err := r.Encode(tt.event)
			if err != nil {
				t.Fatal(err)
			}
			if name != tt.name || string(data) != tt.json {
				t.Errorf("encoded %s %s, want %s %s", name, data, tt.name, tt.json)
			}
			back, err := r.Decode(tt.name, []byte(tt.json))
			if err != nil {
				t.Fatal(err)
			}
			if back != tt.event {
				t.Errorf("decoded %#v, want %#v", back, tt.event)
			}
		})
	}
}
