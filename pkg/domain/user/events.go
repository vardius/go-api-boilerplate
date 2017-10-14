package user

import "github.com/google/uuid"

type UserWasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e UserWasRegisteredWithEmail) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type UserWasRegisteredWithGoogle struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e UserWasRegisteredWithGoogle) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type UserWasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func (e UserWasRegisteredWithFacebook) Apply(u *User) {
	u.id = e.ID
	u.email = e.Email
}

type UserEmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (e UserEmailAddressWasChanged) Apply(u *User) {
	u.email = e.Email
}
