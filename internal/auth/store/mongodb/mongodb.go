// Package mongodb keeps the auth service's store in MongoDB.
//
// Each kind of thing has its collection, and token and code hashes are the
// _id of theirs. An operation that touches several documents runs in a
// transaction, whose snapshot makes the loser of two concurrent uses retry
// and find the thing already used. TTL indexes on expires_at delete what
// has expired, so this store needs no purge worker; lookups compare
// expires_at themselves, since the TTL monitor runs only once a minute.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	platform "github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// The collections.
const (
	clientsCollection  = "auth_clients"
	grantsCollection   = "auth_grants"
	tokensCollection   = "auth_refresh_tokens"
	requestsCollection = "auth_authorization_requests"
	codesCollection    = "auth_authorization_codes"
)

func ttl(collection string, after time.Duration) platform.Index {
	return platform.Index{
		Collection: collection,
		Model: mongo.IndexModel{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(after / time.Second)),
		},
	}
}

func index(collection string, keys ...string) platform.Index {
	d := bson.D{}
	for _, k := range keys {
		d = append(d, bson.E{Key: k, Value: 1})
	}
	return platform.Index{Collection: collection, Model: mongo.IndexModel{Keys: d}}
}

// Indexes are the indexes the store needs: those its queries sort and
// filter by, and the TTL indexes that delete what has expired.
var Indexes = []platform.Index{
	index(clientsCollection, "owner_id", "created_at", "_id"),
	index(grantsCollection, "user_id", "client_id", "last_used_at", "_id"),
	index(grantsCollection, "client_id"),
	ttl(grantsCollection, 0),
	index(tokensCollection, "grant_id"),
	index(tokensCollection, "client_id"),
	ttl(tokensCollection, 0),
	index(requestsCollection, "client_id"),
	ttl(requestsCollection, 0),
	index(codesCollection, "client_id"),
	ttl(codesCollection, store.UsedCodeRetention),
}

// EnsureIndexes creates Indexes. It is safe to run at every start.
func EnsureIndexes(ctx context.Context, db *platform.DB) error {
	return platform.EnsureIndexes(ctx, db.Database, Indexes...)
}

// Store is a store.Store in MongoDB.
type Store struct {
	db                                       *platform.DB
	clients, grants, tokens, requests, codes *mongo.Collection
}

var _ store.Store = (*Store)(nil)

// New returns the store on db, which needs EnsureIndexes.
func New(db *platform.DB) *Store {
	return &Store{
		db:       db,
		clients:  db.Database.Collection(clientsCollection),
		grants:   db.Database.Collection(grantsCollection),
		tokens:   db.Database.Collection(tokensCollection),
		requests: db.Database.Collection(requestsCollection),
		codes:    db.Database.Collection(codesCollection),
	}
}

// tx runs fn in a transaction, with ctx carrying its session. fn may run
// more than once.
func (s *Store) tx(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.db.Do(ctx, func(tx uow.Tx) error {
		sctx, err := platform.SessionContext(ctx, tx)
		if err != nil {
			return err
		}
		return fn(sctx)
	})
}

// UUIDs are BSON binary subtype 4, as the platform's documents keep them.
func binaryUUID(id uuid.UUID) bson.Binary {
	return bson.Binary{Subtype: bson.TypeBinaryUUID, Data: id[:]}
}

func parseUUID(b bson.Binary) (uuid.UUID, error) {
	if b.Subtype != bson.TypeBinaryUUID || len(b.Data) != len(uuid.UUID{}) {
		return uuid.UUID{}, fmt.Errorf("want a UUID, got %d bytes of binary subtype %#x", len(b.Data), b.Subtype)
	}
	return uuid.UUID(b.Data), nil
}

// uuidField is a UUID to parse from a document into dst.
type uuidField struct {
	dst *uuid.UUID
	src bson.Binary
}

func parseUUIDs(fields ...uuidField) error {
	for _, f := range fields {
		id, err := parseUUID(f.src)
		if err != nil {
			return err
		}
		*f.dst = id
	}
	return nil
}

func insertError(what string, err error) error {
	if platform.IsDuplicate(err) {
		return fmt.Errorf("mongodb: %s: %w: %w", what, apperr.ErrConflict, err)
	}
	return fmt.Errorf("mongodb: %s: %w", what, err)
}

func findError(what string, err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("mongodb: %s: %w", what, apperr.ErrNotFound)
	}
	return fmt.Errorf("mongodb: %s: %w", what, err)
}

type clientDoc struct {
	ID           string      `bson:"_id"`
	OwnerID      bson.Binary `bson:"owner_id"`
	Name         string      `bson:"name"`
	RedirectURIs []string    `bson:"redirect_uris"`
	Scopes       []string    `bson:"scopes"`
	SecretHash   string      `bson:"secret_hash"`
	CreatedAt    time.Time   `bson:"created_at"`
}

func (d clientDoc) client() (store.Client, error) {
	c := store.Client{ID: d.ID, Name: d.Name, RedirectURIs: d.RedirectURIs, Scopes: d.Scopes, SecretHash: d.SecretHash, CreatedAt: d.CreatedAt}
	return c, parseUUIDs(uuidField{&c.OwnerID, d.OwnerID})
}

// CreateClient implements store.Store.
func (s *Store) CreateClient(ctx context.Context, c store.Client) error {
	_, err := s.clients.InsertOne(ctx, clientDoc{
		ID: c.ID, OwnerID: binaryUUID(c.OwnerID), Name: c.Name, RedirectURIs: c.RedirectURIs,
		Scopes: c.Scopes, SecretHash: c.SecretHash, CreatedAt: c.CreatedAt,
	})
	if err != nil {
		return insertError("create client "+c.ID, err)
	}
	return nil
}

// Client implements store.Store.
func (s *Store) Client(ctx context.Context, id string) (store.Client, error) {
	var d clientDoc
	if err := s.clients.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&d); err != nil {
		return store.Client{}, findError("client "+id, err)
	}
	return d.client()
}

// Clients implements store.Store.
func (s *Store) Clients(ctx context.Context, owner uuid.UUID, limit, offset int) ([]store.Client, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))
	cur, err := s.clients.Find(ctx, bson.D{{Key: "owner_id", Value: binaryUUID(owner)}}, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: list clients: %w", err)
	}
	var docs []clientDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongodb: list clients: %w", err)
	}
	out := make([]store.Client, 0, len(docs))
	for _, d := range docs {
		c, err := d.client()
		if err != nil {
			return nil, fmt.Errorf("mongodb: list clients: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// DeleteClient implements store.Store.
func (s *Store) DeleteClient(ctx context.Context, owner uuid.UUID, id string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		res, err := s.clients.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}, {Key: "owner_id", Value: binaryUUID(owner)}})
		if err != nil {
			return fmt.Errorf("mongodb: delete client %s: %w", id, err)
		}
		if res.DeletedCount == 0 {
			return fmt.Errorf("mongodb: delete client %s: %w", id, apperr.ErrNotFound)
		}
		byClient := bson.D{{Key: "client_id", Value: id}}
		for _, coll := range []*mongo.Collection{s.grants, s.tokens, s.requests, s.codes} {
			if _, err := coll.DeleteMany(ctx, byClient); err != nil {
				return fmt.Errorf("mongodb: delete client %s from %s: %w", id, coll.Name(), err)
			}
		}
		return nil
	})
}

type grantDoc struct {
	ID         bson.Binary `bson:"_id"`
	UserID     bson.Binary `bson:"user_id"`
	ClientID   string      `bson:"client_id"`
	Scopes     []string    `bson:"scopes"`
	UserAgent  string      `bson:"user_agent"`
	CreatedAt  time.Time   `bson:"created_at"`
	LastUsedAt time.Time   `bson:"last_used_at"`
	ExpiresAt  time.Time   `bson:"expires_at"`
}

func (d grantDoc) grant() (store.Grant, error) {
	g := store.Grant{ClientID: d.ClientID, Scopes: d.Scopes, UserAgent: d.UserAgent, CreatedAt: d.CreatedAt, LastUsedAt: d.LastUsedAt, ExpiresAt: d.ExpiresAt}
	return g, parseUUIDs(uuidField{&g.ID, d.ID}, uuidField{&g.UserID, d.UserID})
}

// tokenDoc is a refresh token. It carries its grant's client so that
// deleting a client finds its tokens through an index.
type tokenDoc struct {
	Hash      []byte      `bson:"_id"`
	GrantID   bson.Binary `bson:"grant_id"`
	ClientID  string      `bson:"client_id"`
	ExpiresAt time.Time   `bson:"expires_at"`
	Rotated   bool        `bson:"rotated"`
}

func (s *Store) insertGrant(ctx context.Context, g store.Grant, first store.RefreshToken) error {
	_, err := s.grants.InsertOne(ctx, grantDoc{
		ID: binaryUUID(g.ID), UserID: binaryUUID(g.UserID), ClientID: g.ClientID, Scopes: g.Scopes,
		UserAgent: g.UserAgent, CreatedAt: g.CreatedAt, LastUsedAt: g.LastUsedAt, ExpiresAt: g.ExpiresAt,
	})
	if err != nil {
		return insertError("create grant "+g.ID.String(), err)
	}
	return s.insertToken(ctx, g.ID, g.ClientID, first)
}

func (s *Store) insertToken(ctx context.Context, grant uuid.UUID, client string, t store.RefreshToken) error {
	_, err := s.tokens.InsertOne(ctx, tokenDoc{Hash: t.Hash[:], GrantID: binaryUUID(grant), ClientID: client, ExpiresAt: t.ExpiresAt})
	if err != nil {
		return insertError("store a refresh token", err)
	}
	return nil
}

// CreateGrant implements store.Store.
func (s *Store) CreateGrant(ctx context.Context, g store.Grant, first store.RefreshToken) error {
	return s.tx(ctx, func(ctx context.Context) error { return s.insertGrant(ctx, g, first) })
}

func unexpired(now time.Time) bson.D {
	return bson.D{{Key: "$gt", Value: now}}
}

// Grant implements store.Store.
func (s *Store) Grant(ctx context.Context, id uuid.UUID, now time.Time) (store.Grant, error) {
	var d grantDoc
	err := s.grants.FindOne(ctx, bson.D{{Key: "_id", Value: binaryUUID(id)}, {Key: "expires_at", Value: unexpired(now)}}).Decode(&d)
	if err != nil {
		return store.Grant{}, findError("grant "+id.String(), err)
	}
	return d.grant()
}

// Grants implements store.Store.
func (s *Store) Grants(ctx context.Context, user uuid.UUID, client string, now time.Time, limit int) ([]store.Grant, error) {
	filter := bson.D{
		{Key: "user_id", Value: binaryUUID(user)},
		{Key: "client_id", Value: client},
		{Key: "expires_at", Value: unexpired(now)},
	}
	opts := options.Find().SetSort(bson.D{{Key: "last_used_at", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(int64(limit))
	cur, err := s.grants.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: list grants: %w", err)
	}
	var docs []grantDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongodb: list grants: %w", err)
	}
	out := make([]store.Grant, 0, len(docs))
	for _, d := range docs {
		g, err := d.grant()
		if err != nil {
			return nil, fmt.Errorf("mongodb: list grants: %w", err)
		}
		out = append(out, g)
	}
	return out, nil
}

// RevokeGrant implements store.Store.
func (s *Store) RevokeGrant(ctx context.Context, id uuid.UUID) error {
	return s.tx(ctx, func(ctx context.Context) error {
		if _, err := s.grants.DeleteOne(ctx, bson.D{{Key: "_id", Value: binaryUUID(id)}}); err != nil {
			return fmt.Errorf("mongodb: revoke grant %s: %w", id, err)
		}
		if _, err := s.tokens.DeleteMany(ctx, bson.D{{Key: "grant_id", Value: binaryUUID(id)}}); err != nil {
			return fmt.Errorf("mongodb: revoke grant %s: %w", id, err)
		}
		return nil
	})
}

// RefreshTokenGrant implements store.Store.
func (s *Store) RefreshTokenGrant(ctx context.Context, h store.Hash, now time.Time) (store.Grant, bool, error) {
	var t tokenDoc
	err := s.tokens.FindOne(ctx, bson.D{{Key: "_id", Value: h[:]}, {Key: "expires_at", Value: unexpired(now)}}).Decode(&t)
	if err != nil {
		return store.Grant{}, false, findError("refresh token", err)
	}
	var d grantDoc
	if err := s.grants.FindOne(ctx, bson.D{{Key: "_id", Value: t.GrantID}}).Decode(&d); err != nil {
		// A grant revoked between the two reads takes its tokens with it.
		return store.Grant{}, false, findError("refresh token's grant", err)
	}
	g, err := d.grant()
	return g, t.Rotated, err
}

// RotateRefreshToken implements store.Store.
func (s *Store) RotateRefreshToken(ctx context.Context, h store.Hash, next store.RefreshToken, now time.Time) error {
	return s.tx(ctx, func(ctx context.Context) error {
		var t tokenDoc
		if err := s.tokens.FindOne(ctx, bson.D{{Key: "_id", Value: h[:]}}).Decode(&t); err != nil {
			return findError("rotate refresh token", err)
		}
		switch {
		case !now.Before(t.ExpiresAt):
			return fmt.Errorf("mongodb: rotate refresh token: expired: %w", apperr.ErrNotFound)
		case t.Rotated:
			return fmt.Errorf("mongodb: rotate refresh token: %w", store.ErrUsed)
		}
		res, err := s.tokens.UpdateOne(ctx, bson.D{{Key: "_id", Value: h[:]}, {Key: "rotated", Value: false}}, bson.D{{Key: "$set", Value: bson.D{{Key: "rotated", Value: true}}}})
		if err != nil {
			return fmt.Errorf("mongodb: rotate refresh token: %w", err)
		}
		if res.MatchedCount == 0 {
			return fmt.Errorf("mongodb: rotate refresh token: %w", store.ErrUsed)
		}
		grant, err := parseUUID(t.GrantID)
		if err != nil {
			return fmt.Errorf("mongodb: rotate refresh token: %w", err)
		}
		if err := s.insertToken(ctx, grant, t.ClientID, next); err != nil {
			return err
		}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "last_used_at", Value: now}, {Key: "expires_at", Value: next.ExpiresAt}}}}
		if _, err := s.grants.UpdateOne(ctx, bson.D{{Key: "_id", Value: t.GrantID}}, update); err != nil {
			return fmt.Errorf("mongodb: rotate refresh token: %w", err)
		}
		return nil
	})
}

type requestDoc struct {
	ID            bson.Binary `bson:"_id"`
	UserID        bson.Binary `bson:"user_id"`
	ClientID      string      `bson:"client_id"`
	RedirectURI   string      `bson:"redirect_uri"`
	Scopes        []string    `bson:"scopes"`
	State         string      `bson:"state"`
	CodeChallenge string      `bson:"code_challenge"`
	ExpiresAt     time.Time   `bson:"expires_at"`
}

func (d requestDoc) request() (store.AuthorizationRequest, error) {
	r := store.AuthorizationRequest{
		ClientID: d.ClientID, RedirectURI: d.RedirectURI, Scopes: d.Scopes, State: d.State,
		CodeChallenge: d.CodeChallenge, ExpiresAt: d.ExpiresAt,
	}
	return r, parseUUIDs(uuidField{&r.ID, d.ID}, uuidField{&r.UserID, d.UserID})
}

// CreateAuthorizationRequest implements store.Store.
func (s *Store) CreateAuthorizationRequest(ctx context.Context, r store.AuthorizationRequest) error {
	_, err := s.requests.InsertOne(ctx, requestDoc{
		ID: binaryUUID(r.ID), UserID: binaryUUID(r.UserID), ClientID: r.ClientID, RedirectURI: r.RedirectURI,
		Scopes: r.Scopes, State: r.State, CodeChallenge: r.CodeChallenge, ExpiresAt: r.ExpiresAt,
	})
	if err != nil {
		return insertError("create authorization request "+r.ID.String(), err)
	}
	return nil
}

func requestFilter(id, user uuid.UUID, now time.Time) bson.D {
	return bson.D{{Key: "_id", Value: binaryUUID(id)}, {Key: "user_id", Value: binaryUUID(user)}, {Key: "expires_at", Value: unexpired(now)}}
}

// AuthorizationRequest implements store.Store.
func (s *Store) AuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	var d requestDoc
	if err := s.requests.FindOne(ctx, requestFilter(id, user, now)).Decode(&d); err != nil {
		return store.AuthorizationRequest{}, findError("authorization request "+id.String(), err)
	}
	return d.request()
}

// TakeAuthorizationRequest implements store.Store. findOneAndDelete is
// atomic, so only one caller gets the request.
func (s *Store) TakeAuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	var d requestDoc
	if err := s.requests.FindOneAndDelete(ctx, requestFilter(id, user, now)).Decode(&d); err != nil {
		return store.AuthorizationRequest{}, findError("take authorization request "+id.String(), err)
	}
	return d.request()
}

type codeDoc struct {
	Hash          []byte      `bson:"_id"`
	ClientID      string      `bson:"client_id"`
	UserID        bson.Binary `bson:"user_id"`
	RedirectURI   string      `bson:"redirect_uri"`
	Scopes        []string    `bson:"scopes"`
	CodeChallenge string      `bson:"code_challenge"`
	ExpiresAt     time.Time   `bson:"expires_at"`
	Used          bool        `bson:"used"`
	// GrantID is uuid.Nil() until a redemption creates a grant.
	GrantID bson.Binary `bson:"grant_id"`
}

// CreateAuthorizationCode implements store.Store.
func (s *Store) CreateAuthorizationCode(ctx context.Context, c store.AuthorizationCode) error {
	_, err := s.codes.InsertOne(ctx, codeDoc{
		Hash: c.Hash[:], ClientID: c.ClientID, UserID: binaryUUID(c.UserID), RedirectURI: c.RedirectURI,
		Scopes: c.Scopes, CodeChallenge: c.CodeChallenge, ExpiresAt: c.ExpiresAt, GrantID: binaryUUID(uuid.Nil()),
	})
	if err != nil {
		return insertError("create authorization code", err)
	}
	return nil
}

// AuthorizationCode implements store.Store.
func (s *Store) AuthorizationCode(ctx context.Context, h store.Hash) (store.AuthorizationCode, error) {
	var d codeDoc
	if err := s.codes.FindOne(ctx, bson.D{{Key: "_id", Value: h[:]}}).Decode(&d); err != nil {
		return store.AuthorizationCode{}, findError("authorization code", err)
	}
	if len(d.Hash) != len(store.Hash{}) {
		return store.AuthorizationCode{}, fmt.Errorf("mongodb: authorization code: a hash of %d bytes", len(d.Hash))
	}
	c := store.AuthorizationCode{
		Hash: store.Hash(d.Hash), ClientID: d.ClientID, RedirectURI: d.RedirectURI, Scopes: d.Scopes,
		CodeChallenge: d.CodeChallenge, ExpiresAt: d.ExpiresAt, Used: d.Used,
	}
	if err := parseUUIDs(uuidField{&c.UserID, d.UserID}, uuidField{&c.GrantID, d.GrantID}); err != nil {
		return store.AuthorizationCode{}, fmt.Errorf("mongodb: authorization code: %w", err)
	}
	return c, nil
}

// useCode marks the unused code with hash h used by grant, failing when it
// is unknown or used.
func (s *Store) useCode(ctx context.Context, h store.Hash, grant uuid.UUID) error {
	res, err := s.codes.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: h[:]}, {Key: "used", Value: false}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "used", Value: true}, {Key: "grant_id", Value: binaryUUID(grant)}}}})
	if err != nil {
		return fmt.Errorf("mongodb: use authorization code: %w", err)
	}
	if res.MatchedCount == 1 {
		return nil
	}
	n, err := s.codes.CountDocuments(ctx, bson.D{{Key: "_id", Value: h[:]}})
	if err != nil {
		return fmt.Errorf("mongodb: use authorization code: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mongodb: use authorization code: %w", apperr.ErrNotFound)
	}
	return fmt.Errorf("mongodb: use authorization code: %w", store.ErrUsed)
}

// SpendAuthorizationCode implements store.Store.
func (s *Store) SpendAuthorizationCode(ctx context.Context, h store.Hash) error {
	return s.useCode(ctx, h, uuid.Nil())
}

// RedeemAuthorizationCode implements store.Store.
func (s *Store) RedeemAuthorizationCode(ctx context.Context, h store.Hash, g store.Grant, first store.RefreshToken) error {
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.useCode(ctx, h, g.ID); err != nil {
			return err
		}
		return s.insertGrant(ctx, g, first)
	})
}
