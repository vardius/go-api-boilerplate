package memory_test

import (
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/user/store/memory"
	"github.com/vardius/go-api-boilerplate/internal/user/store/storetest"
)

func TestStores(t *testing.T) {
	storetest.Test(t, func(*testing.T) storetest.Backend {
		db := memdb.New()
		return storetest.Backend{
			UnitOfWork: db,
			Events:     memdb.NewEventStore(db),
			Users:      memory.NewUsers(db),
			LoginCodes: memory.NewLoginCodes(db),
		}
	})
}
