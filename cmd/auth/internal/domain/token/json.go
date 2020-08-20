/*
Package token holds token domain logic
*/
package token

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
