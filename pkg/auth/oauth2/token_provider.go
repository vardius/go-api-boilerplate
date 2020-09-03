package oauth2

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

type TokenProvider interface {
	RetrievePasswordCredentialsToken(ctx context.Context, clientID, clientSecret, email string, scopes []string) (*oauth2.Token, error)
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

func (a *credentialsProvider) RetrievePasswordCredentialsToken(ctx context.Context, clientID, clientSecret, email string, scopes []string) (*oauth2.Token, error) {
	if len(scopes) == 0 {
		scopes = []string{"all"}
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://%s:%d/v1/authorize", a.host, a.port),
			TokenURL: fmt.Sprintf("http://%s:%d/v1/token", a.host, a.port),
		},
	}

	token, err := config.PasswordCredentialsToken(ctx, email, a.secretKey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return token, nil
}
