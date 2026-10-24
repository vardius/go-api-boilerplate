package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

// Limits on a client's registration.
const (
	MaxClientNameRunes   = 100
	MaxRedirectURIs      = 10
	MaxRedirectURIBytes  = 2000
	MaxClientsPage       = 100
	maxClientSecretBytes = 256
)

// NewClient is a client to register.
type NewClient struct {
	Name         string
	RedirectURIs []string
	Scopes       []string
}

// CreateClient registers a client owned by owner and returns it with its
// secret, which is shown this once: only its hash is kept.
func (s *Service) CreateClient(ctx context.Context, owner uuid.UUID, in NewClient) (store.Client, config.Secret, error) {
	c, err := validateClient(in)
	if err != nil {
		return store.Client{}, "", err
	}
	secret := newSecret()
	hash, err := hashSecret(ctx, secret)
	if err != nil {
		return store.Client{}, "", fmt.Errorf("hash a client secret: %w", err)
	}
	c.ID, c.OwnerID, c.SecretHash, c.CreatedAt = uuid.New().String(), owner, hash, s.clock()
	if err := s.store.CreateClient(ctx, c); err != nil {
		return store.Client{}, "", fmt.Errorf("create client: %w", err)
	}
	return c, config.Secret(secret), nil
}

func validateClient(in NewClient) (store.Client, error) {
	fields := map[string]string{}
	name := strings.TrimSpace(in.Name)
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		fields["name"] = "is required"
	case n > MaxClientNameRunes:
		fields["name"] = fmt.Sprintf("is longer than %d characters", MaxClientNameRunes)
	}

	var uris []string
	switch {
	case len(in.RedirectURIs) == 0:
		fields["redirect_uris"] = "needs at least one URI"
	case len(in.RedirectURIs) > MaxRedirectURIs:
		fields["redirect_uris"] = fmt.Sprintf("has more than %d URIs", MaxRedirectURIs)
	default:
		// Keep the order they were given in, without repeats.
		for _, u := range in.RedirectURIs {
			if !slices.Contains(uris, u) {
				uris = append(uris, u)
			}
		}
		for _, u := range uris {
			if problem := redirectURIProblem(u); problem != "" {
				fields["redirect_uris"] = problem
				break
			}
		}
	}

	scopes, ok := canonicalScopes(in.Scopes)
	switch {
	case len(in.Scopes) == 0:
		fields["scopes"] = "needs at least one scope"
	case !ok:
		fields["scopes"] = "may only hold " + strings.Join(GrantableScopes(), ", ")
	}

	if len(fields) > 0 {
		return store.Client{}, &apperr.ValidationError{Fields: fields}
	}
	return store.Client{Name: name, RedirectURIs: uris, Scopes: scopes}, nil
}

// redirectURIProblem says what is wrong with a redirect URI to register,
// or "" when it is fine: an absolute https URL, or http on localhost, with
// no fragment or credentials (RFC 6749 §3.1.2, RFC 9700 §4.1.1).
func redirectURIProblem(raw string) string {
	if len(raw) > MaxRedirectURIBytes {
		return fmt.Sprintf("holds a URI longer than %d bytes", MaxRedirectURIBytes)
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil || !u.IsAbs() || u.Host == "" || u.Opaque != "":
		return "holds a URI that is not an absolute URL: " + raw
	case u.Fragment != "" || strings.Contains(raw, "#"):
		return "holds a URI with a fragment: " + raw
	case u.User != nil:
		return "holds a URI with credentials in it"
	case u.Scheme == "https":
		return ""
	case u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"):
		return ""
	default:
		return "holds a URI that is neither https nor http on localhost: " + raw
	}
}

// Clients returns limit of owner's clients from offset, oldest first.
// limit is capped at MaxClientsPage.
func (s *Service) Clients(ctx context.Context, owner uuid.UUID, limit, offset int) ([]store.Client, error) {
	limit = min(max(limit, 1), MaxClientsPage)
	offset = max(offset, 0)
	cs, err := s.store.Clients(ctx, owner, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	return cs, nil
}

// DeleteClient deletes owner's client id, revoking every grant to it.
// Another owner's client is apperr.ErrNotFound.
func (s *Service) DeleteClient(ctx context.Context, owner uuid.UUID, id string) error {
	err := errNoClient
	if validClientID(id) {
		err = s.store.DeleteClient(ctx, owner, id)
	}
	if errors.Is(err, apperr.ErrNotFound) {
		err = errNoClient
	}
	if err != nil {
		return fmt.Errorf("delete client %q: %w", id, err)
	}
	return nil
}

var errNoClient = apperr.New(apperr.ErrNotFound, "No such client.")

// AuthenticateClient returns the client id whose secret is secret. A
// wrong ID or secret is an *OAuthError invalid_client.
func (s *Service) AuthenticateClient(ctx context.Context, id string, secret config.Secret) (store.Client, error) {
	if id == "" || secret == "" {
		return store.Client{}, oauthError(ErrorInvalidClient, "Client authentication is required.")
	}
	refused := oauthError(ErrorInvalidClient, "The client ID or secret is wrong.")
	if !validClientID(id) || len(secret) > maxClientSecretBytes {
		return store.Client{}, refused
	}
	c, err := s.store.Client(ctx, id)
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return store.Client{}, refused.because(fmt.Errorf("no client %s", id))
	case err != nil:
		return store.Client{}, fmt.Errorf("authenticate client %s: %w", id, err)
	}
	ok, err := verifySecret(ctx, secret.Reveal(), c.SecretHash)
	switch {
	case err != nil:
		return store.Client{}, fmt.Errorf("authenticate client %s: %w", id, err)
	case !ok:
		return store.Client{}, refused.because(fmt.Errorf("wrong secret for client %s", id))
	}
	return c, nil
}

// lookupClient returns the client id, or false when there is none.
func (s *Service) lookupClient(ctx context.Context, id string) (store.Client, bool, error) {
	if !validClientID(id) {
		return store.Client{}, false, nil
	}
	c, err := s.store.Client(ctx, id)
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return store.Client{}, false, nil
	case err != nil:
		return store.Client{}, false, fmt.Errorf("look up client %s: %w", id, err)
	}
	return c, true, nil
}
