package oauth2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"golang.org/x/oauth2"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/middleware"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

var AllScopes = []Scope{ScopeAll}

type TokenProvider interface {
	RetrievePasswordCredentialsToken(ctx context.Context, clientID, clientSecret, email string, scopes []Scope) (*oauth2.Token, error)
}

type credentialsProvider struct {
	host      string
	port      int
	secretKey string
}

func NewCredentialsAuthenticator(host string, port int, secretKey string) TokenProvider {
	return &credentialsProvider{
		host,
		port,
		secretKey,
	}
}

func (a *credentialsProvider) RetrievePasswordCredentialsToken(ctx context.Context, clientID, clientSecret, email string, scopes []Scope) (*oauth2.Token, error) {
	if len(scopes) == 0 {
		return nil, apperrors.Wrap(fmt.Errorf("insufficent scope: %v", scopes))
	}

	var meta url.Values
	if m, ok := metadata.FromContext(ctx); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return nil, apperrors.Wrap(err)
		}

		meta.Set(middleware.InternalRequestMetadataKey, base64.RawURLEncoding.EncodeToString(data))
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthStyle: oauth2.AuthStyleInHeader,
			AuthURL:   fmt.Sprintf("http://%s:%d/v1/authorize", a.host, a.port),
			TokenURL:  fmt.Sprintf("http://%s:%d/v1/token?%s", a.host, a.port, meta.Encode()),
		},
	}

	for _, scope := range scopes {
		config.Scopes = append(config.Scopes, string(scope))
	}

	token, err := config.PasswordCredentialsToken(ctx, email, a.secretKey)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return token, nil
}
