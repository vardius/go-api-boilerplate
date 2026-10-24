package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type tokenAuthorizer map[string]*identity.Identity

func (a tokenAuthorizer) Auth(_ context.Context, token string) (*identity.Identity, error) {
	if i, ok := a[token]; ok {
		return i, nil
	}
	return nil, errors.New("unknown token")
}

type recordingBus struct {
	mu        sync.Mutex
	published []domain.Command
}

func (b *recordingBus) Publish(_ context.Context, c domain.Command) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, c)
	return nil
}

func (b *recordingBus) Subscribe(context.Context, string, commandbus.CommandHandler) error {
	return nil
}

func (b *recordingBus) Unsubscribe(context.Context, string) error { return nil }

var (
	routerOnce sync.Once
	router     http.Handler
	bus        = &recordingBus{}
	victimID   = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
)

// testRouter builds the router once: its metrics middleware registers expvars, which can only happen once per process.
func testRouter(t *testing.T) http.Handler {
	t.Helper()
	routerOnce.Do(func() {
		repository := memory.NewUserRepository()
		if err := repository.Add(context.Background(), &user.WasRegisteredWithGoogle{
			ID:          victimID,
			Email:       user.EmailAddress("victim@example.com"),
			GoogleID:    "google-victim",
			AccessToken: "provider-access-token",
		}); err != nil {
			panic(err)
		}

		authorizer := tokenAuthorizer{
			"reader": {UserID: uuid.New(), Permission: identity.PermissionUserRead},
		}
		router = NewRouter(&config.Config{}, authorizer, repository, bus, nil, nil, nil)
	})
	return router
}

func serve(t *testing.T, method, target, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	testRouter(t).ServeHTTP(w, req)
	return w
}

func TestReadingUsersNeedsPermission(t *testing.T) {
	for _, target := range []string{"/v1/", "/v1/me", "/v1/" + victimID.String()} {
		t.Run(target, func(t *testing.T) {
			if w := serve(t, http.MethodGet, target, "", ""); w.Code != http.StatusUnauthorized {
				t.Errorf("anonymous GET %s = %d, want %d", target, w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestUserResponsesHideProviderTokens(t *testing.T) {
	for _, target := range []string{"/v1/", "/v1/" + victimID.String()} {
		t.Run(target, func(t *testing.T) {
			w := serve(t, http.MethodGet, target, "reader", "")
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d: %s", target, w.Code, http.StatusOK, w.Body)
			}
			if strings.Contains(w.Body.String(), "provider-access-token") {
				t.Errorf("GET %s leaks the provider access token: %s", target, w.Body)
			}
			if !strings.Contains(w.Body.String(), "victim@example.com") {
				t.Errorf("GET %s doesn't show the user: %s", target, w.Body)
			}
		})
	}
}

func TestDispatchRejectsProviderCommands(t *testing.T) {
	for _, command := range []string{user.RegisterUserWithGoogle, user.RegisterUserWithFacebook} {
		t.Run(command, func(t *testing.T) {
			body := `{"email":"victim@example.com","google_id":"attacker","facebook_id":"attacker","access_token":"x"}`
			w := serve(t, http.MethodPost, "/v1/dispatch/user/"+command, "", body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("POST dispatch %s = %d, want %d", command, w.Code, http.StatusBadRequest)
			}

			bus.mu.Lock()
			defer bus.mu.Unlock()
			for _, c := range bus.published {
				if c.GetName() == user.RegisterWithGoogleName || c.GetName() == user.RegisterWithFacebookName {
					t.Errorf("provider command %s reached the command bus", c.GetName())
				}
			}
		})
	}
}
