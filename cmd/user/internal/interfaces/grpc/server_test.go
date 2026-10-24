package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/cmd/user/proto"
)

func TestDispatchOverGRPCIsRejected(t *testing.T) {
	server := NewServer(memory.NewUserRepository())

	_, err := server.DispatchUserCommand(context.Background(), &proto.DispatchUserCommandRequest{
		Name:    user.ChangeUserEmailAddress,
		Payload: []byte(`{"id":"00000000-0000-0000-0000-00000000000a","email":"attacker@example.com"}`),
	})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("err = %v, want code %s", err, codes.Unimplemented)
	}
}
