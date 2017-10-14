package command

import (
	"encoding/json"

	"github.com/google/uuid"
)

type ChangeUserEmailAddress struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func NewChangeUserEmailAddress(payload json.RawMessage) (*ChangeUserEmailAddress, error) {
	var c ChangeUserEmailAddress

	err := json.Unmarshal(payload, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
