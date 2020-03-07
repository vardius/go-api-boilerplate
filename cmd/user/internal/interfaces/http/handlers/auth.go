package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/response"
)

// User contains the information common amongst most OAuth and OAuth2 providers.
// All of the "raw" datafrom the provider can be found in the `RawData` field.
type User struct {
	ID uuid.UUID
	goth.User
}

// UserProviderClaims Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type UserProviderClaims struct {
	UserID string `json:"userId"`
	jwt.StandardClaims
}

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(cb commandbus.CommandBus, commandName, secretKey string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// try to get the user without re-authenticating
		gothUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			gothic.BeginAuthHandler(w, r)
		}

		if len(gothUser.Email) == 0 {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "gothUser is empty"))
			return
		}

		id, err := uuid.NewRandom()
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Could not generate new id"))
			return
		}

		userProfile, err := json.Marshal(User{id, gothUser})
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Could not json marshal userProfile"))
			return
		}

		c, err := user.NewCommandFromPayload(commandName, userProfile)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Invalid request commandAuthUserWithProvider"))
			return
		}

		out := make(chan error, 2)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			response.RespondJSONError(r.Context(), w, errors.Wrap(r.Context().Err(), errors.INTERNAL, "Invalid request"))
			return
		case err = <-out:
			if err != nil {
				response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Invalid request"))
				return
			}
		}

		// Declare the expiration time of the token
		// here, we have kept it as 5 minutes
		expirationTime := time.Now().Add(5 * time.Minute)
		// Create the JWT claims, which includes the username and expiry time
		claims := &UserProviderClaims{
			UserID: id.String(),
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

		response.RespondJSON(r.Context(), w, tokenString, http.StatusOK)
	}

	return http.HandlerFunc(fn)
}
