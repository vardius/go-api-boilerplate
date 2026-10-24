package connectx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect/v2"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
)

func TestToConnect(t *testing.T) {
	local := connect.NewError(connect.CodeResourceExhausted, "Slow down.")
	tests := []struct {
		name string
		err  error
		code connect.Code
		msg  string
	}{
		{"invalid", fmt.Errorf("parse: %w", apperr.New(apperr.ErrInvalid, "Bad input.")), connect.CodeInvalidArgument, "Bad input."},
		{"validation", fmt.Errorf("register: %w", &apperr.ValidationError{Fields: map[string]string{"email": "is not valid", "name": "is required"}}), connect.CodeInvalidArgument, "email: is not valid; name: is required"},
		{"bare kind", fmt.Errorf("token: %w", apperr.ErrUnauthorized), connect.CodeUnauthenticated, ""},
		{"forbidden", apperr.New(apperr.ErrForbidden, "No."), connect.CodePermissionDenied, "No."},
		{"not found", apperr.New(apperr.ErrNotFound, "No such user."), connect.CodeNotFound, "No such user."},
		{"conflict", apperr.New(apperr.ErrConflict, "Reload and try again."), connect.CodeAborted, "Reload and try again."},
		{"deadline", fmt.Errorf("query: %w", context.DeadlineExceeded), connect.CodeDeadlineExceeded, ""},
		{"canceled", fmt.Errorf("query: %w", context.Canceled), connect.CodeCanceled, ""},
		{"internal keeps its text", errors.New("dial tcp 10.0.0.7:3306: password=hunter2"), connect.CodeInternal, ""},
		{"local connect error", fmt.Errorf("limit: %w", local), connect.CodeResourceExhausted, "Slow down."},
		{"peer's error", connect.NewError(connect.CodeNotFound, "Peer's own words.").WithRemote(), connect.CodeInternal, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := connectx.ToConnect(tt.err)
			if ce.Code() != tt.code || ce.Message() != tt.msg {
				t.Errorf("ToConnect = %v %q, want %v %q", ce.Code(), ce.Message(), tt.code, tt.msg)
			}
			if ce.Code() != connect.CodeResourceExhausted && !errors.Is(ce, tt.err) {
				t.Error("the cause was dropped")
			}
		})
	}
	if connectx.ToConnect(nil) != nil {
		t.Error("ToConnect(nil) != nil")
	}
}

func TestFromConnect(t *testing.T) {
	tests := []struct {
		code connect.Code
		kind error
	}{
		{connect.CodeInvalidArgument, apperr.ErrInvalid},
		{connect.CodeUnauthenticated, apperr.ErrUnauthorized},
		{connect.CodePermissionDenied, apperr.ErrForbidden},
		{connect.CodeNotFound, apperr.ErrNotFound},
		{connect.CodeAborted, apperr.ErrConflict},
		{connect.CodeAlreadyExists, apperr.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			remote := connect.NewError(tt.code, "The peer says no.").WithRemote()
			err := connectx.FromConnect(fmt.Errorf("call: %w", remote))
			if !errors.Is(err, tt.kind) || apperr.Message(err) != "The peer says no." {
				t.Errorf("FromConnect = %v (message %q), want %v", err, apperr.Message(err), tt.kind)
			}
			if ce, ok := errors.AsType[*connect.Error](err); !ok || ce != remote {
				t.Error("the call's error is no longer reachable")
			}
			if err.Error() != "call: "+remote.Error() {
				t.Errorf("Error() = %q", err.Error())
			}
			// Returned by a handler, it goes out as the kind, not internal.
			if got := connectx.ToConnect(err); got.Code() != tt.code && tt.code != connect.CodeAlreadyExists {
				t.Errorf("forwarded as %v", got.Code())
			}
		})
	}

	deadline := connectx.FromConnect(connect.NewError(connect.CodeDeadlineExceeded, "").WithRemote())
	if !errors.Is(deadline, context.DeadlineExceeded) {
		t.Errorf("deadline_exceeded = %v", deadline)
	}
	unavailable := connect.NewError(connect.CodeUnavailable, "").WithRemote()
	if err := connectx.FromConnect(unavailable); err != error(unavailable) {
		t.Errorf("unavailable = %v, want it unchanged", err)
	}
	plain := errors.New("dial: refused")
	if err := connectx.FromConnect(plain); err != plain {
		t.Errorf("a non-Connect error = %v, want it unchanged", err)
	}
	if connectx.FromConnect(nil) != nil {
		t.Error("FromConnect(nil) != nil")
	}
}
