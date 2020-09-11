package oauth2

type Scope string

const (
	ScopeAll       Scope = "all"
	ScopeUserRead  Scope = "user_read"
	ScopeUserWrite Scope = "user_write"
)
