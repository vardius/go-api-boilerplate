package user

import (
	"app/pkg/domain"
	"fmt"

	"github.com/google/uuid"
)

type User struct {
	id      uuid.UUID
	version int
	changes []*domain.Event

	email string
}

func (self *User) FromHistory(events []*domain.Event) {
	for _, event := range events {
		self.transition(event.Payload)
		self.version++
	}
}

func (self *User) ID() uuid.UUID {
	return self.id
}

func (self *User) Version() int {
	return self.version
}

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
	eventEnvelop := domain.NewEvent(self.id, fmt.Sprintf("%T", self), self.version, event)
	self.changes = append(self.changes, eventEnvelop)
}

func (self *User) registerUserWithEmail(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&UserWasRegisteredWithEmail{id, email, authToken})

	return nil
}

func (self *User) registerUserWithGoogle(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&UserWasRegisteredWithGoogle{id, email, authToken})

	return nil
}

func (self *User) registerUserWithFacebook(id uuid.UUID, email string, authToken string) error {
	self.trackChange(&UserWasRegisteredWithFacebook{id, email, authToken})

	return nil
}

func (self *User) changeEmailAddress(email string) error {
	self.trackChange(&UserEmailAddressWasChanged{self.id, email})

	return nil
}

// New creates an User
func New() *User {
	return &User{}
}
