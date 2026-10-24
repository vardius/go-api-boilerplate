package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vardius/gorouter/v4"
	"gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

func loginToken(t *testing.T, userID uuid.UUID, access string) *token.WasCreated {
	t.Helper()
	data, err := json.Marshal(&models.Token{
		ClientID:        uuid.Nil.String(),
		UserID:          userID.String(),
		Access:          access,
		AccessCreateAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		AccessExpiresIn: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &token.WasCreated{ID: uuid.New(), ClientID: uuid.Nil, UserID: userID, Data: data}
}

func TestListUserAuthTokensShowsOnlyCallersTokens(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	repository := memory.NewTokenRepository()
	for _, tk := range []*token.WasCreated{
		loginToken(t, alice, "alice-access-token"),
		loginToken(t, bob, "bob-access-token"),
	} {
		if err := repository.Add(context.Background(), tk); err != nil {
			t.Fatal(err)
		}
	}

	router := gorouter.New()
	router.GET("/users/{userID}/tokens", BuildListUserAuthTokensHandler(repository))

	get := func(caller, path uuid.UUID) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/users/"+path.String()+"/tokens", nil)
		req = req.WithContext(identity.ContextWithIdentity(req.Context(), &identity.Identity{UserID: caller}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("own tokens", func(t *testing.T) {
		w := get(alice, alice)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body)
		}

		var body struct {
			AuthTokens []tokenView `json:"auth_tokens"`
			Total      int64       `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Total != 1 || len(body.AuthTokens) != 1 {
			t.Fatalf("got %d of %d tokens, want 1 of 1", len(body.AuthTokens), body.Total)
		}
		if strings.Contains(w.Body.String(), "access-token") {
			t.Errorf("response contains token material: %s", w.Body)
		}
	})

	t.Run("someone else's tokens", func(t *testing.T) {
		w := get(alice, bob)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body)
		}
		if strings.Contains(w.Body.String(), "bob-access-token") {
			t.Errorf("response leaks bob's token: %s", w.Body)
		}
	})
}
