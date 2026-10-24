// Package memory keeps the auth service's store in process memory, on a
// memdb.DB, for tests and for running without a database.
//
// Every write holds the DB's write lock, so a check and the write it guards
// happen together and single-use operations are atomic. Values are copied in
// and out, so callers never share a slice with the store.
package memory

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Store is a store.Store in memory.
type Store struct {
	db       *memdb.DB
	clients  map[string]store.Client
	grants   map[uuid.UUID]store.Grant
	refresh  map[store.Hash]*refreshToken
	families map[uuid.UUID]map[store.Hash]struct{} // a grant's refresh tokens
	requests map[uuid.UUID]store.AuthorizationRequest
	codes    map[store.Hash]store.AuthorizationCode
}

type refreshToken struct {
	store.RefreshToken
	rotated bool
}

var (
	_ store.Store  = (*Store)(nil)
	_ store.Purger = (*Store)(nil)
)

// New returns an empty Store on db.
func New(db *memdb.DB) *Store {
	return &Store{
		db:       db,
		clients:  map[string]store.Client{},
		grants:   map[uuid.UUID]store.Grant{},
		refresh:  map[store.Hash]*refreshToken{},
		families: map[uuid.UUID]map[store.Hash]struct{}{},
		requests: map[uuid.UUID]store.AuthorizationRequest{},
		codes:    map[store.Hash]store.AuthorizationCode{},
	}
}

func (s *Store) write(ctx context.Context, fn func() error) error {
	return s.db.Do(ctx, func(uow.Tx) error { return fn() })
}

func (s *Store) read(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.db.Read(fn)
}

func cloneClient(c store.Client) store.Client {
	c.RedirectURIs, c.Scopes = slices.Clone(c.RedirectURIs), slices.Clone(c.Scopes)
	return c
}

func cloneGrant(g store.Grant) store.Grant {
	g.Scopes = slices.Clone(g.Scopes)
	return g
}

// CreateClient implements store.Store.
func (s *Store) CreateClient(ctx context.Context, c store.Client) error {
	return s.write(ctx, func() error {
		if _, ok := s.clients[c.ID]; ok {
			return fmt.Errorf("memory: create client %s: %w", c.ID, apperr.ErrConflict)
		}
		s.clients[c.ID] = cloneClient(c)
		return nil
	})
}

// Client implements store.Store.
func (s *Store) Client(ctx context.Context, id string) (store.Client, error) {
	var c store.Client
	err := s.read(ctx, func() error {
		found, ok := s.clients[id]
		if !ok {
			return fmt.Errorf("memory: client %s: %w", id, apperr.ErrNotFound)
		}
		c = cloneClient(found)
		return nil
	})
	return c, err
}

// Clients implements store.Store.
func (s *Store) Clients(ctx context.Context, owner uuid.UUID, limit, offset int) ([]store.Client, error) {
	var out []store.Client
	err := s.read(ctx, func() error {
		var owned []store.Client
		for _, c := range s.clients {
			if c.OwnerID == owner {
				owned = append(owned, c)
			}
		}
		slices.SortFunc(owned, func(a, b store.Client) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
		for _, c := range owned[min(offset, len(owned)):min(offset+limit, len(owned))] {
			out = append(out, cloneClient(c))
		}
		return nil
	})
	return out, err
}

// DeleteClient implements store.Store.
func (s *Store) DeleteClient(ctx context.Context, owner uuid.UUID, id string) error {
	return s.write(ctx, func() error {
		c, ok := s.clients[id]
		if !ok || c.OwnerID != owner {
			return fmt.Errorf("memory: delete client %s: %w", id, apperr.ErrNotFound)
		}
		delete(s.clients, id)
		for gid, g := range s.grants {
			if g.ClientID == id {
				s.revoke(gid)
			}
		}
		maps.DeleteFunc(s.requests, func(_ uuid.UUID, r store.AuthorizationRequest) bool { return r.ClientID == id })
		maps.DeleteFunc(s.codes, func(_ store.Hash, c store.AuthorizationCode) bool { return c.ClientID == id })
		return nil
	})
}

// CreateGrant implements store.Store.
func (s *Store) CreateGrant(ctx context.Context, g store.Grant, first store.RefreshToken) error {
	return s.write(ctx, func() error { return s.createGrant(g, first) })
}

func (s *Store) createGrant(g store.Grant, first store.RefreshToken) error {
	if _, ok := s.grants[g.ID]; ok {
		return fmt.Errorf("memory: create grant %s: %w", g.ID, apperr.ErrConflict)
	}
	if _, ok := s.refresh[first.Hash]; ok {
		return fmt.Errorf("memory: create grant %s: refresh token: %w", g.ID, apperr.ErrConflict)
	}
	s.grants[g.ID] = cloneGrant(g)
	s.families[g.ID] = map[store.Hash]struct{}{}
	first.GrantID = g.ID
	s.addRefreshToken(first)
	return nil
}

// Grant implements store.Store.
func (s *Store) Grant(ctx context.Context, id uuid.UUID, now time.Time) (store.Grant, error) {
	var g store.Grant
	err := s.read(ctx, func() error {
		found, ok := s.grants[id]
		if !ok || !now.Before(found.ExpiresAt) {
			return fmt.Errorf("memory: grant %s: %w", id, apperr.ErrNotFound)
		}
		g = cloneGrant(found)
		return nil
	})
	return g, err
}

// Grants implements store.Store.
func (s *Store) Grants(ctx context.Context, user uuid.UUID, client string, now time.Time, limit int) ([]store.Grant, error) {
	var out []store.Grant
	err := s.read(ctx, func() error {
		for _, g := range s.grants {
			if g.UserID == user && g.ClientID == client && now.Before(g.ExpiresAt) {
				out = append(out, cloneGrant(g))
			}
		}
		slices.SortFunc(out, func(a, b store.Grant) int {
			return cmp.Or(b.LastUsedAt.Compare(a.LastUsedAt), cmp.Compare(string(b.ID[:]), string(a.ID[:])))
		})
		out = out[:min(limit, len(out))]
		return nil
	})
	return out, err
}

// RevokeGrant implements store.Store.
func (s *Store) RevokeGrant(ctx context.Context, id uuid.UUID) error {
	return s.write(ctx, func() error {
		s.revoke(id)
		return nil
	})
}

func (s *Store) addRefreshToken(t store.RefreshToken) {
	s.refresh[t.Hash] = &refreshToken{RefreshToken: t}
	s.families[t.GrantID][t.Hash] = struct{}{}
}

func (s *Store) revoke(id uuid.UUID) {
	for h := range s.families[id] {
		delete(s.refresh, h)
	}
	delete(s.families, id)
	delete(s.grants, id)
}

// RefreshTokenGrant implements store.Store.
func (s *Store) RefreshTokenGrant(ctx context.Context, h store.Hash, now time.Time) (store.Grant, bool, error) {
	var (
		g       store.Grant
		rotated bool
	)
	err := s.read(ctx, func() error {
		t, ok := s.refresh[h]
		if !ok || !now.Before(t.ExpiresAt) {
			return fmt.Errorf("memory: refresh token: %w", apperr.ErrNotFound)
		}
		g, rotated = cloneGrant(s.grants[t.GrantID]), t.rotated
		return nil
	})
	return g, rotated, err
}

// RotateRefreshToken implements store.Store.
func (s *Store) RotateRefreshToken(ctx context.Context, h store.Hash, next store.RefreshToken, now time.Time) error {
	return s.write(ctx, func() error {
		t, ok := s.refresh[h]
		if !ok || !now.Before(t.ExpiresAt) {
			return fmt.Errorf("memory: rotate refresh token: %w", apperr.ErrNotFound)
		}
		if t.rotated {
			return fmt.Errorf("memory: rotate refresh token: %w", store.ErrUsed)
		}
		if _, ok := s.refresh[next.Hash]; ok {
			return fmt.Errorf("memory: rotate refresh token: %w", apperr.ErrConflict)
		}
		t.rotated = true
		next.GrantID = t.GrantID
		s.addRefreshToken(next)
		g := s.grants[t.GrantID]
		g.LastUsedAt, g.ExpiresAt = now, next.ExpiresAt
		s.grants[t.GrantID] = g
		return nil
	})
}

// CreateAuthorizationRequest implements store.Store.
func (s *Store) CreateAuthorizationRequest(ctx context.Context, r store.AuthorizationRequest) error {
	return s.write(ctx, func() error {
		if _, ok := s.requests[r.ID]; ok {
			return fmt.Errorf("memory: create authorization request %s: %w", r.ID, apperr.ErrConflict)
		}
		r.Scopes = slices.Clone(r.Scopes)
		s.requests[r.ID] = r
		return nil
	})
}

func (s *Store) request(id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	r, ok := s.requests[id]
	if !ok || r.UserID != user || !now.Before(r.ExpiresAt) {
		return store.AuthorizationRequest{}, fmt.Errorf("memory: authorization request %s: %w", id, apperr.ErrNotFound)
	}
	r.Scopes = slices.Clone(r.Scopes)
	return r, nil
}

// AuthorizationRequest implements store.Store.
func (s *Store) AuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	var r store.AuthorizationRequest
	err := s.read(ctx, func() error {
		var err error
		r, err = s.request(id, user, now)
		return err
	})
	return r, err
}

// TakeAuthorizationRequest implements store.Store.
func (s *Store) TakeAuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	var r store.AuthorizationRequest
	err := s.write(ctx, func() error {
		var err error
		if r, err = s.request(id, user, now); err != nil {
			return err
		}
		delete(s.requests, id)
		return nil
	})
	return r, err
}

// CreateAuthorizationCode implements store.Store.
func (s *Store) CreateAuthorizationCode(ctx context.Context, c store.AuthorizationCode) error {
	return s.write(ctx, func() error {
		if _, ok := s.codes[c.Hash]; ok {
			return fmt.Errorf("memory: create authorization code: %w", apperr.ErrConflict)
		}
		c.Scopes = slices.Clone(c.Scopes)
		c.Used, c.GrantID = false, uuid.Nil()
		s.codes[c.Hash] = c
		return nil
	})
}

// AuthorizationCode implements store.Store.
func (s *Store) AuthorizationCode(ctx context.Context, h store.Hash) (store.AuthorizationCode, error) {
	var c store.AuthorizationCode
	err := s.read(ctx, func() error {
		found, ok := s.codes[h]
		if !ok {
			return fmt.Errorf("memory: authorization code: %w", apperr.ErrNotFound)
		}
		c = found
		c.Scopes = slices.Clone(found.Scopes)
		return nil
	})
	return c, err
}

// SpendAuthorizationCode implements store.Store.
func (s *Store) SpendAuthorizationCode(ctx context.Context, h store.Hash) error {
	return s.write(ctx, func() error { return s.useCode(h, uuid.Nil()) })
}

// RedeemAuthorizationCode implements store.Store.
func (s *Store) RedeemAuthorizationCode(ctx context.Context, h store.Hash, g store.Grant, first store.RefreshToken) error {
	return s.write(ctx, func() error {
		c, ok := s.codes[h]
		if !ok {
			return fmt.Errorf("memory: redeem authorization code: %w", apperr.ErrNotFound)
		}
		if c.Used {
			return fmt.Errorf("memory: redeem authorization code: %w", store.ErrUsed)
		}
		if err := s.createGrant(g, first); err != nil {
			return err
		}
		return s.useCode(h, g.ID)
	})
}

func (s *Store) useCode(h store.Hash, grant uuid.UUID) error {
	c, ok := s.codes[h]
	if !ok {
		return fmt.Errorf("memory: use authorization code: %w", apperr.ErrNotFound)
	}
	if c.Used {
		return fmt.Errorf("memory: use authorization code: %w", store.ErrUsed)
	}
	c.Used, c.GrantID = true, grant
	s.codes[h] = c
	return nil
}

// Purge implements store.Purger.
func (s *Store) Purge(ctx context.Context, now time.Time) error {
	return s.write(ctx, func() error {
		for id, g := range s.grants {
			if !now.Before(g.ExpiresAt) {
				s.revoke(id)
			}
		}
		for h, t := range s.refresh {
			if !now.Before(t.ExpiresAt) {
				delete(s.refresh, h)
				delete(s.families[t.GrantID], h)
			}
		}
		maps.DeleteFunc(s.requests, func(_ uuid.UUID, r store.AuthorizationRequest) bool { return !now.Before(r.ExpiresAt) })
		maps.DeleteFunc(s.codes, func(_ store.Hash, c store.AuthorizationCode) bool {
			return !now.Before(c.ExpiresAt.Add(store.UsedCodeRetention))
		})
		return nil
	})
}
