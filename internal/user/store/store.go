// Package store defines what the user service keeps besides its event
// streams: the users projection and the one-time login codes. The memory,
// mysql and mongodb packages implement it, and storetest.Test holds each of
// them to the same rules.
//
// Writes take the transaction (uow.Tx) they belong to, so a code, a
// stream's events, the projection and the outbox commit together. Reads see
// committed data only and run outside transactions.
package store

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
)

// User is a row of the users projection.
type User struct {
	ID    uuid.UUID
	Email string
	// GoogleSubject is the linked Google account's sub, "" when none is.
	GoogleSubject string
	CreatedAt     time.Time
}

var (
	// ErrEmailTaken means another user has the address. It is an
	// apperr.ErrConflict.
	ErrEmailTaken = apperr.New(apperr.ErrConflict, "This email address belongs to another account.")
	// ErrGoogleTaken means the Google account is linked to another user. It
	// is an apperr.ErrConflict.
	ErrGoogleTaken = apperr.New(apperr.ErrConflict, "This Google account is linked to another account.")
)

// Users is the users projection: one row per user stream, kept in step by
// the repository, which runs Project in the transaction that appends the
// events.
type Users interface {
	// Project applies newly stored user events as part of tx. Running it
	// again on the same events leaves the same rows. An address another
	// user has fails with ErrEmailTaken (addresses are lower-cased
	// domain.Emails, so case never tells two apart); a Google subject
	// another user has, with ErrGoogleTaken.
	Project(ctx context.Context, tx uow.Tx, events []es.Event) error
	// ByID returns the user with id, or an apperr.ErrNotFound.
	ByID(ctx context.Context, id uuid.UUID) (User, error)
	// ByEmail returns the user with the address, or an apperr.ErrNotFound.
	ByEmail(ctx context.Context, email domain.Email) (User, error)
	// ByGoogleSubject returns the user the Google account is linked to, or
	// an apperr.ErrNotFound.
	ByGoogleSubject(ctx context.Context, subject string) (User, error)
}

// CodeHash is the SHA-256 of a login code. Only the hash is stored, so a
// leaked table signs nobody in.
type CodeHash [32]byte

// Purpose is what a login code is for. A code redeemed for another purpose
// is no code at all.
type Purpose string

// The purposes.
const (
	// PurposeLogin signs in, registering the address when it is new.
	PurposeLogin Purpose = "login"
	// PurposeEmailChange confirms a new address for UserID.
	PurposeEmailChange Purpose = "email_change"
)

// LoginCode is a one-time code mailed to an address.
type LoginCode struct {
	Hash    CodeHash
	Purpose Purpose
	// Email is the address the code went to.
	Email string
	// UserID is the user the code is for, or uuid.Nil() for an address no
	// account had when the code was sent.
	UserID uuid.UUID
	// Next is the same-origin path to open after signing in, or "".
	Next      string
	ExpiresAt time.Time
}

// ErrCodeUnusable means no code has the hash and purpose, or it was used,
// or it expired. Callers decide what that means to their client.
var ErrCodeUnusable = errors.New("login code is unknown, used or expired")

// LoginCodes keeps login codes until they are used or expire.
type LoginCodes interface {
	// Add stores a new code as part of tx.
	Add(ctx context.Context, tx uow.Tx, code LoginCode) error
	// Find returns the code with hash and purpose if it is unused and
	// expires after now, and ErrCodeUnusable otherwise. It uses nothing up.
	Find(ctx context.Context, hash CodeHash, purpose Purpose, now time.Time) (LoginCode, error)
	// Consume marks the code used as part of tx, under the same conditions
	// as Find, and returns it. Of concurrent consumes of one code, exactly
	// one succeeds; the rest get ErrCodeUnusable. A rolled-back consume
	// leaves the code usable.
	Consume(ctx context.Context, tx uow.Tx, hash CodeHash, purpose Purpose, now time.Time) (LoginCode, error)
	// DeleteExpired removes the codes that expired at or before now, used
	// or not, and returns how many it removed.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
