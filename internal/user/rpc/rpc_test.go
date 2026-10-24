package rpc_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	userv1 "github.com/vardius/go-api-boilerplate/gen/user/v1"
	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/rpc"
	"github.com/vardius/go-api-boilerplate/internal/user/store/memory"
)

var t0 = time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC)

type rig struct {
	iss    *authntest.Issuer
	app    *app.Service
	outbox *memdb.Outbox
	srv    *httptest.Server
}

func newRig(t *testing.T) *rig {
	t.Helper()
	db := memdb.New()
	web, _ := url.Parse("https://app.example.com")
	r := &rig{iss: authntest.NewIssuer(t), outbox: memdb.NewOutbox(db)}
	r.app = app.New(app.Deps{
		UnitOfWork: db,
		Events:     memdb.NewEventStore(db),
		Outbox:     r.outbox,
		Users:      memory.NewUsers(db),
		LoginCodes: memory.NewLoginCodes(db),
		WebURL:     web,
		Now:        func() time.Time { return t0 },
	})
	r.srv = httptest.NewServer(rpc.NewHandler(r.app, r.iss.Verifier(), slog.New(slog.DiscardHandler)))
	t.Cleanup(r.srv.Close)
	return r
}

// client calls as the holder of token.
func (r *rig) client(token string) userv1connect.LoginServiceClient {
	var interceptors []connect.ClientInterceptor
	if token != "" {
		interceptors = append(interceptors, connectx.Bearer(func(context.Context) (string, error) { return token, nil }))
	}
	return userv1connect.NewLoginServiceClient(connect.NewClient(connecthttp.NewTransport(r.srv.Client(), r.srv.URL), interceptors...))
}

func (r *rig) authToken(t *testing.T) string {
	return r.iss.Token(t, authn.Identity{Subject: "auth", ClientID: "auth", Scopes: []string{app.ScopeRedeemLoginCodes}})
}

// code asks for a login link to email and returns its code.
func (r *rig) code(t *testing.T, email string) string {
	t.Helper()
	if err := r.app.RequestLoginLink(t.Context(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	msgs, err := r.outbox.Claim(t.Context(), t0, time.Minute, 10)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("outbox holds %d messages, %v", len(msgs), err)
	}
	var link mail.Link
	if err := json.Unmarshal(msgs[0].Payload, &link); err != nil {
		t.Fatal(err)
	}
	if err := r.outbox.Ack(t.Context(), msgs[0].ID); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("code")
}

func TestRedeemLoginCode(t *testing.T) {
	r := newRig(t)
	code := r.code(t, "Ada@Example.com")
	c := r.client(r.authToken(t))

	res, err := c.RedeemLoginCode(t.Context(), &userv1.RedeemLoginCodeRequest{Code: code})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(res.GetUserId()); err != nil || res.GetEmail() != "ada@example.com" {
		t.Errorf("redeemed %v", res)
	}

	_, err = c.RedeemLoginCode(t.Context(), &userv1.RedeemLoginCodeRequest{Code: code})
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("redeeming again: %v, want unauthenticated", err)
	}
	if ce, ok := errors.AsType[*connect.Error](err); !ok || ce.Message() != "This sign-in code is unknown, used or expired." {
		t.Errorf("err = %v, want the app's message", err)
	}
}

func TestRedeemLoginCodeRefuses(t *testing.T) {
	r := newRig(t)
	code := r.code(t, "ada@example.com")
	user := uuid.NewV7()
	tests := []struct {
		name  string
		token string
		code  string
		want  connect.Code
	}{
		{"no token", "", code, connect.CodeUnauthenticated},
		{"a bad token", "not-a-jwt", code, connect.CodeUnauthenticated},
		{"a user's token", r.iss.Token(t, authn.Identity{Subject: user.String(), UserID: user, Scopes: []string{"user:read", "user:write"}}), code, connect.CodePermissionDenied},
		{"a client without the scope", r.iss.Token(t, authn.Identity{Subject: "app-1", ClientID: "app-1", Scopes: []string{"user:read"}}), code, connect.CodePermissionDenied},
		{"another client with the scope", r.iss.Token(t, authn.Identity{Subject: "app-1", ClientID: "app-1", Scopes: []string{app.ScopeRedeemLoginCodes}}), code, connect.CodePermissionDenied},
		{"a user's token with the scope", r.iss.Token(t, authn.Identity{Subject: user.String(), UserID: user, ClientID: "web", Scopes: []string{app.ScopeRedeemLoginCodes}}), code, connect.CodePermissionDenied},
		{"a tampered token", r.authToken(t) + "x", code, connect.CodeUnauthenticated},
		{"an expired token", r.iss.Token(t, authn.Identity{Subject: "auth", ClientID: "auth", Scopes: []string{app.ScopeRedeemLoginCodes}}, authntest.WithExpiry(-time.Minute)), code, connect.CodeUnauthenticated},
		{"a made-up code", r.authToken(t), "made-up", connect.CodeUnauthenticated},
		{"no code", r.authToken(t), "", connect.CodeUnauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.client(tt.token).RedeemLoginCode(t.Context(), &userv1.RedeemLoginCodeRequest{Code: tt.code})
			if got := connect.CodeOf(err); got != tt.want {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
	// None of the refusals used the code up.
	if _, err := r.client(r.authToken(t)).RedeemLoginCode(t.Context(), &userv1.RedeemLoginCodeRequest{Code: code}); err != nil {
		t.Errorf("redeeming after the refusals: %v", err)
	}
}

// AuthClientID is the client_id the auth service signs its own token with.
// The test reads the auth service's source, since importing it would join
// the two services (internal/archtest).
func TestAuthClientIDIsTheAuthServiceTokens(t *testing.T) {
	src, err := os.ReadFile("../../auth/keys/signer.go")
	if err != nil {
		t.Fatal(err)
	}
	if want := "const ServiceClientID = " + strconv.Quote(rpc.AuthClientID) + "\n"; !strings.Contains(string(src), want) {
		t.Errorf("internal/auth/keys has no %q", want)
	}
}
