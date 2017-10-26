package auth

// BasicAuthFunc returns Identity from username and password combination
type BasicAuthFunc func(username, password string) (*Identity, error)

// TokenAuthFunc returns Identity from auth token
type TokenAuthFunc func(token string) (*Identity, error)
