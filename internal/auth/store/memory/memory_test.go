package memory_test

import (
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/auth/store/memory"
	"github.com/vardius/go-api-boilerplate/internal/auth/store/storetest"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
)

func TestStore(t *testing.T) {
	storetest.Run(t, func(*testing.T) store.Store { return memory.New(memdb.New()) })
}
