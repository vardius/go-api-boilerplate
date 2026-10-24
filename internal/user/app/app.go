// Package app holds the user service's use cases, one method each.
//
// The caller is always the authn.Identity in the context, never something
// the request body says. Every write is one unit of work: the user's
// events, the projection they update, the login code and the outbox
// message that mails it commit together or not at all. Aggregates change
// before the transaction starts, because a backend may run the
// transaction's function more than once.
package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// The scopes the use cases need.
const (
	ScopeUserRead  = "user:read"
	ScopeUserWrite = "user:write"
	// ScopeRedeemLoginCodes is held only by the auth service's own token.
	ScopeRedeemLoginCodes = "login-codes:redeem"
)

const (
	// LoginCodeTTL is how long a mailed code works.
	LoginCodeTTL = 15 * time.Minute
	// GoogleCodeTTL is how long the code a Google sign-in ends with works:
	// the browser redeems it at once.
	GoogleCodeTTL = 2 * time.Minute
)

// Deps are what a Service works with.
type Deps struct {
	UnitOfWork uow.UnitOfWork
	Events     es.Store
	Outbox     outbox.Store
	Users      store.Users
	LoginCodes store.LoginCodes
	// WebURL is the web app's origin. Mailed links and redirects point
	// into it.
	WebURL *url.URL
	// Now is the clock. It defaults to time.Now.
	Now func() time.Time
}

// Service runs the use cases.
type Service struct {
	uow    uow.UnitOfWork
	repo   *es.Repository[*domain.User]
	outbox outbox.Store
	users  store.Users
	codes  store.LoginCodes
	webURL *url.URL
	now    func() time.Time
}

// New returns a Service on d.
func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	registry := es.NewRegistry()
	domain.RegisterEvents(registry)
	return &Service{
		uow: d.UnitOfWork,
		repo: es.NewRepository(es.RepositoryConfig[*domain.User]{
			StreamType: domain.StreamType,
			New:        domain.New,
			Store:      d.Events,
			Registry:   registry,
			Project:    d.Users.Project,
			Now:        d.Now,
		}),
		outbox: d.Outbox,
		users:  d.Users,
		codes:  d.LoginCodes,
		webURL: d.WebURL,
		now:    d.Now,
	}
}

// clock returns the time to millisecond precision, the finest every store
// keeps, so a code expires at the same instant whatever holds it.
func (s *Service) clock() time.Time { return s.now().UTC().Truncate(time.Millisecond) }

var errNotSignedIn = apperr.New(apperr.ErrUnauthorized, "This needs an access token.")

// caller returns the identity in ctx, which must have every scope in
// scopes.
func caller(ctx context.Context, scopes ...string) (authn.Identity, error) {
	id, ok := authn.FromContext(ctx)
	if !ok {
		return authn.Identity{}, errNotSignedIn
	}
	if err := authn.MissingScope(id, scopes); err != nil {
		return authn.Identity{}, err
	}
	return id, nil
}

// Actor returns the user ctx's identity acts for, which must hold every
// scope in scopes: an apperr.ErrUnauthorized for an anonymous request, and
// an apperr.ErrForbidden for a client acting for itself or a missing scope.
func Actor(ctx context.Context, scopes ...string) (uuid.UUID, error) {
	id, ok := authn.FromContext(ctx)
	switch {
	case !ok:
		return uuid.Nil(), errNotSignedIn
	case !id.IsUser():
		return uuid.Nil(), apperr.New(apperr.ErrForbidden, "This needs a user's access token.")
	}
	if err := authn.MissingScope(id, scopes); err != nil {
		return uuid.Nil(), err
	}
	return id.UserID, nil
}

// meta says who caused an event: identifiers only, as events keep them
// forever.
func meta(ctx context.Context) es.Meta {
	m := es.Meta{CorrelationID: obs.RequestID(ctx)}
	if id, ok := authn.FromContext(ctx); ok {
		m.ActorID = id.Subject
	}
	return m
}

// codeBytes is the size of a login code before encoding: 256 bits.
const codeBytes = 32

// newCode returns a fresh login code and the hash that is stored for it.
func newCode() (string, store.CodeHash) {
	b := make([]byte, codeBytes)
	_, _ = rand.Read(b) // crypto/rand.Read never fails.
	code := base64.RawURLEncoding.EncodeToString(b)
	return code, sha256.Sum256([]byte(code))
}

// codeHash returns the hash of code, and false for anything newCode can't
// have made, which no stored code matches.
func codeHash(code string) (store.CodeHash, bool) {
	if b, err := base64.RawURLEncoding.DecodeString(code); err != nil || len(b) != codeBytes {
		return store.CodeHash{}, false
	}
	return sha256.Sum256([]byte(code)), true
}

// maxNextBytes bounds the path kept with a code and put in its link.
const maxNextBytes = 2048

// SafeNext returns next when it is a path on the web app's own origin, such
// as "/consent?request=42", and "" for anything else: an absolute or
// scheme-relative URL ("//evil.example"), a backslash, which browsers read
// as a slash, or a control character. Redirecting to what it returns can't
// leave the site.
func SafeNext(next string) string {
	if next == "" || len(next) > maxNextBytes || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return ""
	}
	if strings.ContainsFunc(next, func(r rune) bool { return r == '\\' || r <= ' ' || r == 0x7f }) {
		return ""
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" {
		return ""
	}
	return next
}

// link returns the web app's page at path with query.
func (s *Service) link(query url.Values, path ...string) *url.URL {
	u := s.webURL.JoinPath(path...)
	if !strings.HasPrefix(u.Path, "/") {
		// JoinPath keeps the path relative when WEB_URL has none.
		u.Path, u.RawPath = "/"+u.Path, ""
	}
	u.RawQuery = query.Encode()
	return u
}

func (s *Service) signInLink(code, next string) *url.URL {
	q := url.Values{"code": {code}}
	if next != "" {
		q.Set("next", next)
	}
	return s.link(q, "sign-in", "complete")
}

// RequestLoginLink mails a sign-in link to email, whether or not an account
// has the address: redeeming the link registers one. It reports nothing
// about the address beyond whether it is one, so it can't tell anyone who
// has an account. next, if it is a SafeNext path, is where the link leads
// after signing in.
func (s *Service) RequestLoginLink(ctx context.Context, email, next string) error {
	addr, err := domain.ParseEmail(email)
	if err != nil {
		return err
	}
	next = SafeNext(next)
	var userID uuid.UUID
	switch u, err := s.users.ByEmail(ctx, addr); {
	case err == nil:
		userID = u.ID
	case !errors.Is(err, apperr.ErrNotFound):
		return fmt.Errorf("request login link: %w", err)
	}

	now := s.clock()
	code, hash := newCode()
	lc := store.LoginCode{Hash: hash, Purpose: store.PurposeLogin, Email: addr.String(), UserID: userID, Next: next, ExpiresAt: now.Add(LoginCodeTTL)}
	msg, err := outbox.NewMessage(mail.TopicLoginLink, mail.Link{To: addr.String(), URL: s.signInLink(code, next).String(), ExpiresAt: lc.ExpiresAt}, now)
	if err != nil {
		return fmt.Errorf("request login link: %w", err)
	}
	err = s.uow.Do(ctx, func(tx uow.Tx) error {
		if err := s.codes.Add(ctx, tx, lc); err != nil {
			return err
		}
		return s.outbox.Add(ctx, tx, msg)
	})
	if err != nil {
		return fmt.Errorf("request login link: %w", err)
	}
	return nil
}

var errBadLoginCode = apperr.New(apperr.ErrUnauthorized, "This sign-in code is unknown, used or expired.")

// RedeemLoginCode uses up a sign-in code and returns the user it signs in,
// registering the address first when no account has it. Only a caller with
// ScopeRedeemLoginCodes, the auth service, may redeem. A code that is
// unknown, used, expired or for an address its account no longer has is an
// apperr.ErrUnauthorized.
func (s *Service) RedeemLoginCode(ctx context.Context, code string) (store.User, error) {
	if _, err := caller(ctx, ScopeRedeemLoginCodes); err != nil {
		return store.User{}, err
	}
	hash, ok := codeHash(code)
	if !ok {
		return store.User{}, errBadLoginCode
	}
	now := s.clock()
	lc, err := s.codes.Find(ctx, hash, store.PurposeLogin, now)
	if errors.Is(err, store.ErrCodeUnusable) {
		return store.User{}, errBadLoginCode
	}
	if err != nil {
		return store.User{}, fmt.Errorf("redeem login code: %w", err)
	}
	u, err := s.redeem(ctx, hash, lc, now)
	if errors.Is(err, store.ErrEmailTaken) {
		// Another code for the address registered it first, and this one
		// is still unused: sign in to the account it made.
		u, err = s.redeem(ctx, hash, lc, now)
	}
	if errors.Is(err, store.ErrCodeUnusable) {
		return store.User{}, errBadLoginCode
	}
	if err != nil {
		return store.User{}, fmt.Errorf("redeem login code: %w", err)
	}
	return u, nil
}

// redeem consumes the code lc came from and signs in the user it is for,
// registering one when needed.
func (s *Service) redeem(ctx context.Context, hash store.CodeHash, lc store.LoginCode, now time.Time) (store.User, error) {
	u, newUser, err := s.signInAs(ctx, lc)
	if err != nil {
		return store.User{}, err
	}
	err = s.uow.Do(ctx, func(tx uow.Tx) error {
		if _, err := s.codes.Consume(ctx, tx, hash, store.PurposeLogin, now); err != nil {
			return err
		}
		if newUser == nil {
			return nil
		}
		events, err := s.repo.Save(ctx, tx, newUser, meta(ctx))
		if err != nil {
			return err
		}
		u.CreatedAt = events[0].OccurredAt
		return nil
	})
	return u, err
}

// signInAs resolves who lc signs in: the account it was sent for, the
// account that has its address now, or a new one, returned as an aggregate
// to save.
func (s *Service) signInAs(ctx context.Context, lc store.LoginCode) (store.User, *domain.User, error) {
	if lc.UserID != uuid.Nil() {
		u, err := s.users.ByID(ctx, lc.UserID)
		if errors.Is(err, apperr.ErrNotFound) {
			return store.User{}, nil, errBadLoginCode
		}
		if err != nil {
			return store.User{}, nil, err
		}
		// The account moved to another address after the link was sent,
		// and the old address may have another owner now.
		if u.Email != lc.Email {
			return store.User{}, nil, errBadLoginCode
		}
		return u, nil, nil
	}

	addr, err := domain.ParseEmail(lc.Email)
	if err != nil {
		return store.User{}, nil, fmt.Errorf("stored address %q: %w", lc.Email, err)
	}
	u, err := s.users.ByEmail(ctx, addr)
	if err == nil {
		return u, nil, nil
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return store.User{}, nil, err
	}
	user := domain.New()
	if err := user.Register(uuid.NewV7(), addr); err != nil {
		return store.User{}, nil, err
	}
	return store.User{ID: user.ID(), Email: addr.String()}, user, nil
}

// Me returns the calling user, who needs ScopeUserRead.
func (s *Service) Me(ctx context.Context) (store.User, error) {
	id, err := Actor(ctx, ScopeUserRead)
	if err != nil {
		return store.User{}, err
	}
	u, err := s.users.ByID(ctx, id)
	if err != nil {
		return store.User{}, fmt.Errorf("me: %w", err)
	}
	return u, nil
}

// RequestEmailChange mails a confirmation link to email, the address the
// calling user wants instead of theirs. The address changes only when the
// link's code comes back through ConfirmEmailChange, and whether another
// account has it is told only then.
func (s *Service) RequestEmailChange(ctx context.Context, email string) error {
	id, err := Actor(ctx, ScopeUserWrite)
	if err != nil {
		return err
	}
	addr, err := domain.ParseEmail(email)
	if err != nil {
		return err
	}
	u, err := s.users.ByID(ctx, id)
	if err != nil {
		return fmt.Errorf("request email change: %w", err)
	}
	if u.Email == addr.String() {
		return domain.ErrSameEmail
	}

	now := s.clock()
	code, hash := newCode()
	lc := store.LoginCode{Hash: hash, Purpose: store.PurposeEmailChange, Email: addr.String(), UserID: id, ExpiresAt: now.Add(LoginCodeTTL)}
	link := s.link(url.Values{"code": {code}}, "profile", "confirm-email")
	msg, err := outbox.NewMessage(mail.TopicEmailChange, mail.Link{To: addr.String(), URL: link.String(), ExpiresAt: lc.ExpiresAt}, now)
	if err != nil {
		return fmt.Errorf("request email change: %w", err)
	}
	err = s.uow.Do(ctx, func(tx uow.Tx) error {
		if err := s.codes.Add(ctx, tx, lc); err != nil {
			return err
		}
		return s.outbox.Add(ctx, tx, msg)
	})
	if err != nil {
		return fmt.Errorf("request email change: %w", err)
	}
	return nil
}

// errBadEmailChangeCode is an ErrInvalid, not an ErrUnauthorized: the
// caller is signed in, and a 401 would send the web app to refresh a
// session that is fine.
var errBadEmailChangeCode = apperr.Invalid("code", "is unknown, used or expired")

// ConfirmEmailChange uses up the calling user's email-change code and moves
// the account to the address it was mailed to. Another user's code is no
// code at all. An address another account has taken since is
// store.ErrEmailTaken, and the code stays usable.
func (s *Service) ConfirmEmailChange(ctx context.Context, code string) (store.User, error) {
	id, err := Actor(ctx, ScopeUserWrite)
	if err != nil {
		return store.User{}, err
	}
	hash, ok := codeHash(code)
	if !ok {
		return store.User{}, errBadEmailChangeCode
	}
	now := s.clock()
	lc, err := s.codes.Find(ctx, hash, store.PurposeEmailChange, now)
	switch {
	case errors.Is(err, store.ErrCodeUnusable), err == nil && lc.UserID != id:
		return store.User{}, errBadEmailChangeCode
	case err != nil:
		return store.User{}, fmt.Errorf("confirm email change: %w", err)
	}
	addr, err := domain.ParseEmail(lc.Email)
	if err != nil {
		return store.User{}, fmt.Errorf("confirm email change: stored address %q: %w", lc.Email, err)
	}
	user, err := s.repo.Load(ctx, id)
	if err != nil {
		return store.User{}, fmt.Errorf("confirm email change: %w", err)
	}
	if err := user.ChangeEmail(addr); err != nil {
		return store.User{}, err
	}

	err = s.uow.Do(ctx, func(tx uow.Tx) error {
		if _, err := s.codes.Consume(ctx, tx, hash, store.PurposeEmailChange, now); err != nil {
			return err
		}
		_, err := s.repo.Save(ctx, tx, user, meta(ctx))
		return err
	})
	if errors.Is(err, store.ErrCodeUnusable) {
		return store.User{}, errBadEmailChangeCode
	}
	if err != nil {
		return store.User{}, fmt.Errorf("confirm email change: %w", err)
	}
	return s.saved(ctx, id)
}

// saved reads back a user the use case just wrote.
func (s *Service) saved(ctx context.Context, id uuid.UUID) (store.User, error) {
	u, err := s.users.ByID(ctx, id)
	if err != nil {
		return store.User{}, fmt.Errorf("read user %s back: %w", id, err)
	}
	return u, nil
}

// GoogleAccount is who a verified Google ID token says signed in.
type GoogleAccount struct {
	Subject       string
	Email         string
	EmailVerified bool
}

// GoogleSignIn signs in the owner of a Google account: the user it is
// linked to, else the user with its address, whom it links, else a new user
// with both. It returns the web app's sign-in page with a short-lived code
// for the auth service to redeem, the same way a mailed link ends. Google
// must have verified the address, or anyone could claim one.
func (s *Service) GoogleSignIn(ctx context.Context, g GoogleAccount, next string) (*url.URL, error) {
	if !g.EmailVerified {
		return nil, apperr.New(apperr.ErrForbidden, "Google hasn't verified this account's email address.")
	}
	addr, err := domain.ParseEmail(g.Email)
	if err != nil {
		return nil, err
	}
	code, hash := newCode()
	next = SafeNext(next)
	err = s.googleSignIn(ctx, g.Subject, addr, store.LoginCode{Hash: hash, Purpose: store.PurposeLogin, Next: next})
	if errors.Is(err, store.ErrEmailTaken) || errors.Is(err, store.ErrGoogleTaken) {
		// A sign-in running alongside registered or linked first: this
		// attempt finds the account it made.
		err = s.googleSignIn(ctx, g.Subject, addr, store.LoginCode{Hash: hash, Purpose: store.PurposeLogin, Next: next})
	}
	if err != nil {
		return nil, fmt.Errorf("google sign-in: %w", err)
	}
	return s.signInLink(code, next), nil
}

// googleSignIn saves the user a Google account signs in, when it changed,
// and lc for them.
func (s *Service) googleSignIn(ctx context.Context, subject string, addr domain.Email, lc store.LoginCode) error {
	u, user, err := s.googleUser(ctx, subject, addr)
	if err != nil {
		return err
	}
	now := s.clock()
	lc.Email, lc.UserID, lc.ExpiresAt = u.Email, u.ID, now.Add(GoogleCodeTTL)
	return s.uow.Do(ctx, func(tx uow.Tx) error {
		if user != nil {
			if _, err := s.repo.Save(ctx, tx, user, meta(ctx)); err != nil {
				return err
			}
		}
		return s.codes.Add(ctx, tx, lc)
	})
}

// googleUser finds or makes the user a Google account signs in, returning
// the aggregate to save when it changed.
func (s *Service) googleUser(ctx context.Context, subject string, addr domain.Email) (store.User, *domain.User, error) {
	u, err := s.users.ByGoogleSubject(ctx, subject)
	if err == nil {
		return u, nil, nil
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return store.User{}, nil, err
	}

	var user *domain.User
	switch u, err := s.users.ByEmail(ctx, addr); {
	case err == nil:
		if user, err = s.repo.Load(ctx, u.ID); err != nil {
			return store.User{}, nil, err
		}
	case errors.Is(err, apperr.ErrNotFound):
		user = domain.New()
		if err := user.Register(uuid.NewV7(), addr); err != nil {
			return store.User{}, nil, err
		}
	default:
		return store.User{}, nil, err
	}
	if err := user.LinkGoogle(subject); err != nil {
		return store.User{}, nil, err
	}
	return store.User{ID: user.ID(), Email: user.Email().String(), GoogleSubject: subject}, user, nil
}

// LinkGoogle links the Google account with subject to the calling user, who
// needs ScopeUserWrite, and returns the web app's profile page to go back
// to. A Google account linked to someone else is store.ErrGoogleTaken.
func (s *Service) LinkGoogle(ctx context.Context, subject string) (*url.URL, error) {
	id, err := Actor(ctx, ScopeUserWrite)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("link google: %w", err)
	}
	if err := user.LinkGoogle(subject); err != nil {
		return nil, err
	}
	err = s.uow.Do(ctx, func(tx uow.Tx) error {
		_, err := s.repo.Save(ctx, tx, user, meta(ctx))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("link google: %w", err)
	}
	return s.link(url.Values{"google": {"linked"}}, "profile"), nil
}

// PurgeExpiredCodes deletes the login codes that have expired, which can
// never be used again.
func (s *Service) PurgeExpiredCodes(ctx context.Context) error {
	if _, err := s.codes.DeleteExpired(ctx, s.clock()); err != nil {
		return fmt.Errorf("purge expired login codes: %w", err)
	}
	return nil
}
