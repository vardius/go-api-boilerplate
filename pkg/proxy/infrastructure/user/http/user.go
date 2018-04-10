/*
Package http provides user http client
*/
package http

import (
	"errors"
	"io/ioutil"
	nethttp "net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/proto"
	"github.com/vardius/gorouter"
)

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New("Empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = errors.New("Invalid request URL params")

// FromGRPC wraps user gRPC client with http.Handler
func FromGRPC(c user_proto.UserClient) nethttp.Handler {
	fn := func(w nethttp.ResponseWriter, r *nethttp.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), response.HTTPError{
				Code:    nethttp.StatusBadRequest,
				Error:   ErrEmptyRequestBody,
				Message: ErrEmptyRequestBody.Error(),
			})
			return
		}

		params, ok := gorouter.FromContext(r.Context())
		if !ok {
			response.WithError(r.Context(), response.HTTPError{
				Code:    nethttp.StatusBadRequest,
				Error:   ErrInvalidURLParams,
				Message: ErrInvalidURLParams.Error(),
			})
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.WithError(r.Context(), response.HTTPError{
				Code:    nethttp.StatusBadRequest,
				Error:   e,
				Message: "Invalid request body",
			})
			return
		}

		_, e = c.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
			Name:    params.Value("command"),
			Payload: body,
		})
		if e != nil {
			response.WithError(r.Context(), response.HTTPError{
				Code:    nethttp.StatusBadRequest,
				Error:   e,
				Message: "Invalid request",
			})
			return
		}

		w.WriteHeader(nethttp.StatusCreated)

		return
	}

	return nethttp.HandlerFunc(fn)
}
