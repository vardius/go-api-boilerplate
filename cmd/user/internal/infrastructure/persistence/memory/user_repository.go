/*
Package memory holds view model repositories
*/
package memory

import (
	"context"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"sync"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
)

// NewUserRepository returns memory view model repository for user
func NewUserRepository() persistence.UserRepository {
	return &userRepository{users: make(map[string]persistence.User)}
}

type userRepository struct {
	sync.RWMutex
	users map[string]persistence.User
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int64) ([]persistence.User, error) {
	r.RLock()
	defer r.RUnlock()

	var i int64
	var users []persistence.User
	for _, v := range r.users {
		if i < offset {
			continue
		}
		i++

		users = append(users, v)

		if int64(len(users)) == limit {
			return users, nil
		}
	}

	return users, nil
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	r.RLock()
	defer r.RUnlock()

	v, ok := r.users[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return v, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (persistence.User, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.users {
		if v.GetEmail() == email {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *userRepository) GetByFacebookID(ctx context.Context, facebookID string) (persistence.User, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.users {
		if v.GetFacebookID() == facebookID {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *userRepository) GetByGoogleID(ctx context.Context, googleID string) (persistence.User, error) {
	r.RLock()
	defer r.RUnlock()

	for _, v := range r.users {
		if v.GetGoogleID() == googleID {
			return v, nil
		}
	}

	return nil, apperrors.ErrNotFound
}

func (r *userRepository) Add(ctx context.Context, u persistence.User) error {
	r.Lock()
	defer r.Unlock()

	r.users[u.GetID()] = u
	return nil
}

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	r.Lock()
	defer r.Unlock()

	v, ok := r.users[id]
	if !ok {
		return apperrors.ErrNotFound
	}

	r.users[id] = User{
		ID:         v.GetID(),
		Email:      email,
		FacebookID: v.GetFacebookID(),
		GoogleID:   v.GetGoogleID(),
		Role:       v.GetRole(),
	}

	return nil
}

func (r *userRepository) UpdateFacebookID(ctx context.Context, id, facebookID string) error {
	r.Lock()
	defer r.Unlock()

	v, ok := r.users[id]
	if !ok {
		return apperrors.ErrNotFound
	}

	r.users[id] = User{
		ID:         v.GetID(),
		Email:      v.GetEmail(),
		FacebookID: facebookID,
		GoogleID:   v.GetGoogleID(),
		Role:       v.GetRole(),
	}

	return nil
}

func (r *userRepository) UpdateGoogleID(ctx context.Context, id, googleID string) error {
	r.Lock()
	defer r.Unlock()

	v, ok := r.users[id]
	if !ok {
		return apperrors.ErrNotFound
	}

	r.users[id] = User{
		ID:         v.GetID(),
		Email:      v.GetEmail(),
		FacebookID: v.GetFacebookID(),
		GoogleID:   googleID,
		Role:       v.GetRole(),
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.users[id]; ok {
		delete(r.users, id)
	}

	return nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	r.RLock()
	defer r.RUnlock()

	return int64(len(r.users)), nil
}
