package connectx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"connectrpc.com/connect/v2"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// Errors maps every error a handler returns with ToConnect and turns a
// panic into an internal error. Internal failures and panics are logged to
// log with the procedure; nothing else is, since the caller gets those.
// It goes first, so it sees the errors of the interceptors after it.
func Errors(log *slog.Logger) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) (err error) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if e, ok := v.(error); ok && errors.Is(e, http.ErrAbortHandler) {
					panic(v)
				}
				log.ErrorContext(ctx, "rpc panicked",
					"procedure", spec.Procedure,
					"panic", v,
					"stack", string(debug.Stack()),
				)
				err = connect.NewError(connect.CodeInternal, "")
			}()
			if err := next(ctx, spec, stream); err != nil {
				ce := ToConnect(err)
				if ce.Code() == connect.CodeInternal || ce.Code() == connect.CodeUnknown {
					log.ErrorContext(ctx, "rpc failed", "procedure", spec.Procedure, "err", err)
				}
				return ce
			}
			return nil
		}
	}
}

// Authenticate verifies the call's "Authorization: Bearer" token with v,
// answers unauthenticated without a good one and permission_denied when it
// lacks any of scopes, and puts the authn.Identity in the context.
func Authenticate(v authn.TokenVerifier, scopes ...string) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			var header string
			if info, ok := connect.CallInfoForServerContext(ctx); ok {
				header = info.RequestHeader().Get("Authorization")
			}
			scheme, token, _ := strings.Cut(header, " ")
			token = strings.TrimSpace(token)
			if !strings.EqualFold(scheme, "Bearer") || token == "" {
				return ToConnect(apperr.New(apperr.ErrUnauthorized, "This needs an access token."))
			}
			id, err := v.Verify(ctx, token)
			if err != nil {
				return ToConnect(err)
			}
			if err := authn.MissingScope(id, scopes); err != nil {
				return ToConnect(err)
			}
			return next(authn.NewContext(ctx, id), spec, stream)
		}
	}
}

// Bearer sends the token from token as "Authorization: Bearer" on every
// call. token runs per call, so it can hand out a cached token and renew it
// when it nears expiry; its error fails the call before anything is sent.
func Bearer(token func(context.Context) (string, error)) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			t, err := token(ctx)
			if err != nil {
				return nil, fmt.Errorf("access token for %s: %w", spec.Procedure, err)
			}
			info, ok := connect.CallInfoForClientContext(ctx)
			if !ok {
				ctx, info = connect.NewClientContext(ctx)
			}
			info.RequestHeader().Set("Authorization", "Bearer "+t)
			return next(ctx, spec)
		}
	}
}
