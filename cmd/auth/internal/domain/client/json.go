/*
Package client holds client domain logic
*/
package client

import (
	"encoding/json"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func unmarshalPayload(payload []byte, model interface{}) error {
	if err := json.Unmarshal(payload, model); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
