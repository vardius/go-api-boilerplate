// Package mongodb keeps the event store, the outbox and the services' read
// models in MongoDB, using the mongo-driver v2.
//
// Transactions need a replica set; one node is enough. DB.Do runs a function
// in a session transaction, and every store taking its Tx runs its
// operations inside it through SessionContext, so a stream's events, the
// projections updated from them and their outbox messages commit together.
//
// Documents store UUIDs as BSON binary subtype 4 and times as BSON dates,
// which keep milliseconds.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Config says where the database is.
type Config struct {
	URI      config.Secret `env:"URI,required" desc:"MongoDB connection string. Transactions need a replica set, so name one: mongodb://localhost:27017/?replicaSet=rs0"`
	Database string        `env:"DATABASE,required" desc:"Name of the MongoDB database the service keeps its data in"`
}

// DB is a connected database. It implements uow.UnitOfWork.
type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

var _ uow.UnitOfWork = (*DB)(nil)

// Connect connects to cfg.URI and pings the primary, so a wrong address
// fails at start rather than on the first request. Close the connection
// with db.Client.Disconnect.
func Connect(ctx context.Context, cfg Config) (*DB, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI.Reveal()))
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		// Disconnect closes the pool even when ctx has ended; the ping's
		// error is the one worth returning.
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongodb: ping: %w", err)
	}
	return &DB{Client: client, Database: client.Database(cfg.Database)}, nil
}

// Snapshot reads with majority commits give each transaction a consistent
// view and make a write that raced another fail with a write conflict.
var txOptions = options.Transaction().
	SetReadConcern(readconcern.Snapshot()).
	SetWriteConcern(writeconcern.Majority())

// Do runs fn in a session transaction. It commits when fn returns nil; when
// fn returns an error, it aborts and returns that error as it is. The
// driver runs fn again when the transaction fails transiently, such as on a
// write conflict with a concurrent transaction, for up to two minutes, and
// then returns a timeout error wrapping fn's last error. If fn panics, the
// transaction is aborted and the panic goes on.
func (db *DB) Do(ctx context.Context, fn func(tx uow.Tx) error) error {
	session, err := db.Client.StartSession()
	if err != nil {
		return fmt.Errorf("mongodb: start session: %w", err)
	}
	tx := &Tx{session: session}
	defer func() {
		tx.done = true
		// Ending the session aborts a transaction that fn's panic left open,
		// and must run even when ctx is what made fn fail.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		session.EndSession(ctx)
	}()

	var fnErr error
	_, err = session.WithTransaction(ctx, func(context.Context) (any, error) {
		fnErr = fn(tx)
		return nil, fnErr
	}, txOptions)
	if err != nil && fnErr == nil {
		return fmt.Errorf("mongodb: transaction: %w", err)
	}
	return err
}

// Tx is a MongoDB transaction: the session that DB.Do runs its function in.
// Like the session, it is not safe for concurrent use.
type Tx struct {
	session *mongo.Session
	done    bool
}

// Backend implements uow.Tx.
func (*Tx) Backend() string { return "mongodb" }

// ErrTxDone means a Tx was used after its Do returned.
var ErrTxDone = errors.New("mongodb: transaction has already finished")

// SessionContext returns ctx carrying tx's session. Operations run with it
// are part of the transaction. It fails for another backend's Tx and for a
// transaction that has finished.
func SessionContext(ctx context.Context, tx uow.Tx) (context.Context, error) {
	t, ok := tx.(*Tx)
	if !ok {
		return nil, fmt.Errorf("mongodb: got a %s transaction", backend(tx))
	}
	if t.done {
		return nil, ErrTxDone
	}
	return mongo.NewSessionContext(ctx, t.session), nil
}

func backend(tx uow.Tx) string {
	if tx == nil {
		return "nil"
	}
	return tx.Backend()
}

// IsDuplicate reports whether err is a unique index violation.
func IsDuplicate(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
