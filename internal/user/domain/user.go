package domain

import (
	"fmt"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
)

// maxGoogleSubjectBytes is the longest sub Google issues.
const maxGoogleSubjectBytes = 255

var (
	// ErrSameEmail rejects changing an address to itself.
	ErrSameEmail = apperr.Invalid("email", "is already your email address")
	// ErrGoogleAlreadyLinked rejects linking a second Google account: a
	// user has at most one.
	ErrGoogleAlreadyLinked = apperr.New(apperr.ErrConflict, "This account is already linked to another Google account.")

	errRegistered    = apperr.New(apperr.ErrConflict, "This user is already registered.")
	errNotRegistered = apperr.New(apperr.ErrNotFound, "This user doesn't exist.")
)

// User is someone who can sign in: an address, and at most one Google
// account.
type User struct {
	es.Changes
	id            uuid.UUID
	email         Email
	googleSubject string
}

// New returns an empty User, to register or to rebuild from its stream.
func New() *User { return &User{} }

// ID implements es.Aggregate.
func (u *User) ID() uuid.UUID { return u.id }

// Email is the user's current address.
func (u *User) Email() Email { return u.email }

// GoogleSubject is the linked Google account's subject, or "".
func (u *User) GoogleSubject() string { return u.googleSubject }

// Register makes u, a new User, the account with id for email.
func (u *User) Register(id uuid.UUID, email Email) error {
	switch {
	case u.registered():
		return errRegistered
	case id == uuid.Nil():
		return apperr.Invalid("user_id", "is required")
	case email.IsZero():
		return apperr.Invalid("email", "is required")
	}
	return es.Raise(u, Registered{UserID: id, Email: email})
}

// ChangeEmail moves the account to the address to.
func (u *User) ChangeEmail(to Email) error {
	switch {
	case !u.registered():
		return errNotRegistered
	case to.IsZero():
		return apperr.Invalid("email", "is required")
	case to == u.email:
		return ErrSameEmail
	}
	return es.Raise(u, EmailChanged{Old: u.email, New: to})
}

// LinkGoogle lets the Google account with subject sign in as u. Linking the
// account that is already linked changes nothing.
func (u *User) LinkGoogle(subject string) error {
	switch {
	case !u.registered():
		return errNotRegistered
	case subject == "":
		return apperr.Invalid("subject", "is required")
	case len(subject) > maxGoogleSubjectBytes:
		return apperr.Invalid("subject", "is longer than 255 bytes")
	case subject == u.googleSubject:
		return nil
	case u.googleSubject != "":
		return ErrGoogleAlreadyLinked
	}
	return es.Raise(u, GoogleLinked{Subject: subject})
}

func (u *User) registered() bool { return u.id != uuid.Nil() }

// Apply implements es.Aggregate.
func (u *User) Apply(event any) error {
	switch e := event.(type) {
	case Registered:
		u.id, u.email = e.UserID, e.Email
	case EmailChanged:
		u.email = e.New
	case GoogleLinked:
		u.googleSubject = e.Subject
	default:
		return fmt.Errorf("user: can't apply %T", event)
	}
	return nil
}
