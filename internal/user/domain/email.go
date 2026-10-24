// Package domain holds the user service's rules: the User aggregate, the
// events it raises and the Email value object. It knows nothing of
// transport or storage.
package domain

import (
	"net/mail"
	"strings"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// maxEmailBytes is the longest address SMTP can carry (RFC 5321's path
// limit, less the angle brackets).
const maxEmailBytes = 254

// Email is an email address: a bare addr-spec, lower-cased. The zero Email
// is no address.
type Email struct {
	address string
}

// ParseEmail returns s as an Email. It accepts one bare address, such as
// "Ada@Example.com", which it lower-cases, and rejects display names,
// groups, comments and anything over 254 bytes, with an
// *apperr.ValidationError on the field "email".
func ParseEmail(s string) (Email, error) {
	// Lower-casing can lengthen a string, so it comes before the length
	// check.
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "":
		return Email{}, apperr.Invalid("email", "is required")
	case len(s) > maxEmailBytes:
		return Email{}, apperr.Invalid("email", "is longer than 254 bytes")
	}
	addr, err := mail.ParseAddress(s)
	// ParseAddress also accepts "Name <addr>", "<addr>" and quoted local
	// parts, which it unquotes; only a bare address that comes back as it
	// was typed is one.
	if err != nil || addr.Name != "" || addr.Address != s {
		return Email{}, apperr.Invalid("email", "is not an email address")
	}
	return Email{address: s}, nil
}

// String returns the address.
func (e Email) String() string { return e.address }

// IsZero reports whether e is no address.
func (e Email) IsZero() bool { return e.address == "" }

// MarshalText returns the address.
func (e Email) MarshalText() ([]byte, error) { return []byte(e.address), nil }

// UnmarshalText reads an address as an event stored it. It doesn't check
// it again: stored events are facts, and the rules may have changed since
// they were written.
func (e *Email) UnmarshalText(text []byte) error {
	e.address = string(text)
	return nil
}
