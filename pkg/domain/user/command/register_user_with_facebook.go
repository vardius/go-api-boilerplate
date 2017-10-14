package command

import (
	"encoding/json"
)

type RegisterUserWithFacebook struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func NewRegisterUserWithFacebook(payload json.RawMessage) (*RegisterUserWithFacebook, error) {
	var c RegisterUserWithFacebook

	err := json.Unmarshal(payload, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
