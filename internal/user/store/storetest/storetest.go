// Package storetest holds the contract every backend of the user service's
// stores must meet.
package storetest

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// Backend is one implementation of the stores, with the event store and
// unit of work they share.
type Backend struct {
	UnitOfWork uow.UnitOfWork
	Events     es.Store
	Users      store.Users
	LoginCodes store.LoginCodes
}

// Open returns an empty Backend that shares no data with any other.
type Open func(t *testing.T) Backend

// Test checks the rules every backend must follow. Each backend's tests
// call it:
//
//	func TestStores(t *testing.T) {
//		storetest.Test(t, func(*testing.T) storetest.Backend {
//			db := memdb.New()
//			return storetest.Backend{UnitOfWork: db, Events: memdb.NewEventStore(db), …}
//		})
//	}
func Test(t *testing.T, open Open) {
	t.Run("Users", func(t *testing.T) {
		t.Run("RegisteredUserIsFound", func(t *testing.T) { testRegisteredUserIsFound(t, open(t)) })
		t.Run("UnknownUserIsNotFound", func(t *testing.T) { testUnknownUserIsNotFound(t, open(t)) })
		t.Run("TwoUsersWithoutGoogleBothSave", func(t *testing.T) { testTwoUsersWithoutGoogle(t, open(t)) })
		t.Run("DuplicateEmailConflicts", func(t *testing.T) { testDuplicateEmail(t, open(t)) })
		t.Run("EmailChangeMovesTheAddress", func(t *testing.T) { testEmailChange(t, open(t)) })
		t.Run("EmailChangeToATakenAddressConflicts", func(t *testing.T) { testEmailChangeConflict(t, open(t)) })
		t.Run("LinkedGoogleAccountFindsTheUser", func(t *testing.T) { testLinkGoogle(t, open(t)) })
		t.Run("GoogleAccountOfAnotherUserConflicts", func(t *testing.T) { testGoogleConflict(t, open(t)) })
		t.Run("ProjectingTwiceChangesNothing", func(t *testing.T) { testProjectTwice(t, open(t)) })
		t.Run("RollbackDiscardsTheProjection", func(t *testing.T) { testProjectionRollback(t, open(t)) })
	})
	t.Run("LoginCodes", func(t *testing.T) {
		t.Run("CodeIsFoundThenConsumedOnce", func(t *testing.T) { testConsumeOnce(t, open(t)) })
		t.Run("CodeWithoutUserKeepsNoUser", func(t *testing.T) { testCodeWithoutUser(t, open(t)) })
		t.Run("ExpiredCodeFails", func(t *testing.T) { testExpiredCode(t, open(t)) })
		t.Run("CodeForAnotherPurposeFails", func(t *testing.T) { testWrongPurpose(t, open(t)) })
		t.Run("UnknownCodeFails", func(t *testing.T) { testUnknownCode(t, open(t)) })
		t.Run("RolledBackConsumeKeepsTheCode", func(t *testing.T) { testConsumeRollback(t, open(t)) })
		t.Run("ConcurrentConsumesOneWins", func(t *testing.T) { testConcurrentConsumes(t, open(t)) })
		t.Run("RepeatedHashFails", func(t *testing.T) { testRepeatedHash(t, open(t)) })
		t.Run("DeleteExpiredKeepsLiveCodes", func(t *testing.T) { testDeleteExpired(t, open(t)) })
	})
}

// t0 is the stores' clock, with the millisecond precision they keep. It is
// far ahead so that MongoDB's TTL monitor, which runs on the wall clock,
// never removes a code a test still needs.
var t0 = time.Date(2100, 1, 2, 9, 30, 15, 123_000_000, time.UTC)

var (
	ada   = mustEmail("ada@example.com")
	grace = mustEmail("grace@example.com")
)

func mustEmail(s string) domain.Email {
	e, err := domain.ParseEmail(s)
	if err != nil {
		panic(err)
	}
	return e
}

func repository(b Backend) *es.Repository[*domain.User] {
	r := es.NewRegistry()
	domain.RegisterEvents(r)
	return es.NewRepository(es.RepositoryConfig[*domain.User]{
		StreamType: domain.StreamType,
		New:        domain.New,
		Store:      b.Events,
		Registry:   r,
		Project:    b.Users.Project,
		Now:        func() time.Time { return t0 },
	})
}

func save(t *testing.T, b Backend, u *domain.User) ([]es.Event, error) {
	t.Helper()
	var events []es.Event
	err := b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error {
		var err error
		events, err = repository(b).Save(t.Context(), tx, u, es.Meta{})
		return err
	})
	return events, err
}

func mustSave(t *testing.T, b Backend, u *domain.User) []es.Event {
	t.Helper()
	events, err := save(t, b, u)
	if err != nil {
		t.Fatalf("save user %s: %v", u.ID(), err)
	}
	return events
}

func load(t *testing.T, b Backend, id uuid.UUID) *domain.User {
	t.Helper()
	u, err := repository(b).Load(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func register(t *testing.T, b Backend, email domain.Email) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	u := domain.New()
	if err := u.Register(id, email); err != nil {
		t.Fatal(err)
	}
	mustSave(t, b, u)
	return id
}

func change(t *testing.T, b Backend, id uuid.UUID, act func(*domain.User) error) error {
	t.Helper()
	u := load(t, b, id)
	if err := act(u); err != nil {
		t.Fatal(err)
	}
	_, err := save(t, b, u)
	return err
}

func byID(t *testing.T, b Backend, id uuid.UUID) store.User {
	t.Helper()
	u, err := b.Users.ByID(t.Context(), id)
	if err != nil {
		t.Fatalf("user %s: %v", id, err)
	}
	return u
}

func assertUser(t *testing.T, got, want store.User) {
	t.Helper()
	if got.ID != want.ID || got.Email != want.Email || got.GoogleSubject != want.GoogleSubject || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("user = %+v\nwant   %+v", got, want)
	}
}

func assertNotFound(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("%s: err = %v, want apperr.ErrNotFound", what, err)
	}
}

func testRegisteredUserIsFound(t *testing.T, b Backend) {
	id := register(t, b, ada)
	want := store.User{ID: id, Email: "ada@example.com", CreatedAt: t0}

	assertUser(t, byID(t, b, id), want)
	got, err := b.Users.ByEmail(t.Context(), ada)
	if err != nil {
		t.Fatal(err)
	}
	assertUser(t, got, want)
}

func testUnknownUserIsNotFound(t *testing.T, b Backend) {
	register(t, b, ada)
	_, err := b.Users.ByID(t.Context(), uuid.NewV7())
	assertNotFound(t, "ByID", err)
	_, err = b.Users.ByEmail(t.Context(), grace)
	assertNotFound(t, "ByEmail", err)
	_, err = b.Users.ByGoogleSubject(t.Context(), "1084")
	assertNotFound(t, "ByGoogleSubject", err)
}

func testTwoUsersWithoutGoogle(t *testing.T, b Backend) {
	first, second := register(t, b, ada), register(t, b, grace)
	assertUser(t, byID(t, b, first), store.User{ID: first, Email: "ada@example.com", CreatedAt: t0})
	assertUser(t, byID(t, b, second), store.User{ID: second, Email: "grace@example.com", CreatedAt: t0})

	// Not being linked is no subject, not an empty one.
	_, err := b.Users.ByGoogleSubject(t.Context(), "")
	assertNotFound(t, `ByGoogleSubject("")`, err)
}

func testDuplicateEmail(t *testing.T, b Backend) {
	first := register(t, b, ada)

	second := domain.New()
	if err := second.Register(uuid.NewV7(), mustEmail("ADA@Example.com")); err != nil {
		t.Fatal(err)
	}
	_, err := save(t, b, second)
	if !errors.Is(err, store.ErrEmailTaken) || !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("registering a taken address: err = %v, want store.ErrEmailTaken", err)
	}
	got, err := b.Users.ByEmail(t.Context(), ada)
	if err != nil || got.ID != first {
		t.Errorf("ByEmail = %v, %v; want the first user %s", got.ID, err, first)
	}
	if recs, err := b.Events.Load(t.Context(), second.ID()); err != nil || len(recs) != 0 {
		t.Errorf("the refused user's stream has %d events, %v; want none", len(recs), err)
	}
}

func testEmailChange(t *testing.T, b Backend) {
	id := register(t, b, ada)
	if err := change(t, b, id, func(u *domain.User) error { return u.ChangeEmail(grace) }); err != nil {
		t.Fatal(err)
	}

	assertUser(t, byID(t, b, id), store.User{ID: id, Email: "grace@example.com", CreatedAt: t0})
	got, err := b.Users.ByEmail(t.Context(), grace)
	if err != nil || got.ID != id {
		t.Errorf("ByEmail(new) = %v, %v", got.ID, err)
	}
	_, err = b.Users.ByEmail(t.Context(), ada)
	assertNotFound(t, "ByEmail(old)", err)

	// The old address is free for someone else.
	register(t, b, ada)
}

func testEmailChangeConflict(t *testing.T, b Backend) {
	register(t, b, ada)
	id := register(t, b, grace)

	err := change(t, b, id, func(u *domain.User) error { return u.ChangeEmail(ada) })
	if !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("changing to a taken address: err = %v, want store.ErrEmailTaken", err)
	}
	assertUser(t, byID(t, b, id), store.User{ID: id, Email: "grace@example.com", CreatedAt: t0})
	if u := load(t, b, id); u.Version() != 1 {
		t.Errorf("the refused change was stored: stream at version %d, want 1", u.Version())
	}
}

func testLinkGoogle(t *testing.T, b Backend) {
	id := register(t, b, ada)
	if err := change(t, b, id, func(u *domain.User) error { return u.LinkGoogle("1084") }); err != nil {
		t.Fatal(err)
	}

	want := store.User{ID: id, Email: "ada@example.com", GoogleSubject: "1084", CreatedAt: t0}
	assertUser(t, byID(t, b, id), want)
	got, err := b.Users.ByGoogleSubject(t.Context(), "1084")
	if err != nil {
		t.Fatal(err)
	}
	assertUser(t, got, want)
}

func testGoogleConflict(t *testing.T, b Backend) {
	first := register(t, b, ada)
	if err := change(t, b, first, func(u *domain.User) error { return u.LinkGoogle("1084") }); err != nil {
		t.Fatal(err)
	}
	second := register(t, b, grace)

	err := change(t, b, second, func(u *domain.User) error { return u.LinkGoogle("1084") })
	if !errors.Is(err, store.ErrGoogleTaken) || !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("linking another user's Google account: err = %v, want store.ErrGoogleTaken", err)
	}
	got, err := b.Users.ByGoogleSubject(t.Context(), "1084")
	if err != nil || got.ID != first {
		t.Errorf("ByGoogleSubject = %v, %v; want the first user %s", got.ID, err, first)
	}
	assertUser(t, byID(t, b, second), store.User{ID: second, Email: "grace@example.com", CreatedAt: t0})
}

func testProjectTwice(t *testing.T, b Backend) {
	u := domain.New()
	id := uuid.NewV7()
	for _, act := range []func() error{
		func() error { return u.Register(id, ada) },
		func() error { return u.LinkGoogle("1084") },
		func() error { return u.ChangeEmail(grace) },
	} {
		if err := act(); err != nil {
			t.Fatal(err)
		}
	}
	events := mustSave(t, b, u)

	err := b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error { return b.Users.Project(t.Context(), tx, events) })
	if err != nil {
		t.Fatalf("projecting the same events again: %v", err)
	}
	assertUser(t, byID(t, b, id), store.User{ID: id, Email: "grace@example.com", GoogleSubject: "1084", CreatedAt: t0})
	_, err = b.Users.ByEmail(t.Context(), ada)
	assertNotFound(t, "ByEmail(old)", err)
}

func testProjectionRollback(t *testing.T, b Backend) {
	u := domain.New()
	if err := u.Register(uuid.NewV7(), ada); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("outbox failed")
	err := b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error {
		if _, err := repository(b).Save(t.Context(), tx, u, es.Meta{}); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Do = %v, want the function's error", err)
	}
	_, err = b.Users.ByID(t.Context(), u.ID())
	assertNotFound(t, "ByID after rollback", err)
	register(t, b, ada)
}

var hashSeq struct {
	sync.Mutex
	n int
}

// code returns a code for purpose with a hash no other test uses.
func code(purpose store.Purpose, userID uuid.UUID) store.LoginCode {
	hashSeq.Lock()
	hashSeq.n++
	n := hashSeq.n
	hashSeq.Unlock()
	return store.LoginCode{
		Hash:      sha256.Sum256(fmt.Appendf(nil, "code-%d-%s", n, uuid.NewV7())),
		Purpose:   purpose,
		Email:     "zażółć@example.com",
		UserID:    userID,
		Next:      "/consent?request=42",
		ExpiresAt: t0.Add(15 * time.Minute),
	}
}

func add(t *testing.T, b Backend, c store.LoginCode) {
	t.Helper()
	if err := addIn(t, b, c); err != nil {
		t.Fatalf("add code: %v", err)
	}
}

func addIn(t *testing.T, b Backend, c store.LoginCode) error {
	return b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error { return b.LoginCodes.Add(t.Context(), tx, c) })
}

func consume(t *testing.T, b Backend, c store.LoginCode, at time.Time) (store.LoginCode, error) {
	t.Helper()
	var got store.LoginCode
	err := b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error {
		var err error
		got, err = b.LoginCodes.Consume(t.Context(), tx, c.Hash, c.Purpose, at)
		return err
	})
	return got, err
}

func assertCode(t *testing.T, got, want store.LoginCode) {
	t.Helper()
	if got.Hash != want.Hash || got.Purpose != want.Purpose || got.Email != want.Email ||
		got.UserID != want.UserID || got.Next != want.Next || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("code = %+v\nwant   %+v", got, want)
	}
}

func assertUnusable(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, store.ErrCodeUnusable) {
		t.Errorf("%s: err = %v, want store.ErrCodeUnusable", what, err)
	}
}

func testConsumeOnce(t *testing.T, b Backend) {
	c := code(store.PurposeEmailChange, uuid.NewV7())
	add(t, b, c)

	for i := range 2 {
		found, err := b.LoginCodes.Find(t.Context(), c.Hash, c.Purpose, t0)
		if err != nil {
			t.Fatalf("find %d: %v", i+1, err)
		}
		assertCode(t, found, c)
	}
	got, err := consume(t, b, c, t0)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	assertCode(t, got, c)

	_, err = consume(t, b, c, t0)
	assertUnusable(t, "consume again", err)
	_, err = b.LoginCodes.Find(t.Context(), c.Hash, c.Purpose, t0)
	assertUnusable(t, "find once used", err)
}

func testCodeWithoutUser(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	c.Next = ""
	add(t, b, c)
	got, err := consume(t, b, c, t0)
	if err != nil {
		t.Fatal(err)
	}
	assertCode(t, got, c)
}

func testExpiredCode(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	add(t, b, c)

	_, err := b.LoginCodes.Find(t.Context(), c.Hash, c.Purpose, c.ExpiresAt)
	assertUnusable(t, "find at expiry", err)
	_, err = consume(t, b, c, c.ExpiresAt)
	assertUnusable(t, "consume at expiry", err)
	if _, err := consume(t, b, c, c.ExpiresAt.Add(-time.Millisecond)); err != nil {
		t.Errorf("consume a millisecond before expiry: %v", err)
	}
}

func testWrongPurpose(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	add(t, b, c)

	_, err := b.LoginCodes.Find(t.Context(), c.Hash, store.PurposeEmailChange, t0)
	assertUnusable(t, "find", err)
	wrong := c
	wrong.Purpose = store.PurposeEmailChange
	_, err = consume(t, b, wrong, t0)
	assertUnusable(t, "consume", err)
	if _, err := consume(t, b, c, t0); err != nil {
		t.Errorf("the code stopped working for its own purpose: %v", err)
	}
}

func testUnknownCode(t *testing.T, b Backend) {
	add(t, b, code(store.PurposeLogin, uuid.Nil()))
	unknown := code(store.PurposeLogin, uuid.Nil())
	_, err := b.LoginCodes.Find(t.Context(), unknown.Hash, unknown.Purpose, t0)
	assertUnusable(t, "find", err)
	_, err = consume(t, b, unknown, t0)
	assertUnusable(t, "consume", err)
}

func testConsumeRollback(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	add(t, b, c)

	failure := errors.New("registration failed")
	err := b.UnitOfWork.Do(t.Context(), func(tx uow.Tx) error {
		if _, err := b.LoginCodes.Consume(t.Context(), tx, c.Hash, c.Purpose, t0); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Do = %v, want the function's error", err)
	}
	if _, err := consume(t, b, c, t0); err != nil {
		t.Errorf("a rolled-back consume used the code up: %v", err)
	}
}

func testConcurrentConsumes(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	add(t, b, c)

	const redeemers = 8
	errs := make([]error, redeemers)
	var wg sync.WaitGroup
	for i := range redeemers {
		wg.Go(func() { _, errs[i] = consume(t, b, c, t0) })
	}
	wg.Wait()

	won := 0
	for i, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, store.ErrCodeUnusable):
			t.Errorf("redeemer %d: err = %v, want nil or store.ErrCodeUnusable", i, err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of %d concurrent consumes of one code succeeded, want exactly 1", won, redeemers)
	}
}

func testRepeatedHash(t *testing.T, b Backend) {
	c := code(store.PurposeLogin, uuid.Nil())
	add(t, b, c)
	again := c
	again.Email = "eve@example.com"
	if err := addIn(t, b, again); err == nil {
		t.Fatal("added a second code with the same hash")
	}
	found, err := b.LoginCodes.Find(t.Context(), c.Hash, c.Purpose, t0)
	if err != nil {
		t.Fatal(err)
	}
	assertCode(t, found, c)
}

func testDeleteExpired(t *testing.T, b Backend) {
	expired, live := code(store.PurposeLogin, uuid.Nil()), code(store.PurposeLogin, uuid.Nil())
	expired.ExpiresAt = t0.Add(time.Minute)
	live.ExpiresAt = t0.Add(time.Hour)
	add(t, b, expired)
	add(t, b, live)

	n, err := b.LoginCodes.DeleteExpired(t.Context(), expired.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("deleted %d codes, want 1", n)
	}
	_, err = b.LoginCodes.Find(t.Context(), expired.Hash, expired.Purpose, t0)
	assertUnusable(t, "find a deleted code", err)
	if _, err := b.LoginCodes.Find(t.Context(), live.Hash, live.Purpose, t0); err != nil {
		t.Errorf("a live code was deleted: %v", err)
	}
	if err := addIn(t, b, expired); err != nil {
		t.Errorf("the deleted code's hash is still taken: %v", err)
	}
}
