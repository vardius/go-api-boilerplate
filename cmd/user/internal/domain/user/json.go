/*
Package user holds user domain logic
*/
package user

import (
	"encoding/json"
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func unmarshalPayload(payload []byte, model interface{}) error {
	if err := json.Unmarshal(payload, model); err != nil {
		return errors.Wrap(fmt.Errorf("error while trying to unmarshal payload (%s)", payload))
	}

	return nil
}
