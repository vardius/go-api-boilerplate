package user

import (
	"encoding/json"
	"fmt"

	"github.com/asaskevich/govalidator"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

// EmailAddress is an email address value object
type EmailAddress string

// MarshalJSON implements Marshal interface
func (e EmailAddress) MarshalJSON() ([]byte, error) {
	if err := e.IsValid(); err != nil {
		return []byte("null"), err
	}

	jsn, err := json.Marshal(string(e))
	if err != nil {
		return jsn, errors.Wrap(fmt.Errorf("could not marshal EmailAddress %s", e))
	}

	return jsn, nil
}

// UnmarshalJSON implements Unmarshal interface
func (e *EmailAddress) UnmarshalJSON(b []byte) error {
	var value string

	err := json.Unmarshal(b, &value)
	if err != nil {
		return errors.Wrap(fmt.Errorf("could not unmarshal json %s", b))
	}

	*e = (EmailAddress)(value)

	return e.IsValid()
}

// IsValid returns error if value object is not valid
func (e EmailAddress) IsValid() error {
	if !govalidator.IsEmail(string(e)) {
		return errors.New("Invalid email address")
	}

	return nil
}
