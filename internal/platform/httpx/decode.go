package httpx

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// DecodeJSON decodes r's body, which must be application/json and at most
// maxBytes long, into a T. Unknown or duplicate fields, a value of the wrong
// type, trailing data and an oversize body are all apperr.ErrInvalid, with a
// message for the client; an oversize body is also an *http.MaxBytesError,
// which WriteError answers with 413.
func DecodeJSON[T any](r *http.Request, maxBytes int64) (T, error) {
	var zero T
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return zero, apperr.New(apperr.ErrInvalid, "The request body must be application/json.")
	}
	var v T
	if err := json.UnmarshalRead(http.MaxBytesReader(nil, r.Body, maxBytes), &v, json.RejectUnknownMembers(true)); err != nil {
		return zero, fmt.Errorf("decode json body: %w: %w", decodeProblem(err), err)
	}
	return v, nil
}

func decodeProblem(err error) error {
	if mbe, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return apperr.New(apperr.ErrInvalid, fmt.Sprintf("The request body is larger than %d bytes.", mbe.Limit))
	}
	if se, ok := errors.AsType[*json.SemanticError](err); ok {
		field := strings.Join(slices.Collect(se.JSONPointer.Tokens()), ".")
		switch {
		case errors.Is(err, json.ErrUnknownName):
			return apperr.Invalid(field, "is not a known field")
		case field != "":
			return apperr.Invalid(field, "has the wrong type")
		default:
			return apperr.New(apperr.ErrInvalid, "The request body has the wrong JSON type.")
		}
	}
	if se, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		if se.ByteOffset == 0 && errors.Is(err, io.ErrUnexpectedEOF) {
			return apperr.New(apperr.ErrInvalid, "The request body is empty.")
		}
		return apperr.New(apperr.ErrInvalid, "The request body is not valid JSON.")
	}
	return apperr.New(apperr.ErrInvalid, "The request body could not be read.")
}

// Page is a window into a list.
type Page struct {
	Limit, Offset int
}

// ParsePage reads ?limit= and ?offset=. A missing limit is defaultLimit; a
// limit outside 1..maxLimit or a negative offset is a ValidationError.
func ParsePage(r *http.Request, defaultLimit, maxLimit int) (Page, error) {
	q := r.URL.Query()
	p := Page{Limit: defaultLimit}
	fields := map[string]string{}
	if q.Has("limit") {
		n, err := strconv.Atoi(q.Get("limit"))
		if err != nil || n < 1 || n > maxLimit {
			fields["limit"] = fmt.Sprintf("must be a whole number from 1 to %d", maxLimit)
		}
		p.Limit = n
	}
	if q.Has("offset") {
		n, err := strconv.Atoi(q.Get("offset"))
		if err != nil || n < 0 {
			fields["offset"] = "must be a whole number, 0 or more"
		}
		p.Offset = n
	}
	if len(fields) > 0 {
		return Page{}, &apperr.ValidationError{Fields: fields}
	}
	return p, nil
}
