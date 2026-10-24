package oauth2

import (
	"context"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

func TestClientTokensCarryTheirScopes(t *testing.T) {
	tests := []struct {
		scopes []string
		want   identity.Permission
	}{
		{scopes: []string{"user_read"}, want: identity.PermissionUserRead},
		{scopes: []string{"user_read", "user_write"}, want: identity.PermissionUserRead | identity.PermissionUserWrite},
		{scopes: []string{"all"}, want: identity.PermissionUserRead | identity.PermissionUserWrite},
	}

	for _, tt := range tests {
		t.Run(tt.scopes[0], func(t *testing.T) {
			ctx := context.Background()
			clientID, userID := uuid.New(), uuid.New()
			clients := memory.NewClientRepository(&config.Config{})
			if err := clients.Add(ctx, &client.WasCreated{ID: clientID, UserID: userID, Scopes: tt.scopes}); err != nil {
				t.Fatal(err)
			}

			authenticator := auth.NewSecretAuthenticator([]byte("test-secret-at-least-32-bytes-long"))
			access, _, err := NewJWTAccess(jwt.SigningMethodHS512, authenticator, clients).Token(ctx, &oauth2.GenerateBasic{
				Client:    &models.Client{ID: clientID.String()},
				UserID:    userID.String(),
				CreateAt:  time.Now(),
				TokenInfo: &models.Token{AccessCreateAt: time.Now(), AccessExpiresIn: time.Hour},
			}, false)
			if err != nil {
				t.Fatal(err)
			}

			claims, err := auth.NewClaimsProvider(authenticator).FromJWT(access)
			if err != nil {
				t.Fatal(err)
			}
			if got := claims.Identity.Permission; got != tt.want {
				t.Errorf("permission = %b, want %b", got, tt.want)
			}
		})
	}
}
