package oauth2

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

type TokenProvider interface {
	RetrieveToken(ctx context.Context, email string) (*oauth2.Token, error)
}

type credentialsProvider struct {
	secretKey string
	config    oauth2.Config
}

func NewCredentialsAuthenticator(secretKey string, config oauth2.Config) TokenProvider {
	return &credentialsProvider{
		secretKey,
		config,
	}
}

func (a *credentialsProvider) RetrieveToken(ctx context.Context, email string) (*oauth2.Token, error) {
	token, err := a.config.PasswordCredentialsToken(ctx, email, a.secretKey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return token, nil
}
