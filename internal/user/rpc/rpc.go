// Package rpc serves the user service's Connect API, user.v1.LoginService,
// on its internal listener. Only the auth service calls it, with its own
// token: client_id auth, carrying the login-codes:redeem scope.
package rpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	userv1 "github.com/vardius/go-api-boilerplate/gen/user/v1"
	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
)

// AuthClientID is the client_id of the auth service's own token, the only
// caller of this API. It is keys.ServiceClientID in internal/auth/keys,
// which this package may not import (internal/archtest); the two must
// match.
const AuthClientID = "auth"

// NewHandler returns the Connect API on svc. Every call needs a token that
// verifier accepts with app.ScopeRedeemLoginCodes, issued to AuthClientID;
// failures and panics are logged to log.
func NewHandler(svc *app.Service, verifier authn.TokenVerifier, log *slog.Logger) http.Handler {
	server := connect.NewServer(
		connectx.Errors(log),
		connectx.Authenticate(verifier, app.ScopeRedeemLoginCodes),
		onlyAuth,
	)
	userv1connect.RegisterLoginServiceHandler(server, loginService{app: svc})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	return mux
}

// onlyAuth refuses a token issued to any client but the auth service with
// permission_denied, even one that holds the scope: the scope alone
// should be enough, and this keeps it so if it ever leaks to a client.
func onlyAuth(next connect.ServerFunc) connect.ServerFunc {
	return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
		if id, _ := authn.FromContext(ctx); id.ClientID != AuthClientID {
			return connectx.ToConnect(apperr.New(apperr.ErrForbidden, "Only the auth service may call this."))
		}
		return next(ctx, spec, stream)
	}
}

type loginService struct {
	app *app.Service
}

// RedeemLoginCode answers an unknown, used or expired code with
// unauthenticated, as the app's apperr.ErrUnauthorized maps.
func (s loginService) RedeemLoginCode(ctx context.Context, req *userv1.RedeemLoginCodeRequest) (*userv1.RedeemLoginCodeResponse, error) {
	u, err := s.app.RedeemLoginCode(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}
	return &userv1.RedeemLoginCodeResponse{UserId: u.ID.String(), Email: u.Email}, nil
}
