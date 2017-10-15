package user

import (
	"reflect"

	"github.com/google/uuid"
)

var WasRegisteredWithEmailType = reflect.TypeOf((*WasRegisteredWithEmail)(nil)).String()
var WasRegisteredWithGoogleType = reflect.TypeOf((*WasRegisteredWithGoogle)(nil)).String()
var WasRegisteredWithFacebookType = reflect.TypeOf((*WasRegisteredWithFacebook)(nil)).String()
var EmailAddressWasChangedType = reflect.TypeOf((*EmailAddressWasChanged)(nil)).String()

type domainEvent interface {
	Apply(*User)
}

type WasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e WasRegisteredWithEmail) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type WasRegisteredWithGoogle struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e WasRegisteredWithGoogle) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type WasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e WasRegisteredWithFacebook) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (e EmailAddressWasChanged) Apply(u *User) {
	u.email = e.Email
}
