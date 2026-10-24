// Package uow defines the transaction that stores share.
//
// Everything written through one Tx commits or rolls back together: a
// stream's new events, the projections updated from them and the outbox
// messages they cause. Each backend (memdb, mysql, mongodb) implements
// UnitOfWork, and its stores take the Tx it hands out.
package uow

import "context"

// Tx is one open transaction. Each backend has its own type; its stores
// unwrap it and return an error for another backend's Tx.
type Tx interface {
	// Backend names the backend that opened the transaction, for errors.
	Backend() string
}

// UnitOfWork runs functions in transactions.
type UnitOfWork interface {
	// Do runs fn in a new transaction. It commits when fn returns nil and
	// rolls back when fn returns an error, which Do then returns. A backend
	// may run fn more than once to retry a transient failure, so fn must do
	// nothing but write through tx.
	Do(ctx context.Context, fn func(tx Tx) error) error
}
