package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/memory"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type stubRepository struct {
	user  User
	saved bool
}

func (r *stubRepository) Save(context.Context, User) error {
	r.saved = true
	return nil
}

func (r *stubRepository) Get(context.Context, uuid.UUID) (User, error) { return r.user, nil }

func TestChangeEmailAddressOnlyByOwner(t *testing.T) {
	owner := uuid.New()
	command := ChangeEmailAddress{ID: owner, Email: EmailAddress("new@example.com")}

	tests := []struct {
		name     string
		identity *identity.Identity
		wantErr  error
	}{
		{name: "anonymous", wantErr: apperrors.ErrUnauthorized},
		{name: "another user", identity: &identity.Identity{UserID: uuid.New(), Permission: identity.PermissionUserWrite}, wantErr: apperrors.ErrForbidden},
		{name: "owner without write permission", identity: &identity.Identity{UserID: owner, Permission: identity.PermissionUserRead}, wantErr: apperrors.ErrForbidden},
		{name: "owner", identity: &identity.Identity{UserID: owner, Permission: identity.PermissionUserWrite}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &stubRepository{user: New()}
			handle := OnChangeEmailAddress(repository, memory.NewUserRepository())

			ctx := identity.ContextWithIdentity(context.Background(), tt.identity)
			err := handle(ctx, command)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if repository.saved {
					t.Fatal("the email change was saved")
				}
				return
			}
			if err != nil {
				t.Fatalf("owner's change failed: %v", err)
			}
			if !repository.saved {
				t.Fatal("owner's change wasn't saved")
			}
		})
	}
}
