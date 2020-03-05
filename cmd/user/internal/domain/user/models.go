package user

import (
	"encoding/json"

	"github.com/asaskevich/govalidator"
	"github.com/vardius/go-api-boilerplate/internal/errors"
)

// EmailAddress is an email address value object
type EmailAddress string

// UnmarshalJSON implements Unmarshal interface
func (e *EmailAddress) UnmarshalJSON(b []byte) error {
	var value string

	err := json.Unmarshal(b, &value)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Unmarshal error")
	}

	*e = (EmailAddress)(value)

	return e.IsValid()
}

// IsValid returns error if value object is not valid
func (e EmailAddress) IsValid() error {
	if !govalidator.IsEmail(string(e)) {
		return errors.New(errors.INTERNAL, "Invalid email address")
	}

	return nil
}
