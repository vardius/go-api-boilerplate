/*
Package client holds client domain logic
*/
package client

import (
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func unmarshalPayload(payload []byte, model interface{}) error {
	if err := json.Unmarshal(payload, model); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
