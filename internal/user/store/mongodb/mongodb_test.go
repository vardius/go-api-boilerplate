package mongodb_test

import (
	"testing"

	platform "github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb/mongotest"
	"github.com/vardius/go-api-boilerplate/internal/user/store/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/user/store/storetest"
)

func TestStores(t *testing.T) {
	storetest.Test(t, func(t *testing.T) storetest.Backend {
		db := mongotest.New(t)
		if err := mongodb.EnsureIndexes(t.Context(), db); err != nil {
			t.Fatal(err)
		}
		return storetest.Backend{
			UnitOfWork: db,
			Events:     platform.NewEventStore(db),
			Users:      mongodb.NewUsers(db),
			LoginCodes: mongodb.NewLoginCodes(db),
		}
	})
}

func TestEnsureIndexesTwiceChangesNothing(t *testing.T) {
	db := mongotest.New(t)
	for i := range 2 {
		if err := mongodb.EnsureIndexes(t.Context(), db); err != nil {
			t.Fatalf("ensure %d: %v", i+1, err)
		}
	}
}
