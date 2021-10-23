package memory

import (
	"context"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"sync"
)

// NewTokenRepository returns memory view model repository for token
func NewTokenRepository() persistence.TokenRepository {
	return &tokenRepository{tokens: make(map[string]persistence.Token)}
}

type tokenRepository struct {
	sync.RWMutex
	tokens map[string]persistence.Token
}

func (r *tokenRepository) Get(ctx context.Context, id string) (persistence.Token, error) {
	r.RLock()
	defer r.RUnlock()

	v, ok := r.tokens[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return v, nil
}

func (r *tokenRepository) GetByCode(ctx context.Context, code string) (persistence.Token, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.tokens {
		ti, err := v.TokenInfo()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		if ti.GetCode() == code {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *tokenRepository) GetByAccess(ctx context.Context, access string) (persistence.Token, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.tokens {
		ti, err := v.TokenInfo()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		if ti.GetAccess() == access {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *tokenRepository) GetByRefresh(ctx context.Context, refresh string) (persistence.Token, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.tokens {
		ti, err := v.TokenInfo()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		if ti.GetRefresh() == refresh {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *tokenRepository) Add(ctx context.Context, t persistence.Token) error {
	r.Lock()
	defer r.Unlock()

	r.tokens[t.GetID()] = t
	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.tokens[id]; ok {
		delete(r.tokens, id)
	}

	return nil
}

func (r *tokenRepository) FindAllByClientID(ctx context.Context, clientID string, limit, offset int64) ([]persistence.Token, error) {
	r.RLock()
	defer r.RUnlock()

	var i int64
	var tokens []persistence.Token
	for _, v := range r.tokens {
		if i < offset {
			continue
		}
		i++

		ti, err := v.TokenInfo()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		if ti.GetClientID() != clientID {
			continue
		}

		tokens = append(tokens, v)

		if int64(len(tokens)) == limit {
			return tokens, nil
		}
	}

	return tokens, nil
}

func (r *tokenRepository) CountByClientID(ctx context.Context, clientID string) (int64, error) {
	r.RLock()
	defer r.RUnlock()

	var i int64
	for _, v := range r.tokens {
		ti, err := v.TokenInfo()
		if err != nil {
			return 0, apperrors.Wrap(err)
		}
		if ti.GetClientID() != clientID {
			continue
		}
		i++
	}

	return i, nil
}
