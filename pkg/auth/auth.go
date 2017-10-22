package auth

type BasicAuthFunc func(username, password string) (*Identity, error)
type TokenAuthFunc func(token string) (*Identity, error)
