/*
Package token holds token domain logic
*/
package token

import (
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func unmarshalPayload(payload []byte, model interface{}) error {
	err := json.Unmarshal(payload, model)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
