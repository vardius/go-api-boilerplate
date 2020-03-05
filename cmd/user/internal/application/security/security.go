package application

import (
	"context"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"

	user_persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	http_authenticator "github.com/vardius/go-api-boilerplate/internal/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/internal/identity"
)

// InternalCustomClaims used for internal registration only
type InternalCustomClaims struct {
	UserID string `json:"userId"`
	jwt.StandardClaims
}

// TokenAuthHandler provides token auth function
func TokenAuthHandler(repository user_persistence.UserRepository, secretKey string) http_authenticator.TokenAuthFunc {
	fn := func(tokenString string) (identity.Identity, error) {
		// Parse takes the token string and a function for looking up the key. The latter is especially
		// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
		// head of the token to identify which key to use, but the parsed token (head and claims) is provided
		// to the callback, providing flexibility.
		token, err := jwt.ParseWithClaims(tokenString, &InternalCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(secretKey), nil
		})

		claims, ok := token.Claims.(*InternalCustomClaims)

		if !ok || !token.Valid {
			return identity.NullIdentity, errors.Wrap(err, errors.UNAUTHORIZED, "Could not verify token")
		}

		user, err := repository.Get(context.Background(), claims.UserID)
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err, errors.INTERNAL, "Could not find user for token")
		}

		i := identity.Identity{
			ID:    uuid.MustParse(user.GetID()),
			Token: tokenString,
			Email: user.GetEmail(),
			Roles: []string{"USER"},
		}

		return i, nil
	}

	return fn
}
