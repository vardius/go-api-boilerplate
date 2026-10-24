// Package memdb keeps stores in process memory, for tests and for running
// the services with no database. Data is lost when the process stops.
//
// All stores built on one DB share its lock, so a transaction spans the
// event store, projections and outbox as it would in a database: Do holds
// the write lock while its function runs and undoes every write when the
// function fails. Reads take the read lock, so they never see a write that
// might still be undone.
package memdb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// DB is the lock and transaction manager shared by memory stores.
type DB struct {
	mu sync.RWMutex
}

// New returns an empty DB.
func New() *DB { return &DB{} }

var _ uow.UnitOfWork = (*DB)(nil)

// Do runs fn holding the write lock. If fn returns an error or panics, the
// writes it made through its Tx are undone, newest first. The lock isn't
// reentrant: fn must not call Do or Read.
func (db *DB) Do(ctx context.Context, fn func(tx uow.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	tx := &Tx{}
	committed := false
	defer func() {
		if !committed {
			for _, v := range slices.Backward(tx.undo) {
				v()
			}
		}
		tx.done = true
	}()
	if err := fn(tx); err != nil {
		return err
	}
	committed = true
	return nil
}

// Read runs fn holding the read lock. Stores use it for reads that aren't
// part of a transaction, so fn must not call Do or Read.
func (db *DB) Read(fn func() error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return fn()
}

// Tx is a memdb transaction.
type Tx struct {
	undo []func()
	done bool
}

// Backend implements uow.Tx.
func (*Tx) Backend() string { return "memory" }

// OnRollback registers fn to undo a write the transaction made.
func (tx *Tx) OnRollback(fn func()) { tx.undo = append(tx.undo, fn) }

// ErrTxDone means a Tx was used after its Do returned.
var ErrTxDone = errors.New("memdb: transaction has already finished")

// From returns the memdb transaction behind tx. It fails for another
// backend's Tx and for a transaction that has finished, whose lock is no
// longer held.
func From(tx uow.Tx) (*Tx, error) {
	t, ok := tx.(*Tx)
	if !ok {
		return nil, fmt.Errorf("memdb: got a %s transaction", backend(tx))
	}
	if t.done {
		return nil, ErrTxDone
	}
	return t, nil
}

func backend(tx uow.Tx) string {
	if tx == nil {
		return "nil"
	}
	return tx.Backend()
}
