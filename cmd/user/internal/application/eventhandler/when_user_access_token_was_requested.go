package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/eventbus"
)

// Claims Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	UserID string `json:"userId"`
	jwt.StandardClaims
}

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(db *sql.DB, repository persistence.UserRepository, secretKey string) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s\n", event.Payload)

		e := user.AccessTokenWasRequested{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		// Declare the expiration time of the token
		// here, we have kept it as 5 minutes
		expirationTime := time.Now().Add(5 * time.Minute)
		// Create the JWT claims, which includes the username and expiry time
		claims := &Claims{
			UserID: e.ID.String(),
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}

		var jwtKey = []byte(secretKey)

		// Declare the token with the algorithm used for signing, and the claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// Create the JWT string
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			// If there is an error in creating the JWT return an internal server error
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		magicLink := "https://go-api-boilerplate.me/users/v1/me?authToken=" + tokenString

		// @TODO: send token with an email as magic link
		log.Printf("[EventHandler] Magic link: %s\n", magicLink)
	}

	return fn
}
