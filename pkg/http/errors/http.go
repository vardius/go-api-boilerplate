package errors

import (
	"net/http"
	"strconv"

	appErrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

type HttpError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

var errorCodeToHTTPStatusMap = map[string]int{
	// 4xx codes
	appErrors.INVALID:      http.StatusBadRequest,
	appErrors.UNAUTHORIZED: http.StatusUnauthorized,
	appErrors.FORBIDDEN:    http.StatusForbidden,
	appErrors.NOTFOUND:     http.StatusNotFound,
	appErrors.TIMEOUT:      http.StatusRequestTimeout,

	// 5xx codes
	appErrors.INTERNAL:          http.StatusInternalServerError,
	appErrors.TEMPORARYDISABLED: http.StatusServiceUnavailable,
}

// HTTPStatusCode returns the http status code based on error code if available.
// It returns StatusOK if the error is nil.
func HTTPStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	code := appErrors.ErrorCode(err)
	internalErr := appErrors.ErrorErr(err)

	if code != "" {
		statusCode, isMapped := errorCodeToHTTPStatusMap[code]
		if isMapped {
			return statusCode
		}

		codeAsInt, err := strconv.Atoi(code)
		if err != nil {
			return http.StatusInternalServerError
		}

		if http.StatusText(codeAsInt) != "" {
			return codeAsInt
		}
	} else if internalErr != nil {
		return HTTPStatusCode(internalErr)
	}

	return http.StatusInternalServerError
}
