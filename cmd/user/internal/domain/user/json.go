/*
Package user holds user domain logic
*/
package user

import (
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func unmarshalPayload(payload []byte, model interface{}) error {
	err := json.Unmarshal(payload, model)
	if err != nil {
		return errors.Wrapf(err, errors.INTERNAL, "Error while trying to unmarshal payload (%s)", payload)
	}

	return nil
}
