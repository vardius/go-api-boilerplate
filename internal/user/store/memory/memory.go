// Package memory keeps the user service's stores in process memory, on a
// memdb.DB shared with the event store and the outbox, so they all commit
// and roll back together. Data is lost when the process stops.
package memory

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// Users is the users projection in memory.
type Users struct {
	db      *memdb.DB
	byID    map[uuid.UUID]store.User
	byEmail map[string]uuid.UUID
	bySub   map[string]uuid.UUID
}

var _ store.Users = (*Users)(nil)

// NewUsers returns an empty projection on db.
func NewUsers(db *memdb.DB) *Users {
	return &Users{db: db, byID: map[uuid.UUID]store.User{}, byEmail: map[string]uuid.UUID{}, bySub: map[string]uuid.UUID{}}
}

// Project implements store.Users.
func (s *Users) Project(_ context.Context, tx uow.Tx, events []es.Event) error {
	t, err := memdb.From(tx)
	if err != nil {
		return err
	}
	for _, e := range events {
		u, exists := s.byID[e.StreamID]
		switch p := e.Payload.(type) {
		case domain.Registered:
			if exists {
				continue
			}
			u = store.User{ID: p.UserID, Email: p.Email.String(), CreatedAt: e.OccurredAt}
		case domain.EmailChanged:
			u.Email = p.New.String()
		case domain.GoogleLinked:
			u.GoogleSubject = p.Subject
		default:
			return fmt.Errorf("memory: project %s: unknown event %T", e.Type, e.Payload)
		}
		if !exists && u.ID == uuid.Nil() {
			return fmt.Errorf("memory: project %s on user %s, who isn't registered", e.Type, e.StreamID)
		}
		if err := s.put(t, u); err != nil {
			return err
		}
	}
	return nil
}

// put stores u, checking the unique addresses and subjects, and registers
// the undo of the write with t.
func (s *Users) put(t *memdb.Tx, u store.User) error {
	if owner, ok := s.byEmail[u.Email]; ok && owner != u.ID {
		return store.ErrEmailTaken
	}
	if owner, ok := s.bySub[u.GoogleSubject]; ok && u.GoogleSubject != "" && owner != u.ID {
		return store.ErrGoogleTaken
	}
	old, existed := s.byID[u.ID]
	s.unindex(old, existed)
	s.index(u)
	t.OnRollback(func() {
		s.unindex(u, true)
		if existed {
			s.index(old)
		} else {
			delete(s.byID, u.ID)
		}
	})
	return nil
}

func (s *Users) index(u store.User) {
	s.byID[u.ID] = u
	s.byEmail[u.Email] = u.ID
	if u.GoogleSubject != "" {
		s.bySub[u.GoogleSubject] = u.ID
	}
}

func (s *Users) unindex(u store.User, ok bool) {
	if !ok {
		return
	}
	delete(s.byEmail, u.Email)
	if u.GoogleSubject != "" {
		delete(s.bySub, u.GoogleSubject)
	}
}

// ByID implements store.Users.
func (s *Users) ByID(_ context.Context, id uuid.UUID) (store.User, error) {
	return s.find("user "+id.String(), func() (uuid.UUID, bool) { return id, true })
}

// ByEmail implements store.Users.
func (s *Users) ByEmail(_ context.Context, email domain.Email) (store.User, error) {
	return s.find("user with that email", func() (uuid.UUID, bool) {
		id, ok := s.byEmail[email.String()]
		return id, ok
	})
}

// ByGoogleSubject implements store.Users.
func (s *Users) ByGoogleSubject(_ context.Context, subject string) (store.User, error) {
	return s.find("user with that Google account", func() (uuid.UUID, bool) {
		id, ok := s.bySub[subject]
		return id, ok
	})
}

// find returns the user whose ID key gives, reading both under one lock.
func (s *Users) find(what string, key func() (uuid.UUID, bool)) (store.User, error) {
	var u store.User
	var ok bool
	_ = s.db.Read(func() error {
		var id uuid.UUID
		if id, ok = key(); ok {
			u, ok = s.byID[id]
		}
		return nil
	})
	if !ok {
		return store.User{}, fmt.Errorf("memory: %s: %w", what, apperr.ErrNotFound)
	}
	return u, nil
}

// LoginCodes keeps login codes in memory.
type LoginCodes struct {
	db    *memdb.DB
	codes map[store.CodeHash]*loginCode
}

type loginCode struct {
	store.LoginCode
	used bool
}

var _ store.LoginCodes = (*LoginCodes)(nil)

// NewLoginCodes returns an empty code store on db.
func NewLoginCodes(db *memdb.DB) *LoginCodes {
	return &LoginCodes{db: db, codes: map[store.CodeHash]*loginCode{}}
}

// Add implements store.LoginCodes.
func (s *LoginCodes) Add(_ context.Context, tx uow.Tx, c store.LoginCode) error {
	t, err := memdb.From(tx)
	if err != nil {
		return err
	}
	if _, ok := s.codes[c.Hash]; ok {
		return fmt.Errorf("memory: add login code: %w", apperr.ErrConflict)
	}
	s.codes[c.Hash] = &loginCode{LoginCode: c}
	t.OnRollback(func() { delete(s.codes, c.Hash) })
	return nil
}

// Find implements store.LoginCodes.
func (s *LoginCodes) Find(_ context.Context, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	var c *loginCode
	_ = s.db.Read(func() error { c = s.usable(hash, purpose, now); return nil })
	if c == nil {
		return store.LoginCode{}, store.ErrCodeUnusable
	}
	return c.LoginCode, nil
}

// Consume implements store.LoginCodes.
func (s *LoginCodes) Consume(_ context.Context, tx uow.Tx, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	t, err := memdb.From(tx)
	if err != nil {
		return store.LoginCode{}, err
	}
	c := s.usable(hash, purpose, now)
	if c == nil {
		return store.LoginCode{}, store.ErrCodeUnusable
	}
	c.used = true
	t.OnRollback(func() { c.used = false })
	return c.LoginCode, nil
}

// usable returns the code with hash if it is for purpose, unused and
// unexpired at now, and nil otherwise. The caller holds the lock.
func (s *LoginCodes) usable(hash store.CodeHash, purpose store.Purpose, now time.Time) *loginCode {
	c, ok := s.codes[hash]
	if !ok || c.used || c.Purpose != purpose || !c.ExpiresAt.After(now) {
		return nil
	}
	return c
}

// DeleteExpired implements store.LoginCodes.
func (s *LoginCodes) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	var n int64
	err := s.db.Do(ctx, func(uow.Tx) error {
		for hash, c := range s.codes {
			if !c.ExpiresAt.After(now) {
				delete(s.codes, hash)
				n++
			}
		}
		return nil
	})
	return n, err
}
