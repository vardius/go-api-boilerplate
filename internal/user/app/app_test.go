package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
	"github.com/vardius/go-api-boilerplate/internal/user/store/memory"
)

var t0 = time.Date(2026, 10, 24, 9, 30, 15, 123_456_789, time.UTC)

type rig struct {
	svc    *app.Service
	events *memdb.EventStore
	outbox *memdb.Outbox
	users  *users
	codes  *memory.LoginCodes

	mu  sync.Mutex
	now time.Time
}

// users is the memory projection, with reads a test can make stale or
// fail.
type users struct {
	*memory.Users

	mu sync.Mutex
	// staleEmail is how many ByEmail calls answer not found, as a read
	// that raced a registration would.
	staleEmail int
	// staleSubject is the same for ByGoogleSubject.
	staleSubject int
	// fail, when set, is what every read returns.
	fail error
}

func (u *users) err() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.fail
}

func (u *users) ByID(ctx context.Context, id uuid.UUID) (store.User, error) {
	if err := u.err(); err != nil {
		return store.User{}, err
	}
	return u.Users.ByID(ctx, id)
}

// stale takes one of the misses *n holds, and returns the error every read
// fails with.
func (u *users) stale(n *int) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.fail != nil {
		return u.fail
	}
	if *n > 0 {
		*n--
		return apperr.ErrNotFound
	}
	return nil
}

func (u *users) ByEmail(ctx context.Context, e domain.Email) (store.User, error) {
	if err := u.stale(&u.staleEmail); err != nil {
		return store.User{}, err
	}
	return u.Users.ByEmail(ctx, e)
}

func (u *users) ByGoogleSubject(ctx context.Context, sub string) (store.User, error) {
	if err := u.stale(&u.staleSubject); err != nil {
		return store.User{}, err
	}
	return u.Users.ByGoogleSubject(ctx, sub)
}

func (u *users) set(f func(u *users)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f(u)
}

func newRig(t *testing.T) *rig {
	t.Helper()
	db := memdb.New()
	r := &rig{
		events: memdb.NewEventStore(db),
		outbox: memdb.NewOutbox(db),
		users:  &users{Users: memory.NewUsers(db)},
		codes:  memory.NewLoginCodes(db),
		now:    t0,
	}
	web, err := url.Parse("https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	r.svc = app.New(app.Deps{
		UnitOfWork: db,
		Events:     r.events,
		Outbox:     r.outbox,
		Users:      r.users,
		LoginCodes: r.codes,
		WebURL:     web,
		Now:        r.clock,
	})
	return r
}

func (r *rig) clock() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.now
}

func (r *rig) advance(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = r.now.Add(d)
}

// mail is one message the use cases put in the outbox.
type sent struct {
	topic string
	link  mail.Link
	url   *url.URL
}

// mailed takes every message waiting in the outbox.
func (r *rig) mailed(t *testing.T) []sent {
	t.Helper()
	msgs, err := r.outbox.Claim(t.Context(), r.clock(), time.Minute, 100)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]sent, len(msgs))
	for i, m := range msgs {
		if err := json.Unmarshal(m.Payload, &out[i].link); err != nil {
			t.Fatal(err)
		}
		out[i].topic = m.Topic
		if out[i].url, err = url.Parse(out[i].link.URL); err != nil {
			t.Fatal(err)
		}
		if err := r.outbox.Ack(t.Context(), m.ID); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// lastMail takes the one message waiting in the outbox.
func (r *rig) lastMail(t *testing.T) sent {
	t.Helper()
	got := r.mailed(t)
	if len(got) != 1 {
		t.Fatalf("outbox holds %d messages, want 1", len(got))
	}
	return got[0]
}

func (r *rig) requestCode(t *testing.T, email string) string {
	t.Helper()
	if err := r.svc.RequestLoginLink(t.Context(), email, ""); err != nil {
		t.Fatal(err)
	}
	return r.lastMail(t).url.Query().Get("code")
}

func (r *rig) signUp(t *testing.T, email string) store.User {
	t.Helper()
	u, err := r.svc.RedeemLoginCode(asAuth(t.Context()), r.requestCode(t, email))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// asAuth is the auth service's own token.
func asAuth(ctx context.Context) context.Context {
	return authn.NewContext(ctx, authn.Identity{Subject: "auth", ClientID: "auth", Scopes: []string{app.ScopeRedeemLoginCodes}})
}

func asUser(ctx context.Context, id uuid.UUID, scopes ...string) context.Context {
	return authn.NewContext(ctx, authn.Identity{Subject: id.String(), UserID: id, ClientID: authn.FirstPartyClient, Scopes: scopes})
}

func asOwner(ctx context.Context, u store.User) context.Context {
	return asUser(ctx, u.ID, app.ScopeUserRead, app.ScopeUserWrite)
}

func asClient(ctx context.Context) context.Context {
	return authn.NewContext(ctx, authn.Identity{Subject: "app-1", ClientID: "app-1", Scopes: []string{app.ScopeUserRead, app.ScopeUserWrite}})
}

func assertKind(t *testing.T, what string, err, kind error) {
	t.Helper()
	if !errors.Is(err, kind) {
		t.Fatalf("%s: err = %v, want %v", what, err, kind)
	}
}

func TestRequestLoginLinkMailsALink(t *testing.T) {
	r := newRig(t)
	if err := r.svc.RequestLoginLink(t.Context(), "  Ada@Example.com ", "/consent?request=42"); err != nil {
		t.Fatal(err)
	}

	m := r.lastMail(t)
	if m.topic != mail.TopicLoginLink || m.link.To != "ada@example.com" {
		t.Errorf("mailed %s to %s", m.topic, m.link.To)
	}
	if got := m.url.Scheme + "://" + m.url.Host + m.url.Path; got != "https://app.example.com/sign-in/complete" {
		t.Errorf("link goes to %s", got)
	}
	if next := m.url.Query().Get("next"); next != "/consent?request=42" {
		t.Errorf("next = %q", next)
	}
	if code := m.url.Query().Get("code"); len(code) != 43 {
		t.Errorf("code %q is %d characters, want 43 (32 bytes, base64url)", code, len(code))
	}
	// 15 minutes from t0, at the millisecond precision stores keep.
	if want := time.Date(2026, 10, 24, 9, 45, 15, 123_000_000, time.UTC); !m.link.ExpiresAt.Equal(want) {
		t.Errorf("expires at %s, want %s", m.link.ExpiresAt, want)
	}

	// Asking creates nobody: the account appears when the code is redeemed.
	_, err := r.users.ByEmail(t.Context(), mustEmail("ada@example.com"))
	assertKind(t, "ByEmail before redeeming", err, apperr.ErrNotFound)
}

func TestRequestLoginLinkSucceedsWhetherOrNotTheAccountExists(t *testing.T) {
	r := newRig(t)
	r.signUp(t, "ada@example.com")
	for _, email := range []string{"ada@example.com", "nobody@example.com"} {
		if err := r.svc.RequestLoginLink(t.Context(), email, ""); err != nil {
			t.Errorf("RequestLoginLink(%s) = %v", email, err)
		}
	}
	if n := len(r.mailed(t)); n != 2 {
		t.Errorf("mailed %d links, want 2", n)
	}
}

func TestRequestLoginLinkDropsAnUnsafeNext(t *testing.T) {
	r := newRig(t)
	if err := r.svc.RequestLoginLink(t.Context(), "ada@example.com", "https://evil.example/phish"); err != nil {
		t.Fatal(err)
	}
	if q := r.lastMail(t).url.Query(); q.Has("next") {
		t.Errorf("link kept next=%q", q.Get("next"))
	}
}

func TestRequestLoginLinkRejectsABadAddress(t *testing.T) {
	r := newRig(t)
	err := r.svc.RequestLoginLink(t.Context(), "not an address", "")
	assertKind(t, "RequestLoginLink", err, apperr.ErrInvalid)
	if n := len(r.mailed(t)); n != 0 {
		t.Errorf("mailed %d links", n)
	}
}

func TestRedeemLoginCodeRegistersANewAddress(t *testing.T) {
	r := newRig(t)
	code := r.requestCode(t, "Ada@Example.com")

	u, err := r.svc.RedeemLoginCode(asAuth(obs.WithRequestID(t.Context(), "req-1")), code)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == uuid.Nil() || u.Email != "ada@example.com" || !u.CreatedAt.Equal(t0.Truncate(time.Millisecond)) {
		t.Errorf("redeemed %+v", u)
	}
	me, err := r.svc.Me(asOwner(t.Context(), u))
	if err != nil || me.ID != u.ID || me.Email != u.Email {
		t.Errorf("Me = %+v, %v", me, err)
	}

	recs, err := r.events.Load(t.Context(), u.ID)
	if err != nil || len(recs) != 1 {
		t.Fatalf("stream holds %d events, %v", len(recs), err)
	}
	if recs[0].Type != "user.registered.v1" || recs[0].Meta.CorrelationID != "req-1" || recs[0].Meta.ActorID != "auth" {
		t.Errorf("event = %s with %+v", recs[0].Type, recs[0].Meta)
	}

	_, err = r.svc.RedeemLoginCode(asAuth(t.Context()), code)
	assertKind(t, "redeeming again", err, apperr.ErrUnauthorized)
}

func TestRedeemLoginCodeFindsTheSameUserInAnyCase(t *testing.T) {
	r := newRig(t)
	first := r.signUp(t, "ada@example.com")
	again := r.signUp(t, "ADA@EXAMPLE.COM")
	if again.ID != first.ID {
		t.Errorf("second sign-in is user %s, want %s", again.ID, first.ID)
	}
}

func TestRedeemLoginCodeOfAnAddressRegisteredSince(t *testing.T) {
	r := newRig(t)
	early := r.requestCode(t, "ada@example.com")
	u := r.signUp(t, "ada@example.com")

	got, err := r.svc.RedeemLoginCode(asAuth(t.Context()), early)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Errorf("the earlier link signed in %s, want %s", got.ID, u.ID)
	}
}

func TestRedeemLoginCodeRefuses(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")
	code := r.requestCode(t, "ada@example.com")

	tests := []struct {
		name string
		ctx  context.Context
		code string
		want error
	}{
		{"anonymous", t.Context(), code, apperr.ErrUnauthorized},
		{"a user", asOwner(t.Context(), u), code, apperr.ErrForbidden},
		{"a client without the scope", asClient(t.Context()), code, apperr.ErrForbidden},
		{"an empty code", asAuth(t.Context()), "", apperr.ErrUnauthorized},
		{"a malformed code", asAuth(t.Context()), "not-a-code", apperr.ErrUnauthorized},
		{"an unknown code", asAuth(t.Context()), strings.Repeat("A", 43), apperr.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.svc.RedeemLoginCode(tt.ctx, tt.code)
			assertKind(t, "RedeemLoginCode", err, tt.want)
		})
	}
	if _, err := r.svc.RedeemLoginCode(asAuth(t.Context()), code); err != nil {
		t.Errorf("the refused attempts used the code up: %v", err)
	}
}

func TestRedeemLoginCodeExpires(t *testing.T) {
	r := newRig(t)
	code := r.requestCode(t, "ada@example.com")
	r.advance(app.LoginCodeTTL)
	_, err := r.svc.RedeemLoginCode(asAuth(t.Context()), code)
	assertKind(t, "redeeming at expiry", err, apperr.ErrUnauthorized)
}

func TestRedeemLoginCodeAfterTheAccountMovedAddress(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")
	stale := r.requestCode(t, "ada@example.com")
	changeEmail(t, r, u, "grace@example.com")

	_, err := r.svc.RedeemLoginCode(asAuth(t.Context()), stale)
	assertKind(t, "redeeming a link to the old address", err, apperr.ErrUnauthorized)
}

func TestRedeemLoginCodeConcurrentlyOneWins(t *testing.T) {
	r := newRig(t)
	code := r.requestCode(t, "ada@example.com")

	const redeemers = 8
	errs := make([]error, redeemers)
	var wg sync.WaitGroup
	for i := range redeemers {
		wg.Go(func() { _, errs[i] = r.svc.RedeemLoginCode(asAuth(t.Context()), code) })
	}
	wg.Wait()
	won := 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, apperr.ErrUnauthorized):
			t.Errorf("err = %v, want nil or apperr.ErrUnauthorized", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of %d concurrent redeems succeeded, want 1", won, redeemers)
	}
}

// Two links to a new address, opened at once: both sign in, to one account.
func TestRedeemLoginCodesForANewAddressConcurrently(t *testing.T) {
	r := newRig(t)
	const links = 8
	codes := make([]string, links)
	for i := range codes {
		codes[i] = r.requestCode(t, "ada@example.com")
	}

	users := make([]store.User, links)
	errs := make([]error, links)
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Go(func() { users[i], errs[i] = r.svc.RedeemLoginCode(asAuth(t.Context()), code) })
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("redeem %d: %v", i, err)
		}
		if users[i].ID != users[0].ID {
			t.Errorf("redeem %d signed in %s, want %s", i, users[i].ID, users[0].ID)
		}
	}
}

// The same race, made certain: the second redeem reads the address as free,
// loses the registration, and retries into the account the first made.
func TestRedeemLoginCodeAfterLosingTheRegistration(t *testing.T) {
	r := newRig(t)
	late := r.requestCode(t, "ada@example.com")
	first := r.signUp(t, "ada@example.com")
	r.users.set(func(u *users) { u.staleEmail = 1 })

	got, err := r.svc.RedeemLoginCode(asAuth(t.Context()), late)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != first.ID {
		t.Errorf("signed in %s, want %s", got.ID, first.ID)
	}
}

func TestGoogleSignInAfterLosingTheRegistration(t *testing.T) {
	r := newRig(t)
	first, _ := r.googleSignIn(t, google("1084", "ada@gmail.com"), "")
	// Both reads miss, as they would have before the first sign-in
	// committed.
	r.users.set(func(u *users) { u.staleEmail, u.staleSubject = 1, 1 })

	got, _ := r.googleSignIn(t, google("1084", "ada@gmail.com"), "")
	if got.ID != first.ID {
		t.Errorf("signed in %s, want %s", got.ID, first.ID)
	}
}

func TestUseCasesReportStoreFailures(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")
	code := r.requestCode(t, "nobody@example.com")
	down := errors.New("projection unreachable")
	r.users.set(func(u *users) { u.fail = down })

	calls := map[string]func() error{
		"RequestLoginLink": func() error { return r.svc.RequestLoginLink(t.Context(), "ada@example.com", "") },
		"RedeemLoginCode": func() error {
			_, err := r.svc.RedeemLoginCode(asAuth(t.Context()), code)
			return err
		},
		"Me": func() error {
			_, err := r.svc.Me(asOwner(t.Context(), u))
			return err
		},
		"RequestEmailChange": func() error { return r.svc.RequestEmailChange(asOwner(t.Context(), u), "grace@example.com") },
		"GoogleSignIn": func() error {
			_, err := r.svc.GoogleSignIn(t.Context(), google("1084", "ada@example.com"), "")
			return err
		},
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, down) {
			t.Errorf("%s = %v, want the store's error", name, err)
		}
	}
}

func TestGoogleSignInsOfANewAccountConcurrently(t *testing.T) {
	r := newRig(t)
	const signIns = 8
	codes := make([]string, signIns)
	errs := make([]error, signIns)
	var wg sync.WaitGroup
	for i := range signIns {
		wg.Go(func() {
			to, err := r.svc.GoogleSignIn(t.Context(), google("1084", "ada@gmail.com"), "")
			if err == nil {
				codes[i] = to.Query().Get("code")
			}
			errs[i] = err
		})
	}
	wg.Wait()
	var first uuid.UUID
	for i, err := range errs {
		if err != nil {
			t.Fatalf("sign-in %d: %v", i, err)
		}
		u, err := r.svc.RedeemLoginCode(asAuth(t.Context()), codes[i])
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = u.ID
		}
		if u.ID != first || u.GoogleSubject != "1084" {
			t.Errorf("sign-in %d is %+v, want %s linked to 1084", i, u, first)
		}
	}
}

func TestMeRefuses(t *testing.T) {
	r := newRig(t)
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"anonymous", t.Context(), apperr.ErrUnauthorized},
		{"a client for itself", asClient(t.Context()), apperr.ErrForbidden},
		{"a token without user:read", asUser(t.Context(), uuid.NewV7(), app.ScopeUserWrite), apperr.ErrForbidden},
		{"a user who doesn't exist", asUser(t.Context(), uuid.NewV7(), app.ScopeUserRead), apperr.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.svc.Me(tt.ctx)
			assertKind(t, "Me", err, tt.want)
		})
	}
}

func changeEmail(t *testing.T, r *rig, u store.User, to string) {
	t.Helper()
	if err := r.svc.RequestEmailChange(asOwner(t.Context(), u), to); err != nil {
		t.Fatal(err)
	}
	if _, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), u), r.lastMail(t).url.Query().Get("code")); err != nil {
		t.Fatal(err)
	}
}

func TestChangeEmail(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")

	if err := r.svc.RequestEmailChange(asOwner(t.Context(), u), "Grace@Example.com"); err != nil {
		t.Fatal(err)
	}
	m := r.lastMail(t)
	if m.topic != mail.TopicEmailChange || m.link.To != "grace@example.com" {
		t.Errorf("mailed %s to %s", m.topic, m.link.To)
	}
	if got := m.url.Host + m.url.Path; got != "app.example.com/profile/confirm-email" {
		t.Errorf("link goes to %s", got)
	}
	if me, _ := r.svc.Me(asOwner(t.Context(), u)); me.Email != "ada@example.com" {
		t.Errorf("the address changed before it was confirmed: %s", me.Email)
	}

	got, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), u), m.url.Query().Get("code"))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID || got.Email != "grace@example.com" {
		t.Errorf("confirmed %+v", got)
	}
	_, err = r.svc.ConfirmEmailChange(asOwner(t.Context(), u), m.url.Query().Get("code"))
	assertKind(t, "confirming again", err, apperr.ErrInvalid)

	// The new address signs in to the same account.
	if again := r.signUp(t, "grace@example.com"); again.ID != u.ID {
		t.Errorf("the new address signs in %s, want %s", again.ID, u.ID)
	}
}

func TestRequestEmailChangeRefuses(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")
	tests := []struct {
		name  string
		ctx   context.Context
		email string
		want  error
	}{
		{"anonymous", t.Context(), "grace@example.com", apperr.ErrUnauthorized},
		{"a client for itself", asClient(t.Context()), "grace@example.com", apperr.ErrForbidden},
		{"a token without user:write", asUser(t.Context(), u.ID, app.ScopeUserRead), "grace@example.com", apperr.ErrForbidden},
		{"a bad address", asOwner(t.Context(), u), "grace", apperr.ErrInvalid},
		{"the same address", asOwner(t.Context(), u), "ADA@example.com", apperr.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.svc.RequestEmailChange(tt.ctx, tt.email)
			assertKind(t, "RequestEmailChange", err, tt.want)
		})
	}
	if n := len(r.mailed(t)); n != 0 {
		t.Errorf("mailed %d links", n)
	}
}

func TestConfirmEmailChangeRefusesAnotherUsersCode(t *testing.T) {
	r := newRig(t)
	ada, eve := r.signUp(t, "ada@example.com"), r.signUp(t, "eve@example.com")
	if err := r.svc.RequestEmailChange(asOwner(t.Context(), ada), "grace@example.com"); err != nil {
		t.Fatal(err)
	}
	code := r.lastMail(t).url.Query().Get("code")

	_, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), eve), code)
	assertKind(t, "eve confirming ada's code", err, apperr.ErrInvalid)
	if _, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), ada), code); err != nil {
		t.Errorf("eve's attempt used ada's code up: %v", err)
	}
	for _, bad := range []string{"", "nope", strings.Repeat("A", 43)} {
		_, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), ada), bad)
		assertKind(t, "confirming "+bad, err, apperr.ErrInvalid)
	}
	// A sign-in code is no email-change code.
	_, err = r.svc.ConfirmEmailChange(asOwner(t.Context(), ada), r.requestCode(t, "ada@example.com"))
	assertKind(t, "confirming a sign-in code", err, apperr.ErrInvalid)
}

func TestConfirmEmailChangeToATakenAddress(t *testing.T) {
	r := newRig(t)
	ada := r.signUp(t, "ada@example.com")
	if err := r.svc.RequestEmailChange(asOwner(t.Context(), ada), "grace@example.com"); err != nil {
		t.Fatal(err)
	}
	code := r.lastMail(t).url.Query().Get("code")
	grace := r.signUp(t, "grace@example.com")

	_, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), ada), code)
	if !errors.Is(err, store.ErrEmailTaken) || !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("confirming a taken address: err = %v, want store.ErrEmailTaken", err)
	}

	// Once the address is free again, the same code still works.
	changeEmail(t, r, grace, "grace.hopper@example.com")
	got, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), ada), code)
	if err != nil || got.Email != "grace@example.com" {
		t.Errorf("confirm after the address was freed = %+v, %v", got, err)
	}
}

func TestConfirmEmailChangeExpires(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")
	if err := r.svc.RequestEmailChange(asOwner(t.Context(), u), "grace@example.com"); err != nil {
		t.Fatal(err)
	}
	r.advance(app.LoginCodeTTL)
	_, err := r.svc.ConfirmEmailChange(asOwner(t.Context(), u), r.lastMail(t).url.Query().Get("code"))
	assertKind(t, "confirming at expiry", err, apperr.ErrInvalid)
}

func google(sub, email string) app.GoogleAccount {
	return app.GoogleAccount{Subject: sub, Email: email, EmailVerified: true}
}

func (r *rig) googleSignIn(t *testing.T, g app.GoogleAccount, next string) (store.User, *url.URL) {
	t.Helper()
	to, err := r.svc.GoogleSignIn(t.Context(), g, next)
	if err != nil {
		t.Fatal(err)
	}
	u, err := r.svc.RedeemLoginCode(asAuth(t.Context()), to.Query().Get("code"))
	if err != nil {
		t.Fatal(err)
	}
	return u, to
}

func TestGoogleSignInRegistersANewAccount(t *testing.T) {
	r := newRig(t)
	u, to := r.googleSignIn(t, google("1084", "Ada@Gmail.com"), "/profile")

	if got := to.Scheme + "://" + to.Host + to.Path; got != "https://app.example.com/sign-in/complete" {
		t.Errorf("redirect goes to %s", got)
	}
	if to.Query().Get("next") != "/profile" {
		t.Errorf("next = %q", to.Query().Get("next"))
	}
	if u.Email != "ada@gmail.com" || u.GoogleSubject != "1084" {
		t.Errorf("signed in %+v", u)
	}
	if n := len(r.mailed(t)); n != 0 {
		t.Errorf("mailed %d messages; a Google sign-in mails nothing", n)
	}

	again, _ := r.googleSignIn(t, google("1084", "renamed@gmail.com"), "")
	if again.ID != u.ID {
		t.Errorf("signing in again with the account is user %s, want %s", again.ID, u.ID)
	}
}

func TestGoogleSignInLinksTheAccountWithItsAddress(t *testing.T) {
	r := newRig(t)
	existing := r.signUp(t, "ada@gmail.com")

	u, _ := r.googleSignIn(t, google("1084", "ada@gmail.com"), "")
	if u.ID != existing.ID || u.GoogleSubject != "1084" {
		t.Errorf("signed in %+v, want %s linked to 1084", u, existing.ID)
	}
}

func TestGoogleSignInCodeIsShortLived(t *testing.T) {
	r := newRig(t)
	to, err := r.svc.GoogleSignIn(t.Context(), google("1084", "ada@gmail.com"), "")
	if err != nil {
		t.Fatal(err)
	}
	r.advance(app.GoogleCodeTTL)
	_, err = r.svc.RedeemLoginCode(asAuth(t.Context()), to.Query().Get("code"))
	assertKind(t, "redeeming after 2 minutes", err, apperr.ErrUnauthorized)
}

func TestGoogleSignInRefuses(t *testing.T) {
	r := newRig(t)
	r.googleSignIn(t, google("1084", "ada@gmail.com"), "")

	tests := []struct {
		name string
		g    app.GoogleAccount
		want error
	}{
		{"an unverified address", app.GoogleAccount{Subject: "2048", Email: "eve@gmail.com"}, apperr.ErrForbidden},
		{"a bad address", google("2048", "eve"), apperr.ErrInvalid},
		{"a second account with a linked user's address", google("2048", "ada@gmail.com"), apperr.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.svc.GoogleSignIn(t.Context(), tt.g, "")
			assertKind(t, "GoogleSignIn", err, tt.want)
		})
	}
}

func TestLinkGoogle(t *testing.T) {
	r := newRig(t)
	u := r.signUp(t, "ada@example.com")

	to, err := r.svc.LinkGoogle(asOwner(t.Context(), u), "1084")
	if err != nil {
		t.Fatal(err)
	}
	if to.String() != "https://app.example.com/profile?google=linked" {
		t.Errorf("redirect = %s", to)
	}
	if me, _ := r.svc.Me(asOwner(t.Context(), u)); me.GoogleSubject != "1084" {
		t.Errorf("subject = %q", me.GoogleSubject)
	}
	if _, err := r.svc.LinkGoogle(asOwner(t.Context(), u), "1084"); err != nil {
		t.Errorf("linking the same account again: %v", err)
	}

	// Google sign-in now finds this user, whatever the Google address is.
	if got, _ := r.googleSignIn(t, google("1084", "other@gmail.com"), ""); got.ID != u.ID {
		t.Errorf("Google sign-in is user %s, want %s", got.ID, u.ID)
	}
}

func TestLinkGoogleRefuses(t *testing.T) {
	r := newRig(t)
	ada, eve := r.signUp(t, "ada@example.com"), r.signUp(t, "eve@example.com")
	if _, err := r.svc.LinkGoogle(asOwner(t.Context(), ada), "1084"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		ctx     context.Context
		subject string
		want    error
	}{
		{"anonymous", t.Context(), "2048", apperr.ErrUnauthorized},
		{"a client for itself", asClient(t.Context()), "2048", apperr.ErrForbidden},
		{"a token without user:write", asUser(t.Context(), eve.ID, app.ScopeUserRead), "2048", apperr.ErrForbidden},
		{"another user's Google account", asOwner(t.Context(), eve), "1084", apperr.ErrConflict},
		{"a second Google account", asOwner(t.Context(), ada), "2048", apperr.ErrConflict},
		{"no account", asOwner(t.Context(), eve), "", apperr.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.svc.LinkGoogle(tt.ctx, tt.subject)
			assertKind(t, "LinkGoogle", err, tt.want)
		})
	}
}

func TestPurgeExpiredCodes(t *testing.T) {
	r := newRig(t)
	if err := r.svc.RequestLoginLink(t.Context(), "ada@example.com", ""); err != nil {
		t.Fatal(err)
	}
	r.advance(time.Minute)
	if err := r.svc.RequestLoginLink(t.Context(), "ada@example.com", ""); err != nil {
		t.Fatal(err)
	}
	links := r.mailed(t)

	r.advance(app.LoginCodeTTL - time.Minute)
	if err := r.svc.PurgeExpiredCodes(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.svc.RedeemLoginCode(asAuth(t.Context()), links[1].url.Query().Get("code")); err != nil {
		t.Errorf("the purge took a live code: %v", err)
	}
	n, err := r.codes.DeleteExpired(t.Context(), t0.Add(time.Hour))
	if err != nil || n != 1 {
		t.Errorf("after the purge, %d expired codes were left, %v; want only the redeemed one", n, err)
	}
}

func TestSafeNext(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"/", "/"},
		{"/profile", "/profile"},
		{"/consent?request=42&x=%2F", "/consent?request=42&x=%2F"},
		{"/a/b#frag", "/a/b#frag"},
		{"profile", ""},
		{"//evil.example", ""},
		{"///evil.example", ""},
		{`/\evil.example`, ""},
		{`/a\b`, ""},
		{"https://evil.example/", ""},
		{"javascript:alert(1)", ""},
		{"/\t/evil.example", ""},
		{"/\n/evil.example", ""},
		{"/a b", ""},
		{"/%zz", ""},
		{"/" + strings.Repeat("a", 2048), ""},
	}
	for _, tt := range tests {
		if got := app.SafeNext(tt.in); got != tt.want {
			t.Errorf("SafeNext(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func mustEmail(s string) domain.Email {
	e, err := domain.ParseEmail(s)
	if err != nil {
		panic(err)
	}
	return e
}
