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

// Reasons the web app's OAuth error page is given, for an authorization
// request that can't be answered at its redirect URI.
const (
	// ReasonInvalidClient is an unknown or malformed client_id.
	ReasonInvalidClient = ErrorInvalidClient
	// ReasonInvalidRedirectURI is a redirect_uri the client didn't
	// register.
	ReasonInvalidRedirectURI = "invalid_redirect_uri"
	// ReasonInvalidRequest is a consent form for no pending request.
	ReasonInvalidRequest = ErrorInvalidRequest
	// ReasonLoginRequired is a consent form sent without a session.
	ReasonLoginRequired = "login_required"
)

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

// AuthorizeError is an authorization request that failed. When the client
// and its redirect URI check out, Location is that URI with the error, RFC
// 6749 §4.1.2.1. When they don't, Location is empty: the user agent must
// not be sent to an unverified URI, so it goes to the web app's error page
// with Code as the reason.
type AuthorizeError struct {
	Code        string
	Description string
	Location    string
}

func (e *AuthorizeError) Error() string { return "authorize: " + e.Code + ": " + e.Description }
