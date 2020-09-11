package user

import (
	"encoding/json"
	"fmt"

	"github.com/asaskevich/govalidator"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
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
		return jsn, apperrors.Wrap(fmt.Errorf("could not marshal EmailAddress %s", e))
	}

	return jsn, nil
}

// UnmarshalJSON implements Unmarshal interface
func (e *EmailAddress) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return apperrors.Wrap(fmt.Errorf("could not unmarshal json %s", b))
	}

	*e = (EmailAddress)(value)

	return e.IsValid()
}

// IsValid returns error if value object is not valid
func (e EmailAddress) IsValid() error {
	if !govalidator.IsEmail(string(e)) {
		return apperrors.New(fmt.Sprintf("Invalid email address: %s", e))
	}

	return nil
}

func (e EmailAddress) String() string {
	return string(e)
}
