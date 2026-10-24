package app

// OAuth 2 error codes, from RFC 6749 §4.1.2.1 and §5.2.
const (
	ErrorInvalidRequest          = "invalid_request"
	ErrorInvalidClient           = "invalid_client"
	ErrorInvalidGrant            = "invalid_grant"
	ErrorUnauthorizedClient      = "unauthorized_client"
	ErrorUnsupportedGrantType    = "unsupported_grant_type"
	ErrorInvalidScope            = "invalid_scope"
	ErrorAccessDenied            = "access_denied"
	ErrorUnsupportedResponseType = "unsupported_response_type"
)

// Reasons the web app's OAuth error page, {WEB_URL}/oauth/error?reason=…,
// is given. Every authorization request that fails before its user has
// decided on the consent page goes there, never to the redirect URI: any
// user can register a client, so a redirect URI is trusted only with an
// answer the user chose, or the endpoint is an open redirector (RFC 9700
// §4.11.2). The set is fixed: api/auth.openapi.yaml documents it and the
// web app explains each one.
const (
	// ReasonInvalidClient is a client_id that is missing, repeated or
	// unknown, or a client deleted before its user decided.
	ReasonInvalidClient = ErrorInvalidClient
	// ReasonInvalidRedirectURI is a redirect_uri that is missing, repeated
	// or not exactly one the client registered.
	ReasonInvalidRedirectURI = "invalid_redirect_uri"
	// ReasonInvalidRequest is a malformed request: a repeated parameter,
	// no response_type, a state that can't be echoed, or a consent form
	// that doesn't parse.
	ReasonInvalidRequest = ErrorInvalidRequest
	// ReasonPKCERequired is a request without a valid S256 code challenge
	// (RFC 9700 §2.1.1).
	ReasonPKCERequired = "pkce_required"
	// ReasonUnsupportedResponseType is a response_type other than code.
	ReasonUnsupportedResponseType = ErrorUnsupportedResponseType
	// ReasonInvalidScope is a scope that doesn't exist or that the client
	// didn't register.
	ReasonInvalidScope = ErrorInvalidScope
	// ReasonLoginRequired is a consent form sent without a session.
	ReasonLoginRequired = "login_required"
	// ReasonRequestExpired is a consent form for no pending request of
	// its user: it expired, was answered already, went with its deleted
	// client, or is someone else's.
	ReasonRequestExpired = "request_expired"
)

// Reasons returns every reason the OAuth error page is given.
func Reasons() []string {
	return []string{
		ReasonInvalidClient, ReasonInvalidRedirectURI, ReasonInvalidRequest, ReasonPKCERequired,
		ReasonUnsupportedResponseType, ReasonInvalidScope, ReasonLoginRequired, ReasonRequestExpired,
	}
}

// OAuthError is an error response of the token or revocation endpoint, RFC
// 6749 §5.2. Code and Description are written for the client; what
// happened, for the log, is in Error's text.
type OAuthError struct {
	Code        string
	Description string
	err         error
}

func oauthError(code, description string) *OAuthError {
	return &OAuthError{Code: code, Description: description}
}

// because records err, which the client never sees, as what happened.
func (e *OAuthError) because(err error) *OAuthError {
	e.err = err
	return e
}

func (e *OAuthError) Error() string {
	if e.err != nil {
		return e.Code + ": " + e.Description + ": " + e.err.Error()
	}
	return e.Code + ": " + e.Description
}

// AuthorizeError is an authorization request, or an answer to one, that
// failed before a redirect URI could be trusted with it. The user agent
// goes to the web app's error page with Code, one of the Reason constants,
// as the reason; Description is for the log.
type AuthorizeError struct {
	Code        string
	Description string
}

func (e *AuthorizeError) Error() string { return "authorize: " + e.Code + ": " + e.Description }
