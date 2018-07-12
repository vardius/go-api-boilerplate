package response

import (
	"net/http"
	"strconv"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
)

// NewErrorFromHTTPStatus returns an app error based on http status code.
func NewErrorFromHTTPStatus(code int) error {
	return errors.New(http.StatusText(code), strconv.Itoa(code))
}
