package domain

import (
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
)

// StreamType names the user streams in the event store.
const StreamType = "user"

// Registered is the first event of every user: an account now exists for
// the address.
type Registered struct {
	UserID uuid.UUID `json:"user_id"`
	Email  Email     `json:"email"`
}

// EmailChanged means the user confirmed a new address.
type EmailChanged struct {
	Old Email `json:"old"`
	New Email `json:"new"`
}

// GoogleLinked means the user can sign in with the Google account whose
// subject (the ID token's sub) this is.
type GoogleLinked struct {
	Subject string `json:"subject"`
}

// RegisterEvents names the user events in r. The names are stored with
// every event, so they never change.
func RegisterEvents(r *es.Registry) {
	es.Register[Registered](r, "user.registered.v1")
	es.Register[EmailChanged](r, "user.email_changed.v1")
	es.Register[GoogleLinked](r, "user.google_linked.v1")
}
