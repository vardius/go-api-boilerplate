package command

import (
	"encoding/json"
)

type RegisterUserWithGoogle struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func NewRegisterUserWithGoogle(payload json.RawMessage) (*RegisterUserWithGoogle, error) {
	var c RegisterUserWithGoogle

	err := json.Unmarshal(payload, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
