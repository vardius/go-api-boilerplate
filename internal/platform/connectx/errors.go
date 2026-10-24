// Package connectx fits Connect RPCs to the platform: apperr kinds to and
// from Connect codes, and access tokens on both ends of a call.
//
// A server puts Errors outermost, then Authenticate:
//
//	server := connect.NewServer(
//		connectx.Errors(log),
//		connectx.Authenticate(verifier, "login-codes:redeem"),
//	)
//
// and a client attaches its token with Bearer, mapping what comes back
// with FromConnect:
//
//	client := connect.NewClient(transport, connectx.Bearer(tokens.Token))
//	err := connectx.FromConnect(client.CallUnary(ctx, spec, req, res))
package connectx

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// code is the one table from errors to Connect codes, the counterpart of
// httpx's table for HTTP. A conflict is aborted, Connect's code for a
// failed concurrency check: the caller may reload and retry.
func code(err error) connect.Code {
	switch {
	case errors.Is(err, apperr.ErrInvalid):
		return connect.CodeInvalidArgument
	case errors.Is(err, apperr.ErrUnauthorized):
		return connect.CodeUnauthenticated
	case errors.Is(err, apperr.ErrForbidden):
		return connect.CodePermissionDenied
	case errors.Is(err, apperr.ErrNotFound):
		return connect.CodeNotFound
	case errors.Is(err, apperr.ErrConflict):
		return connect.CodeAborted
	case errors.Is(err, context.DeadlineExceeded):
		return connect.CodeDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return connect.CodeCanceled
	default:
		return connect.CodeInternal
	}
}

// ToConnect returns the *connect.Error a handler fails with for err. The
// message is the client-safe text set with apperr.New, or a
// ValidationError's fields; anything else sends only the code. err stays
// attached as the cause, for logs.
//
// A *connect.Error the handler made itself goes out as it is. One a peer
// returned to the handler is not the handler's verdict: unless FromConnect
// gave it an apperr kind, it becomes internal.
func ToConnect(err error) *connect.Error {
	if err == nil {
		return nil
	}
	if ce, ok := errors.AsType[*connect.Error](err); ok && !ce.IsRemote() {
		return ce
	}
	c := code(err)
	var msg string
	switch c {
	case connect.CodeInternal, connect.CodeDeadlineExceeded, connect.CodeCanceled:
	default:
		msg = apperr.Message(err)
		if ve, ok := errors.AsType[*apperr.ValidationError](err); ok && msg == "" {
			msg = ve.Error()
		}
	}
	return connect.NewError(c, msg).WithCause(err)
}

// FromConnect turns the error from a Connect call into the apperr kind its
// code means, keeping the peer's message as the client-safe text, so a
// handler can return it as its own: a bad login code the user service
// refuses is the caller's 401 too. A deadline_exceeded is also a
// context.DeadlineExceeded. Other errors are returned as they are.
func FromConnect(err error) error {
	ce, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return err
	}
	var kind error
	switch ce.Code() {
	case connect.CodeInvalidArgument:
		kind = apperr.ErrInvalid
	case connect.CodeUnauthenticated:
		kind = apperr.ErrUnauthorized
	case connect.CodePermissionDenied:
		kind = apperr.ErrForbidden
	case connect.CodeNotFound:
		kind = apperr.ErrNotFound
	case connect.CodeAborted, connect.CodeAlreadyExists:
		kind = apperr.ErrConflict
	case connect.CodeDeadlineExceeded:
		return &peerError{kind: context.DeadlineExceeded, err: err}
	default:
		return err
	}
	return &peerError{kind: apperr.New(kind, ce.Message()), err: err}
}

// peerError is a call's error with the kind it maps to.
type peerError struct {
	kind error
	err  error
}

func (e *peerError) Error() string   { return e.err.Error() }
func (e *peerError) Unwrap() []error { return []error{e.kind, e.err} }
