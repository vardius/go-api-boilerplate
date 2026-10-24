package connectx_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/connect/v2/connectinprocess"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
)

// echo is a handwritten unary method: what it does depends on the request.
var echo = connect.Spec{Procedure: "/test.v1.EchoService/Echo", StreamType: connect.StreamTypeUnary}

func serveEcho(ctx context.Context, _ connect.Spec, stream connect.ServerStream) error {
	var req wrapperspb.StringValue
	if err := stream.Receive(&req); err != nil {
		return err
	}
	switch req.GetValue() {
	case "missing":
		return apperr.New(apperr.ErrNotFound, "No such thing.")
	case "invalid":
		return apperr.Invalid("value", "is wrong")
	case "conflict":
		return apperr.New(apperr.ErrConflict, "Reload and try again.")
	case "boom":
		return errors.New("db: password=hunter2 refused")
	case "panic":
		panic("handler bug")
	case "exhausted":
		return connect.NewError(connect.CodeResourceExhausted, "Slow down.")
	}
	id, _ := authn.FromContext(ctx)
	return stream.Send(wrapperspb.String(id.Subject + " " + strings.Join(id.Scopes, ",")))
}

// logBuffer is written by the server's goroutines and read by the test.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *logBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

type rig struct {
	iss  *authntest.Issuer
	logs *logBuffer
	srv  *connect.Server
	url  string
	http *http.Client
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{iss: authntest.NewIssuer(t), logs: &logBuffer{}}
	r.srv = connect.NewServer(
		connectx.Errors(slog.New(slog.NewJSONHandler(r.logs, nil))),
		connectx.Authenticate(r.iss.Verifier(), "login-codes:redeem"),
	)
	r.srv.Register(connect.Method{Spec: echo, Handler: serveEcho})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, r.srv)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	r.url, r.http = hs.URL, hs.Client()
	return r
}

func (r *rig) service(t *testing.T, opts ...authntest.Option) string {
	return r.iss.Token(t, authn.Identity{ClientID: "auth-service", Scopes: []string{"login-codes:redeem"}}, opts...)
}

func (r *rig) client(interceptors ...connect.ClientInterceptor) *connect.Client {
	return connect.NewClient(connecthttp.NewTransport(r.http, r.url), interceptors...)
}

func static(token string) connect.ClientInterceptor {
	return connectx.Bearer(func(context.Context) (string, error) { return token, nil })
}

func call(t *testing.T, c *connect.Client, value string) (string, error) {
	t.Helper()
	var res wrapperspb.StringValue
	err := c.CallUnary(t.Context(), echo, wrapperspb.String(value), &res)
	return res.GetValue(), err
}

func TestAuthenticatedCall(t *testing.T) {
	r := newRig(t)
	got, err := call(t, r.client(static(r.service(t))), "whoami")
	if err != nil {
		t.Fatal(err)
	}
	if got != "auth-service login-codes:redeem" {
		t.Errorf("handler saw %q", got)
	}
}

func TestAuthenticateRefuses(t *testing.T) {
	r := newRig(t)
	webUser := r.iss.Token(t, authn.Identity{UserID: uuid.NewV7(), Scopes: []string{"user:read"}})
	tests := []struct {
		name   string
		client *connect.Client
		code   connect.Code
		msg    string
	}{
		{"no token", r.client(), connect.CodeUnauthenticated, "This needs an access token."},
		{"empty token", r.client(static("")), connect.CodeUnauthenticated, "This needs an access token."},
		{"expired", r.client(static(r.service(t, authntest.WithExpiry(-time.Hour)))), connect.CodeUnauthenticated, ""},
		{"another audience", r.client(static(r.service(t, authntest.WithAudience("other-api")))), connect.CodeUnauthenticated, ""},
		{"missing scope", r.client(static(webUser)), connect.CodePermissionDenied, "This needs the login-codes:redeem scope."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := call(t, tt.client, "whoami")
			ce, ok := errors.AsType[*connect.Error](err)
			if !ok || ce.Code() != tt.code || ce.Message() != tt.msg {
				t.Fatalf("err = %v, want %v %q", err, tt.code, tt.msg)
			}
			if !ce.IsRemote() {
				t.Error("the refusal isn't marked as the server's")
			}
		})
	}
	if logs := r.logs.String(); logs != "" {
		t.Errorf("refusals were logged: %s", logs)
	}
}

func TestAuthenticateNeedsTheBearerScheme(t *testing.T) {
	r := newRig(t)
	basic := func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info, _ := connect.CallInfoForClientContext(ctx)
			info.RequestHeader().Set("Authorization", "Basic "+r.service(t))
			return next(ctx, spec)
		}
	}
	if _, err := call(t, r.client(basic), "whoami"); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("err = %v", err)
	}
}

func TestErrorsReachTheCaller(t *testing.T) {
	r := newRig(t)
	c := r.client(static(r.service(t)))
	tests := []struct {
		value  string
		code   connect.Code
		msg    string
		kind   error
		logged string
	}{
		{"missing", connect.CodeNotFound, "No such thing.", apperr.ErrNotFound, ""},
		{"invalid", connect.CodeInvalidArgument, "value: is wrong", apperr.ErrInvalid, ""},
		{"conflict", connect.CodeAborted, "Reload and try again.", apperr.ErrConflict, ""},
		{"exhausted", connect.CodeResourceExhausted, "Slow down.", nil, ""},
		{"boom", connect.CodeInternal, "", nil, "rpc failed"},
		{"panic", connect.CodeInternal, "", nil, "rpc panicked"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			r.logs.Reset()
			_, err := call(t, c, tt.value)
			ce, ok := errors.AsType[*connect.Error](err)
			if !ok || ce.Code() != tt.code || ce.Message() != tt.msg {
				t.Fatalf("err = %v, want %v %q", err, tt.code, tt.msg)
			}
			if strings.Contains(err.Error(), "hunter2") || strings.Contains(err.Error(), "handler bug") {
				t.Errorf("internal detail reached the caller: %v", err)
			}
			if tt.kind != nil {
				mapped := connectx.FromConnect(err)
				if !errors.Is(mapped, tt.kind) || apperr.Message(mapped) != tt.msg {
					t.Errorf("FromConnect = %v, want %v %q", mapped, tt.kind, tt.msg)
				}
			}
			logs := r.logs.String()
			if tt.logged == "" && logs != "" {
				t.Errorf("logged a caller's error: %s", logs)
			}
			if tt.logged != "" && (!strings.Contains(logs, tt.logged) || !strings.Contains(logs, echo.Procedure)) {
				t.Errorf("logs = %s, want %q", logs, tt.logged)
			}
		})
	}
	if !strings.Contains(r.logs.String(), `"stack"`) {
		t.Error("the panic was logged without its stack")
	}
}

// TestPeerErrorsAreNotForwarded calls through a front service whose handler
// returns the error of its own downstream call: unmapped, it must not pass
// as the front's verdict; mapped with FromConnect, it does.
func TestPeerErrorsAreNotForwarded(t *testing.T) {
	back := newRig(t)
	downstream := back.client(static(back.service(t)))

	front := connect.NewServer(connectx.Errors(slog.New(slog.DiscardHandler)))
	front.Register(connect.Method{Spec: echo, Handler: func(ctx context.Context, _ connect.Spec, stream connect.ServerStream) error {
		var req wrapperspb.StringValue
		if err := stream.Receive(&req); err != nil {
			return err
		}
		var res wrapperspb.StringValue
		err := downstream.CallUnary(ctx, echo, wrapperspb.String("missing"), &res)
		if req.GetValue() == "map" {
			err = connectx.FromConnect(err)
		}
		return err
	}})
	c := connect.NewClient(connectinprocess.New(front))

	if _, err := call(t, c, "raw"); connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("unmapped peer error reached the caller as %v", err)
	}
	_, err := call(t, c, "map")
	if ce, ok := errors.AsType[*connect.Error](err); !ok || ce.Code() != connect.CodeNotFound || ce.Message() != "No such thing." {
		t.Errorf("mapped peer error = %v", err)
	}
}

func TestBearer(t *testing.T) {
	r := newRig(t)
	calls := 0
	tokens := connectx.Bearer(func(ctx context.Context) (string, error) {
		calls++
		if calls > 1 {
			return "", errors.New("token endpoint down")
		}
		return r.service(t), nil
	})
	c := r.client(tokens)
	if _, err := call(t, c, "whoami"); err != nil {
		t.Fatal(err)
	}
	_, err := call(t, c, "whoami")
	if err == nil || !strings.Contains(err.Error(), "token endpoint down") || !strings.Contains(err.Error(), echo.Procedure) {
		t.Errorf("err = %v, want the token error", err)
	}
	if calls != 2 {
		t.Errorf("token ran %d times, want once per call", calls)
	}
}

// TestBearerWithoutCallInfo covers a ClientFunc driven without a Client,
// which doesn't attach a CallInfo of its own.
func TestBearerWithoutCallInfo(t *testing.T) {
	var header string
	next := func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
		info, _ := connect.CallInfoForClientContext(ctx)
		header = info.RequestHeader().Get("Authorization")
		return nil, nil
	}
	if _, err := static("abc")(next)(t.Context(), echo); err != nil {
		t.Fatal(err)
	}
	if header != "Bearer abc" {
		t.Errorf("Authorization = %q", header)
	}
}

// TestAuthenticateWithoutCallInfo covers a ServerFunc driven without a
// Server, which attaches the CallInfo.
func TestAuthenticateWithoutCallInfo(t *testing.T) {
	iss := authntest.NewIssuer(t)
	reached := false
	h := connectx.Authenticate(iss.Verifier())(func(context.Context, connect.Spec, connect.ServerStream) error {
		reached = true
		return nil
	})
	if err := h(t.Context(), echo, nil); connect.CodeOf(err) != connect.CodeUnauthenticated || reached {
		t.Errorf("err = %v, reached = %v", err, reached)
	}
}

func TestErrorsLetsAnAbortThrough(t *testing.T) {
	h := connectx.Errors(slog.New(slog.DiscardHandler))(func(context.Context, connect.Spec, connect.ServerStream) error {
		panic(http.ErrAbortHandler)
	})
	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	_ = h(t.Context(), echo, nil)
	t.Error("the abort was swallowed")
}
