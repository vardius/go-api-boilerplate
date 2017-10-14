package command

import (
	"encoding/json"
)

type RegisterUserWithEmail struct {
	Email string `json:"email"`
}

func NewRegisterUserWithEmail(payload json.RawMessage) (*RegisterUserWithEmail, error) {
	var c RegisterUserWithEmail

	err := json.Unmarshal(payload, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
