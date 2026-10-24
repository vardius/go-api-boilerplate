// Package rpc serves the user service's Connect API, user.v1.LoginService,
// on its internal listener. Only the auth service calls it, with a token
// carrying the login-codes:redeem scope.
package rpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	userv1 "github.com/vardius/go-api-boilerplate/gen/user/v1"
	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
)

// NewHandler returns the Connect API on svc. Every call needs a token that
// verifier accepts with app.ScopeRedeemLoginCodes; failures and panics are
// logged to log.
func NewHandler(svc *app.Service, verifier authn.TokenVerifier, log *slog.Logger) http.Handler {
	server := connect.NewServer(
		connectx.Errors(log),
		connectx.Authenticate(verifier, app.ScopeRedeemLoginCodes),
	)
	userv1connect.RegisterLoginServiceHandler(server, loginService{app: svc})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	return mux
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
