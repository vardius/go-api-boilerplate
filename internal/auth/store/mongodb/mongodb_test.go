package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/auth/store/storetest"
	platform "github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb/mongotest"
)

func indexed(t *testing.T) *platform.DB {
	t.Helper()
	db := mongotest.New(t)
	if err := EnsureIndexes(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return New(indexed(t)) })
}

// The TTL monitor runs once a minute, too slowly for a test to wait for, so
// this checks the indexes that drive it.
func TestExpiredDocumentsHaveTTLIndexes(t *testing.T) {
	db := indexed(t)
	if err := EnsureIndexes(t.Context(), db); err != nil {
		t.Fatalf("ensuring the indexes again: %v", err)
	}
	want := map[string]int64{
		grantsCollection:   0,
		tokensCollection:   0,
		requestsCollection: 0,
		codesCollection:    3600,
	}
	for coll, after := range want {
		cur, err := db.Database.Collection(coll).Indexes().List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var specs []struct {
			Key                bson.D `bson:"key"`
			ExpireAfterSeconds *int64 `bson:"expireAfterSeconds"`
		}
		if err := cur.All(t.Context(), &specs); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, spec := range specs {
			if len(spec.Key) != 1 || spec.Key[0].Key != "expires_at" {
				continue
			}
			if spec.ExpireAfterSeconds == nil || *spec.ExpireAfterSeconds != after {
				t.Errorf("%s: the expires_at index expires after %v seconds, want %d", coll, spec.ExpireAfterSeconds, after)
			}
			found = true
		}
		if !found {
			t.Errorf("%s has no TTL index on expires_at: %v", coll, specs)
		}
	}
}
