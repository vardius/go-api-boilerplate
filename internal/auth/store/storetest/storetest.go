// Package storetest is the contract every store.Store passes. A backend's
// test calls Run with a function returning a new, empty store.
package storetest

import (
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// t0 is the suite's clock. Times are whole milliseconds, which every
// backend keeps exactly.
var t0 = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

// concurrency is how many callers race for one single-use thing.
const concurrency = 8

// Run runs the contract against stores made by newStore. When the store is
// also a store.Purger, Purge is tested too.
func Run(t *testing.T, newStore func(t *testing.T) store.Store) {
	tests := []struct {
		name string
		fn   func(t *testing.T, s store.Store)
	}{
		{"ClientRoundTrip", testClientRoundTrip},
		{"ClientsListsTheOwnersOldestFirst", testClientsList},
		{"DeleteClientCascades", testDeleteClient},
		{"GrantRoundTrip", testGrantRoundTrip},
		{"ExpiredGrantsAreGone", testExpiredGrants},
		{"GrantsListsMostRecentlyUsedFirst", testGrantsList},
		{"RevokeGrantRemovesItsTokens", testRevokeGrant},
		{"RotateRefreshToken", testRotate},
		{"ConcurrentRotationsHaveOneWinner", testConcurrentRotate},
		{"AuthorizationRequestRoundTrip", testRequestRoundTrip},
		{"TakeAuthorizationRequestOnce", testTakeRequest},
		{"ConcurrentTakesHaveOneWinner", testConcurrentTake},
		{"AuthorizationCodeRoundTrip", testCodeRoundTrip},
		{"SpendAuthorizationCode", testSpendCode},
		{"RedeemAuthorizationCode", testRedeemCode},
		{"ConcurrentRedemptionsHaveOneWinner", testConcurrentRedeem},
		{"Purge", testPurge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.fn(t, newStore(t)) })
	}
}

func token() string { return rand.Text() }

func hash() store.Hash { return store.HashToken(token()) }

func newClient(owner uuid.UUID, created time.Time) store.Client {
	return store.Client{
		ID:           uuid.New().String(),
		OwnerID:      owner,
		Name:         "Café ☕ «app»",
		RedirectURIs: []string{"https://app.example.com/callback", "http://localhost:9999/cb?x=1"},
		Scopes:       []string{"user:read", "clients:read"},
		SecretHash:   "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		CreatedAt:    created,
	}
}

// newGrant returns a grant used at lastUsed whose first token lives a day.
func newGrant(user uuid.UUID, client string, lastUsed time.Time) (store.Grant, store.RefreshToken) {
	g := store.Grant{
		ID:         uuid.NewV7(),
		UserID:     user,
		ClientID:   client,
		Scopes:     []string{"user:read", "user:write", "clients:read", "clients:write"},
		UserAgent:  "Mozilla/5.0 (Macintosh) Safari/605.1.15 ünïcode",
		CreatedAt:  lastUsed.Add(-time.Hour),
		LastUsedAt: lastUsed,
		ExpiresAt:  lastUsed.Add(24 * time.Hour),
	}
	return g, store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: g.ExpiresAt}
}

func newRequest(user uuid.UUID, client string) store.AuthorizationRequest {
	return store.AuthorizationRequest{
		ID:            uuid.New(),
		UserID:        user,
		ClientID:      client,
		RedirectURI:   "https://app.example.com/callback",
		Scopes:        []string{"user:read"},
		State:         "state with spaces & ünïcode",
		CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		ExpiresAt:     t0.Add(10 * time.Minute),
	}
}

func newCode(user uuid.UUID, client string) store.AuthorizationCode {
	return store.AuthorizationCode{
		Hash:          hash(),
		ClientID:      client,
		UserID:        user,
		RedirectURI:   "https://app.example.com/callback",
		Scopes:        []string{"user:read", "user:write"},
		CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		ExpiresAt:     t0.Add(time.Minute),
		GrantID:       uuid.Nil(),
	}
}

// utc gives every time.Time field of the struct v points to the UTC
// location, so values read back compare with reflect.DeepEqual whatever
// zone a driver hands back.
func utc(v any) {
	rv := reflect.ValueOf(v).Elem()
	for _, field := range rv.Fields() {
		if f, ok := field.Interface().(time.Time); ok {
			field.Set(reflect.ValueOf(f.UTC()))
		}
	}
}

func equal[T any](t *testing.T, what string, got, want T) {
	t.Helper()
	utc(&got)
	utc(&want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %+v\nwant %+v", what, got, want)
	}
}

func is(t *testing.T, what string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("%s: err = %v, want %v", what, err, want)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func testClientRoundTrip(t *testing.T, s store.Store) {
	c := newClient(uuid.NewV7(), t0)
	must(t, s.CreateClient(t.Context(), c))
	got, err := s.Client(t.Context(), c.ID)
	must(t, err)
	equal(t, "client", got, c)

	_, err = s.Client(t.Context(), uuid.New().String())
	is(t, "unknown client", err, apperr.ErrNotFound)
	dup := newClient(uuid.NewV7(), t0)
	dup.ID = c.ID
	is(t, "same ID again", s.CreateClient(t.Context(), dup), apperr.ErrConflict)
}

func testClientsList(t *testing.T, s store.Store) {
	owner, other := uuid.NewV7(), uuid.NewV7()
	first, second, third := newClient(owner, t0), newClient(owner, t0.Add(time.Second)), newClient(owner, t0.Add(2*time.Second))
	// A twin shares second's time, so only the ID decides which comes first.
	twin := newClient(owner, second.CreatedAt)
	if twin.ID < second.ID {
		twin, second = second, twin
	}
	for _, c := range []store.Client{third, newClient(other, t0), first, twin, second} {
		must(t, s.CreateClient(t.Context(), c))
	}
	ids := func(cs []store.Client) []string {
		out := []string{}
		for _, c := range cs {
			out = append(out, c.ID)
		}
		return out
	}
	tests := []struct {
		name  string
		after store.ClientKey
		limit int
		want  []string
	}{
		{"all", store.ClientKey{}, 10, []string{first.ID, second.ID, twin.ID, third.ID}},
		{"first page", store.ClientKey{}, 2, []string{first.ID, second.ID}},
		{"after a tie", second.Key(), 2, []string{twin.ID, third.ID}},
		{"after the last", third.Key(), 2, []string{}},
		{"after a place no client holds", store.ClientKey{CreatedAt: t0.Add(1500 * time.Millisecond), ID: first.ID}, 10, []string{third.ID}},
	}
	for _, tt := range tests {
		got, err := s.Clients(t.Context(), owner, tt.after, tt.limit)
		must(t, err)
		if !reflect.DeepEqual(ids(got), tt.want) {
			t.Errorf("%s: Clients = %v, want %v", tt.name, ids(got), tt.want)
		}
	}
	got, err := s.Clients(t.Context(), owner, store.ClientKey{}, 1)
	must(t, err)
	if len(got) == 1 {
		equal(t, "listed client", got[0], first)
	}

	// A page boundary is a place, not a position: adding a client before
	// it or deleting the one at it doesn't move the next page.
	must(t, s.CreateClient(t.Context(), newClient(owner, t0.Add(-time.Hour))))
	must(t, s.DeleteClient(t.Context(), owner, second.ID))
	next, err := s.Clients(t.Context(), owner, second.Key(), 10)
	must(t, err)
	if want := []string{twin.ID, third.ID}; !reflect.DeepEqual(ids(next), want) {
		t.Errorf("after a deleted boundary: Clients = %v, want %v", ids(next), want)
	}

	none, err := s.Clients(t.Context(), uuid.NewV7(), store.ClientKey{}, 10)
	must(t, err)
	if len(none) != 0 {
		t.Errorf("a user with no clients has %d", len(none))
	}
}

func testDeleteClient(t *testing.T, s store.Store) {
	owner, user := uuid.NewV7(), uuid.NewV7()
	c, kept := newClient(owner, t0), newClient(owner, t0)
	must(t, s.CreateClient(t.Context(), c))
	must(t, s.CreateClient(t.Context(), kept))
	g, rt := newGrant(user, c.ID, t0)
	must(t, s.CreateGrant(t.Context(), g, rt))
	keptGrant, keptToken := newGrant(user, kept.ID, t0)
	must(t, s.CreateGrant(t.Context(), keptGrant, keptToken))
	req := newRequest(user, c.ID)
	must(t, s.CreateAuthorizationRequest(t.Context(), req))
	code := newCode(user, c.ID)
	must(t, s.CreateAuthorizationCode(t.Context(), code))

	is(t, "another owner deletes", s.DeleteClient(t.Context(), uuid.NewV7(), c.ID), apperr.ErrNotFound)
	_, err := s.Client(t.Context(), c.ID)
	must(t, err)

	must(t, s.DeleteClient(t.Context(), owner, c.ID))
	_, err = s.Client(t.Context(), c.ID)
	is(t, "deleted client", err, apperr.ErrNotFound)
	_, err = s.Grant(t.Context(), g.ID, t0)
	is(t, "its grant", err, apperr.ErrNotFound)
	_, _, err = s.RefreshTokenGrant(t.Context(), rt.Hash, t0)
	is(t, "its refresh token", err, apperr.ErrNotFound)
	_, err = s.AuthorizationRequest(t.Context(), req.ID, user, t0)
	is(t, "its authorization request", err, apperr.ErrNotFound)
	_, err = s.AuthorizationCode(t.Context(), code.Hash)
	is(t, "its authorization code", err, apperr.ErrNotFound)
	is(t, "deleting again", s.DeleteClient(t.Context(), owner, c.ID), apperr.ErrNotFound)

	_, err = s.Client(t.Context(), kept.ID)
	must(t, err)
	_, _, err = s.RefreshTokenGrant(t.Context(), keptToken.Hash, t0)
	must(t, err)
}

func testGrantRoundTrip(t *testing.T, s store.Store) {
	g, rt := newGrant(uuid.NewV7(), "web", t0)
	must(t, s.CreateGrant(t.Context(), g, rt))
	got, err := s.Grant(t.Context(), g.ID, t0)
	must(t, err)
	equal(t, "grant", got, g)

	byToken, rotated, err := s.RefreshTokenGrant(t.Context(), rt.Hash, t0)
	must(t, err)
	equal(t, "grant by refresh token", byToken, g)
	if rotated {
		t.Error("a new token is rotated")
	}

	_, err = s.Grant(t.Context(), uuid.NewV7(), t0)
	is(t, "unknown grant", err, apperr.ErrNotFound)
	_, _, err = s.RefreshTokenGrant(t.Context(), hash(), t0)
	is(t, "unknown refresh token", err, apperr.ErrNotFound)

	again, _ := newGrant(uuid.NewV7(), "web", t0)
	again.ID = g.ID
	is(t, "same grant ID again", s.CreateGrant(t.Context(), again, store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: again.ExpiresAt}), apperr.ErrConflict)
	other, _ := newGrant(uuid.NewV7(), "web", t0)
	is(t, "same refresh token again", s.CreateGrant(t.Context(), other, store.RefreshToken{Hash: rt.Hash, GrantID: other.ID, ExpiresAt: other.ExpiresAt}), apperr.ErrConflict)
	_, err = s.Grant(t.Context(), other.ID, t0)
	is(t, "a grant whose token clashed", err, apperr.ErrNotFound)
}

func testExpiredGrants(t *testing.T, s store.Store) {
	g, rt := newGrant(uuid.NewV7(), "web", t0)
	must(t, s.CreateGrant(t.Context(), g, rt))
	before := g.ExpiresAt.Add(-time.Millisecond)
	if _, err := s.Grant(t.Context(), g.ID, before); err != nil {
		t.Errorf("a grant a millisecond before it expires: %v", err)
	}
	_, err := s.Grant(t.Context(), g.ID, g.ExpiresAt)
	is(t, "a grant at its expiry", err, apperr.ErrNotFound)
	_, _, err = s.RefreshTokenGrant(t.Context(), rt.Hash, rt.ExpiresAt)
	is(t, "a refresh token at its expiry", err, apperr.ErrNotFound)
	err = s.RotateRefreshToken(t.Context(), rt.Hash, store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: rt.ExpiresAt.Add(time.Hour)}, rt.ExpiresAt)
	is(t, "rotating an expired token", err, apperr.ErrNotFound)
}

func testGrantsList(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	older, _ := newGrant(user, "web", t0.Add(time.Second))
	newest, _ := newGrant(user, "web", t0.Add(3*time.Second))
	middle, _ := newGrant(user, "web", t0.Add(2*time.Second))
	expired, _ := newGrant(user, "web", t0.Add(4*time.Second))
	expired.ExpiresAt = t0.Add(5 * time.Second)
	otherClient, _ := newGrant(user, "client-1", t0.Add(5*time.Second))
	otherUser, _ := newGrant(uuid.NewV7(), "web", t0.Add(5*time.Second))
	for _, g := range []store.Grant{older, newest, middle, expired, otherClient, otherUser} {
		must(t, s.CreateGrant(t.Context(), g, store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: g.ExpiresAt}))
	}
	now := t0.Add(6 * time.Second)
	got, err := s.Grants(t.Context(), user, "web", now, 10)
	must(t, err)
	want := []uuid.UUID{newest.ID, middle.ID, older.ID}
	var ids []uuid.UUID
	for _, g := range got {
		ids = append(ids, g.ID)
	}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("Grants = %v, want %v", ids, want)
	}
	if len(got) == 3 {
		equal(t, "listed grant", got[0], newest)
	}
	limited, err := s.Grants(t.Context(), user, "web", now, 2)
	must(t, err)
	if len(limited) != 2 || limited[0].ID != newest.ID {
		t.Errorf("Grants with limit 2 = %d grants", len(limited))
	}
}

func testRevokeGrant(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	g, first := newGrant(user, "web", t0)
	must(t, s.CreateGrant(t.Context(), g, first))
	second := store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: g.ExpiresAt.Add(time.Hour)}
	must(t, s.RotateRefreshToken(t.Context(), first.Hash, second, t0))
	kept, keptToken := newGrant(user, "web", t0)
	must(t, s.CreateGrant(t.Context(), kept, keptToken))

	must(t, s.RevokeGrant(t.Context(), g.ID))
	_, err := s.Grant(t.Context(), g.ID, t0)
	is(t, "revoked grant", err, apperr.ErrNotFound)
	for name, h := range map[string]store.Hash{"rotated": first.Hash, "current": second.Hash} {
		_, _, err := s.RefreshTokenGrant(t.Context(), h, t0)
		is(t, "the "+name+" token of a revoked grant", err, apperr.ErrNotFound)
	}
	is(t, "rotating a revoked grant's token", s.RotateRefreshToken(t.Context(), second.Hash, store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: g.ExpiresAt}, t0), apperr.ErrNotFound)
	must(t, s.RevokeGrant(t.Context(), g.ID))
	must(t, s.RevokeGrant(t.Context(), uuid.NewV7()))
	_, _, err = s.RefreshTokenGrant(t.Context(), keptToken.Hash, t0)
	must(t, err)
}

func testRotate(t *testing.T, s store.Store) {
	g, first := newGrant(uuid.NewV7(), "client-1", t0)
	must(t, s.CreateGrant(t.Context(), g, first))
	now := t0.Add(time.Minute)
	second := store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: now.Add(30 * 24 * time.Hour)}
	must(t, s.RotateRefreshToken(t.Context(), first.Hash, second, now))

	got, rotated, err := s.RefreshTokenGrant(t.Context(), first.Hash, now)
	must(t, err)
	if !rotated || got.ID != g.ID {
		t.Errorf("the first token: grant %s, rotated %t; want grant %s, rotated", got.ID, rotated, g.ID)
	}
	got, rotated, err = s.RefreshTokenGrant(t.Context(), second.Hash, now)
	must(t, err)
	want := g
	want.LastUsedAt, want.ExpiresAt = now, second.ExpiresAt
	equal(t, "grant after rotation", got, want)
	if rotated {
		t.Error("the new token is rotated")
	}

	third := store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: now.Add(time.Hour)}
	is(t, "rotating the rotated token", s.RotateRefreshToken(t.Context(), first.Hash, third, now), store.ErrUsed)
	_, _, err = s.RefreshTokenGrant(t.Context(), third.Hash, now)
	is(t, "the token a refused rotation brought", err, apperr.ErrNotFound)
	is(t, "rotating an unknown token", s.RotateRefreshToken(t.Context(), hash(), third, now), apperr.ErrNotFound)
	is(t, "rotating to a token that exists", s.RotateRefreshToken(t.Context(), second.Hash, store.RefreshToken{Hash: first.Hash, GrantID: g.ID, ExpiresAt: now.Add(time.Hour)}, now), apperr.ErrConflict)
	_, rotated, err = s.RefreshTokenGrant(t.Context(), second.Hash, now)
	must(t, err)
	if rotated {
		t.Error("a rotation that failed left its token rotated")
	}
}

// race runs fn concurrency times at once and returns the errors.
func race(fn func(i int) error) []error {
	errs := make([]error, concurrency)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range concurrency {
		wg.Go(func() {
			<-start
			errs[i] = fn(i)
		})
	}
	close(start)
	wg.Wait()
	return errs
}

// oneWinner checks that exactly one error is nil and that every other is
// want, and returns the winner.
func oneWinner(t *testing.T, errs []error, want error) int {
	t.Helper()
	winner := -1
	for i, err := range errs {
		switch {
		case err == nil && winner >= 0:
			t.Errorf("callers %d and %d both won", winner, i)
		case err == nil:
			winner = i
		case !errors.Is(err, want):
			t.Errorf("caller %d: err = %v, want %v", i, err, want)
		}
	}
	if winner < 0 {
		t.Fatal("nobody won")
	}
	return winner
}

func testConcurrentRotate(t *testing.T, s store.Store) {
	g, first := newGrant(uuid.NewV7(), "web", t0)
	must(t, s.CreateGrant(t.Context(), g, first))
	next := make([]store.RefreshToken, concurrency)
	for i := range next {
		next[i] = store.RefreshToken{Hash: hash(), GrantID: g.ID, ExpiresAt: g.ExpiresAt.Add(time.Hour)}
	}
	errs := race(func(i int) error { return s.RotateRefreshToken(t.Context(), first.Hash, next[i], t0) })
	winner := oneWinner(t, errs, store.ErrUsed)
	for i, rt := range next {
		_, _, err := s.RefreshTokenGrant(t.Context(), rt.Hash, t0)
		if i == winner {
			must(t, err)
			continue
		}
		is(t, fmt.Sprintf("the token of losing rotation %d", i), err, apperr.ErrNotFound)
	}
}

func testRequestRoundTrip(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	r := newRequest(user, "client-1")
	must(t, s.CreateAuthorizationRequest(t.Context(), r))
	got, err := s.AuthorizationRequest(t.Context(), r.ID, user, t0)
	must(t, err)
	equal(t, "authorization request", got, r)

	_, err = s.AuthorizationRequest(t.Context(), r.ID, uuid.NewV7(), t0)
	is(t, "another user's request", err, apperr.ErrNotFound)
	_, err = s.AuthorizationRequest(t.Context(), r.ID, user, r.ExpiresAt)
	is(t, "an expired request", err, apperr.ErrNotFound)
	_, err = s.AuthorizationRequest(t.Context(), uuid.New(), user, t0)
	is(t, "an unknown request", err, apperr.ErrNotFound)
	is(t, "same ID again", s.CreateAuthorizationRequest(t.Context(), r), apperr.ErrConflict)
}

func testTakeRequest(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	r, expired := newRequest(user, "client-1"), newRequest(user, "client-1")
	expired.ExpiresAt = t0
	must(t, s.CreateAuthorizationRequest(t.Context(), r))
	must(t, s.CreateAuthorizationRequest(t.Context(), expired))

	_, err := s.TakeAuthorizationRequest(t.Context(), r.ID, uuid.NewV7(), t0)
	is(t, "another user takes it", err, apperr.ErrNotFound)
	got, err := s.TakeAuthorizationRequest(t.Context(), r.ID, user, t0)
	must(t, err)
	equal(t, "taken request", got, r)
	_, err = s.TakeAuthorizationRequest(t.Context(), r.ID, user, t0)
	is(t, "taking it again", err, apperr.ErrNotFound)
	_, err = s.AuthorizationRequest(t.Context(), r.ID, user, t0)
	is(t, "reading a taken request", err, apperr.ErrNotFound)
	_, err = s.TakeAuthorizationRequest(t.Context(), expired.ID, user, t0)
	is(t, "taking an expired request", err, apperr.ErrNotFound)
}

func testConcurrentTake(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	r := newRequest(user, "client-1")
	must(t, s.CreateAuthorizationRequest(t.Context(), r))
	errs := race(func(int) error {
		_, err := s.TakeAuthorizationRequest(t.Context(), r.ID, user, t0)
		return err
	})
	oneWinner(t, errs, apperr.ErrNotFound)
}

func testCodeRoundTrip(t *testing.T, s store.Store) {
	c := newCode(uuid.NewV7(), "client-1")
	must(t, s.CreateAuthorizationCode(t.Context(), c))
	got, err := s.AuthorizationCode(t.Context(), c.Hash)
	must(t, err)
	equal(t, "authorization code", got, c)

	_, err = s.AuthorizationCode(t.Context(), hash())
	is(t, "an unknown code", err, apperr.ErrNotFound)
	is(t, "same code again", s.CreateAuthorizationCode(t.Context(), c), apperr.ErrConflict)
}

func testSpendCode(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	c := newCode(user, "client-1")
	must(t, s.CreateAuthorizationCode(t.Context(), c))
	must(t, s.SpendAuthorizationCode(t.Context(), c.Hash))
	got, err := s.AuthorizationCode(t.Context(), c.Hash)
	must(t, err)
	want := c
	want.Used = true
	equal(t, "spent code", got, want)

	is(t, "spending it again", s.SpendAuthorizationCode(t.Context(), c.Hash), store.ErrUsed)
	g, rt := newGrant(user, "client-1", t0)
	is(t, "redeeming a spent code", s.RedeemAuthorizationCode(t.Context(), c.Hash, g, rt), store.ErrUsed)
	_, err = s.Grant(t.Context(), g.ID, t0)
	is(t, "the grant of a refused redemption", err, apperr.ErrNotFound)
	is(t, "spending an unknown code", s.SpendAuthorizationCode(t.Context(), hash()), apperr.ErrNotFound)
}

func testRedeemCode(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	c := newCode(user, "client-1")
	must(t, s.CreateAuthorizationCode(t.Context(), c))
	g, rt := newGrant(user, "client-1", t0)
	must(t, s.RedeemAuthorizationCode(t.Context(), c.Hash, g, rt))

	got, err := s.AuthorizationCode(t.Context(), c.Hash)
	must(t, err)
	want := c
	want.Used, want.GrantID = true, g.ID
	equal(t, "redeemed code", got, want)
	grant, err := s.Grant(t.Context(), g.ID, t0)
	must(t, err)
	equal(t, "the grant it created", grant, g)
	if _, _, err := s.RefreshTokenGrant(t.Context(), rt.Hash, t0); err != nil {
		t.Errorf("the grant's refresh token: %v", err)
	}

	again, againToken := newGrant(user, "client-1", t0)
	is(t, "redeeming it again", s.RedeemAuthorizationCode(t.Context(), c.Hash, again, againToken), store.ErrUsed)
	_, err = s.Grant(t.Context(), again.ID, t0)
	is(t, "the grant of the second redemption", err, apperr.ErrNotFound)
	is(t, "spending a redeemed code", s.SpendAuthorizationCode(t.Context(), c.Hash), store.ErrUsed)
	got, err = s.AuthorizationCode(t.Context(), c.Hash)
	must(t, err)
	if got.GrantID != g.ID {
		t.Errorf("after refused uses the code names grant %s, want %s", got.GrantID, g.ID)
	}

	lost, lostToken := newGrant(user, "client-1", t0)
	is(t, "redeeming an unknown code", s.RedeemAuthorizationCode(t.Context(), hash(), lost, lostToken), apperr.ErrNotFound)
	_, err = s.Grant(t.Context(), lost.ID, t0)
	is(t, "the grant of an unknown code", err, apperr.ErrNotFound)
}

func testConcurrentRedeem(t *testing.T, s store.Store) {
	user := uuid.NewV7()
	c := newCode(user, "client-1")
	must(t, s.CreateAuthorizationCode(t.Context(), c))
	grants := make([]store.Grant, concurrency)
	tokens := make([]store.RefreshToken, concurrency)
	for i := range grants {
		grants[i], tokens[i] = newGrant(user, "client-1", t0)
	}
	errs := race(func(i int) error {
		// Half the callers spend the code, as a failed exchange does.
		if i%2 == 1 {
			return s.SpendAuthorizationCode(t.Context(), c.Hash)
		}
		return s.RedeemAuthorizationCode(t.Context(), c.Hash, grants[i], tokens[i])
	})
	winner := oneWinner(t, errs, store.ErrUsed)

	got, err := s.AuthorizationCode(t.Context(), c.Hash)
	must(t, err)
	wantGrant := uuid.Nil()
	if winner%2 == 0 {
		wantGrant = grants[winner].ID
	}
	if !got.Used || got.GrantID != wantGrant {
		t.Errorf("code: used %t, grant %s; want used, grant %s", got.Used, got.GrantID, wantGrant)
	}
	for i, g := range grants {
		_, err := s.Grant(t.Context(), g.ID, t0)
		if i == winner && i%2 == 0 {
			must(t, err)
			continue
		}
		is(t, fmt.Sprintf("the grant of caller %d", i), err, apperr.ErrNotFound)
	}
}

func testPurge(t *testing.T, s store.Store) {
	p, ok := s.(store.Purger)
	if !ok {
		t.Skip("the store purges on its own")
	}
	now := t0.Add(48 * time.Hour)
	user := uuid.NewV7()

	expiredGrant, expiredToken := newGrant(user, "web", t0)
	must(t, s.CreateGrant(t.Context(), expiredGrant, expiredToken))
	live, oldToken := newGrant(user, "web", t0)
	must(t, s.CreateGrant(t.Context(), live, oldToken))
	current := store.RefreshToken{Hash: hash(), GrantID: live.ID, ExpiresAt: now.Add(time.Hour)}
	must(t, s.RotateRefreshToken(t.Context(), oldToken.Hash, current, t0.Add(time.Hour)))

	expiredRequest, liveRequest := newRequest(user, "client-1"), newRequest(user, "client-1")
	liveRequest.ExpiresAt = now.Add(time.Minute)
	must(t, s.CreateAuthorizationRequest(t.Context(), expiredRequest))
	must(t, s.CreateAuthorizationRequest(t.Context(), liveRequest))

	oldCode, retainedCode := newCode(user, "client-1"), newCode(user, "client-1")
	oldCode.ExpiresAt = now.Add(-store.UsedCodeRetention)
	retainedCode.ExpiresAt = now.Add(-store.UsedCodeRetention + time.Millisecond)
	must(t, s.CreateAuthorizationCode(t.Context(), oldCode))
	must(t, s.CreateAuthorizationCode(t.Context(), retainedCode))

	must(t, p.Purge(t.Context(), now))

	// Read at t0, before anything expired, so that only the purge can make
	// them missing.
	_, err := s.Grant(t.Context(), expiredGrant.ID, t0)
	is(t, "an expired grant", err, apperr.ErrNotFound)
	_, _, err = s.RefreshTokenGrant(t.Context(), expiredToken.Hash, t0)
	is(t, "an expired grant's token", err, apperr.ErrNotFound)
	_, _, err = s.RefreshTokenGrant(t.Context(), oldToken.Hash, t0)
	is(t, "a live grant's expired rotated token", err, apperr.ErrNotFound)
	if _, _, err := s.RefreshTokenGrant(t.Context(), current.Hash, now); err != nil {
		t.Errorf("a live grant's current token: %v", err)
	}
	_, err = s.AuthorizationRequest(t.Context(), expiredRequest.ID, user, t0)
	is(t, "an expired request", err, apperr.ErrNotFound)
	if _, err := s.AuthorizationRequest(t.Context(), liveRequest.ID, user, now); err != nil {
		t.Errorf("a live request: %v", err)
	}
	_, err = s.AuthorizationCode(t.Context(), oldCode.Hash)
	is(t, "a code past its retention", err, apperr.ErrNotFound)
	if _, err := s.AuthorizationCode(t.Context(), retainedCode.Hash); err != nil {
		t.Errorf("an expired code still retained: %v", err)
	}
}
