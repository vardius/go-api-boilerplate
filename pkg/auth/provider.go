package auth

type ClaimsProvider interface {
	FromJWT(jwt string) (Claims, error)
}

func NewClaimsProvider(authenticator Authenticator) ClaimsProvider {
	return &provider{
		authenticator: authenticator,
	}
}

type provider struct {
	authenticator Authenticator
}

func (p *provider) FromJWT(jwt string) (Claims, error) {
	var c Claims
	if err := p.authenticator.Verify(jwt, &c); err != nil {
		return c, err
	}

	return c, nil
}
