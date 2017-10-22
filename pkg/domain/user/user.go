package user

import (
	"app/pkg/domain"
	"fmt"

	"github.com/google/uuid"
)

// User aggregate root
type User struct {
	id      uuid.UUID
	version int
	changes []*domain.Event

	email string
}

// FromHistory loads current aggregate root state by applying all events in order
func (self *User) FromHistory(events []*domain.Event) {
	for _, event := range events {
		self.transition(event.Payload)
		self.version++
	}
}

// ID returns aggregate root id
func (self *User) ID() uuid.UUID {
	return self.id
}

// Version returns current aggregate root version
func (self *User) Version() int {
	return self.version
}

// Changes returns all new applied events
func (self *User) Changes() []*domain.Event {
	return self.changes
}

func (self *User) transition(event interface{}) {
	switch e := event.(type) {
	case domainEvent:
		e.Apply(self)
	default:
		logger.Error(nil, "Unknown event %T", event)
	}
}

func (self *User) trackChange(event interface{}) {
	self.transition(event)
	eventEnvelop, err := domain.NewEvent(self.id, fmt.Sprintf("%T", self), self.version, event)

	if err != nil {
		logger.Error(nil, "Error %v parsing event %v", err, event)
		return
	}

	self.changes = append(self.changes, eventEnvelop)
}

func (self *User) registerWithEmail(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&WasRegisteredWithEmail{id, email, authToken})

	return nil
}

func (self *User) registerWithGoogle(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&WasRegisteredWithGoogle{id, email, authToken})

	return nil
}

func (self *User) registerWithFacebook(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&WasRegisteredWithFacebook{id, email, authToken})

	return nil
}

func (self *User) changeEmailAddress(email string) error {
	self.trackChange(&EmailAddressWasChanged{self.id, email})

	return nil
}

// New creates an User
func New() *User {
	return &User{}
}
