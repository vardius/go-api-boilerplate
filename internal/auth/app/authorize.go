package app

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// Authorization is an authorization request whose client, redirect URI and
// parameters check out, waiting for its user to sign in and decide.
type Authorization struct {
	Client        store.Client
	RedirectURI   string
	Scopes        []string
	State         string
	CodeChallenge string
}

// maxStateBytes bounds the state a client may have echoed back.
const maxStateBytes = 1024

// authorizeParams are the parameters of an authorization request, none of
// which may be repeated (RFC 6749 §3.1).
var authorizeParams = []string{"response_type", "client_id", "redirect_uri", "scope", "state", "code_challenge", "code_challenge_method"}

// CheckAuthorization checks the query of an authorization request, RFC
// 6749 §4.1.1 with PKCE, RFC 7636 §4.3. A request that fails is an
// *AuthorizeError: an unknown client, or a redirect_uri that isn't exactly
// one the client registered, is answered by the web app's error page, and
// everything else at the redirect URI.
func (s *Service) CheckAuthorization(ctx context.Context, q url.Values) (Authorization, error) {
	if len(q["client_id"]) != 1 {
		return Authorization{}, &AuthorizeError{Code: ReasonInvalidClient, Description: "The request needs one client_id."}
	}
	c, ok, err := s.lookupClient(ctx, q.Get("client_id"))
	if err != nil {
		return Authorization{}, err
	}
	if !ok {
		return Authorization{}, &AuthorizeError{Code: ReasonInvalidClient, Description: "No client has this client_id."}
	}
	if len(q["redirect_uri"]) != 1 || !slices.Contains(c.RedirectURIs, q.Get("redirect_uri")) {
		return Authorization{}, &AuthorizeError{Code: ReasonInvalidRedirectURI, Description: "The redirect_uri is missing or isn't one the client registered."}
	}
	a := Authorization{Client: c, RedirectURI: q.Get("redirect_uri")}

	// From here on the redirect URI is trusted, so errors go back to it.
	fail := func(code, description string) (Authorization, error) {
		params := url.Values{"error": {code}, "error_description": {description}}
		if a.State != "" {
			params.Set("state", a.State)
		}
		return Authorization{}, &AuthorizeError{Code: code, Description: description, Location: s.redirect(a.RedirectURI, params)}
	}
	// The state goes back with any error but its own.
	if st := q["state"]; len(st) == 1 && validState(st[0]) {
		a.State = st[0]
	}
	for _, p := range authorizeParams {
		if len(q[p]) > 1 {
			return fail(ErrorInvalidRequest, "The "+p+" parameter is repeated.")
		}
	}
	if !validState(q.Get("state")) {
		return fail(ErrorInvalidRequest, fmt.Sprintf("The state must be at most %d printable ASCII characters.", maxStateBytes))
	}

	switch q.Get("response_type") {
	case "code":
	case "":
		return fail(ErrorInvalidRequest, "The response_type parameter is required.")
	default:
		return fail(ErrorUnsupportedResponseType, "Only the code response type is supported.")
	}
	// RFC 9700 §2.1.1: PKCE is required, and only S256, since plain
	// protects nothing once the request is seen. A missing method means
	// plain (RFC 7636 §4.3).
	switch {
	case q.Get("code_challenge") == "":
		return fail(ErrorInvalidRequest, "PKCE is required: send a code_challenge with code_challenge_method S256.")
	case q.Get("code_challenge_method") != "S256":
		return fail(ErrorInvalidRequest, "The code_challenge_method must be S256.")
	case !validChallenge(q.Get("code_challenge")):
		return fail(ErrorInvalidRequest, "The code_challenge must be the base64url SHA-256 of the code verifier.")
	}
	a.CodeChallenge = q.Get("code_challenge")

	scopes, err := requestedScopes(q.Get("scope"), c.Scopes)
	if oe, ok := errors.AsType[*OAuthError](err); ok {
		return fail(oe.Code, oe.Description)
	}
	if err != nil {
		return Authorization{}, err
	}
	a.Scopes = scopes
	return a, nil
}

// validState reports whether state is one to echo back: at most
// maxStateBytes of the visible ASCII RFC 6749 allows.
func validState(state string) bool {
	return len(state) <= maxStateBytes && !strings.ContainsFunc(state, func(r rune) bool { return r < 0x20 || r > 0x7e })
}

// redirect returns uri, a registered redirect URI, with params and the
// issuer (RFC 9207) added to its query, which is kept (RFC 6749 §3.1.2).
func (s *Service) redirect(uri string, params url.Values) string {
	u, err := url.Parse(uri)
	if err != nil {
		// Registration only takes URIs that parse.
		panic(fmt.Sprintf("app: a registered redirect URI doesn't parse: %v", err))
	}
	q := u.Query()
	maps.Copy(q, params)
	q.Set("iss", s.signer.Issuer())
	u.RawQuery = q.Encode()
	return u.String()
}

// RequestConsent keeps a, which user is signed in for, until user decides
// on it or AuthorizationRequestTTL passes, and returns its ID.
func (s *Service) RequestConsent(ctx context.Context, user uuid.UUID, a Authorization) (uuid.UUID, error) {
	r := store.AuthorizationRequest{
		// Random, so that its ID says nothing; it is bound to user too.
		ID:            uuid.NewV4(),
		UserID:        user,
		ClientID:      a.Client.ID,
		RedirectURI:   a.RedirectURI,
		Scopes:        a.Scopes,
		State:         a.State,
		CodeChallenge: a.CodeChallenge,
		ExpiresAt:     s.clock().Add(AuthorizationRequestTTL),
	}
	if err := s.store.CreateAuthorizationRequest(ctx, r); err != nil {
		return uuid.Nil(), fmt.Errorf("keep an authorization request: %w", err)
	}
	return r.ID, nil
}

// Consent is what the consent page shows.
type Consent struct {
	ClientID    string
	ClientName  string
	Scopes      []Scope
	RedirectURI string
}

var errNoRequest = apperr.New(apperr.ErrNotFound, "No such authorization request.")

// Consent returns the details of user's pending request id. Another user's
// request is apperr.ErrNotFound.
func (s *Service) Consent(ctx context.Context, user, id uuid.UUID) (Consent, error) {
	r, err := s.store.AuthorizationRequest(ctx, id, user, s.clock())
	if errors.Is(err, apperr.ErrNotFound) {
		return Consent{}, fmt.Errorf("authorization request %s: %w", id, errNoRequest)
	}
	if err != nil {
		return Consent{}, fmt.Errorf("authorization request %s: %w", id, err)
	}
	c, ok, err := s.lookupClient(ctx, r.ClientID)
	if err != nil {
		return Consent{}, err
	}
	if !ok {
		return Consent{}, fmt.Errorf("authorization request %s: client gone: %w", id, errNoRequest)
	}
	return Consent{ClientID: c.ID, ClientName: c.Name, Scopes: describe(r.Scopes), RedirectURI: r.RedirectURI}, nil
}

// Decide answers user's pending request id and returns where to send the
// user agent: back to the client with an authorization code when allow is
// set, with access_denied otherwise. A request that is gone, or whose
// client is, is an *AuthorizeError without a Location.
func (s *Service) Decide(ctx context.Context, user, id uuid.UUID, allow bool) (string, error) {
	now := s.clock()
	r, err := s.store.TakeAuthorizationRequest(ctx, id, user, now)
	if errors.Is(err, apperr.ErrNotFound) {
		return "", &AuthorizeError{Code: ReasonInvalidRequest, Description: "No authorization request is waiting for this decision."}
	}
	if err != nil {
		return "", fmt.Errorf("take authorization request %s: %w", id, err)
	}
	params := url.Values{}
	if r.State != "" {
		params.Set("state", r.State)
	}
	if !allow {
		params.Set("error", ErrorAccessDenied)
		params.Set("error_description", "The user denied the request.")
		return s.redirect(r.RedirectURI, params), nil
	}
	if _, ok, err := s.lookupClient(ctx, r.ClientID); err != nil || !ok {
		if err == nil {
			err = &AuthorizeError{Code: ReasonInvalidClient, Description: "The client was deleted."}
		}
		return "", err
	}

	code := newSecret()
	c := store.AuthorizationCode{
		Hash:          store.HashToken(code),
		ClientID:      r.ClientID,
		UserID:        user,
		RedirectURI:   r.RedirectURI,
		Scopes:        r.Scopes,
		CodeChallenge: r.CodeChallenge,
		ExpiresAt:     now.Add(AuthorizationCodeTTL),
		GrantID:       uuid.Nil(),
	}
	if err := s.store.CreateAuthorizationCode(ctx, c); err != nil {
		return "", fmt.Errorf("issue an authorization code: %w", err)
	}
	params.Set("code", code)
	return s.redirect(r.RedirectURI, params), nil
}
