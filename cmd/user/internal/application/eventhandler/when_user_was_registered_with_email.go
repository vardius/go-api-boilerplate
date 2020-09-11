package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

// WhenUserWasRegisteredWithEmail handles event
func WhenUserWasRegisteredWithEmail(db *sql.DB, repository persistence.UserRepository, tokenProvider oauth2.TokenProvider, authClient proto.AuthenticationServiceClient) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.WasRegisteredWithEmail{}

		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return apperrors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return apperrors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, e); err != nil {
			return apperrors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return apperrors.Wrap(err)
		}

		var scopes []string
		for _, scope := range oauth2.AllScopes {
			scopes = append(scopes, string(scope))
		}
		clientResp, err := authClient.CreateClient(ctx, &proto.CreateClientRequest{
			UserID: e.ID.String(),
			Domain: config.Env.App.Domain,
			Scopes: scopes,
		})
		if err != nil {
			return apperrors.Wrap(err)
		}

		token, err := tokenProvider.RetrievePasswordCredentialsToken(ctx, clientResp.ClientID, clientResp.ClientSecret, string(e.Email), oauth2.AllScopes)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if executioncontext.Has(ctx, executioncontext.LIVE) {
			if err := mailer.SendLoginEmail(ctx, "WhenUserWasRegisteredWithEmail", string(e.Email), token.AccessToken, e.RedirectPath); err != nil {
				return apperrors.Wrap(err)
			}
		}

		return nil
	}

	return fn
}
