/*
Package grpc provides user grpc client
*/
package grpc

import (
	"context"
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/user/interfaces/proto"
	"google.golang.org/grpc"
)

// UserClient interface
type UserClient interface {
	DispatchAndClose(ctx context.Context, command string, payload []byte) error
}

type userClient struct {
	host string
	port int
}

// DispatchAndClose dials user domain server and dispatches command
// then closes connection
func (c *userClient) DispatchAndClose(ctx context.Context, command string, payload []byte) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", c.host, c.port), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := proto.NewDomainClient(conn)
	_, err = client.DispatchCommand(ctx, &proto.DispatchCommandRequest{
		Name:    command,
		Payload: payload,
	})

	return err
}

// New creates new user client
func New(host string, port int) UserClient {
	return &userClient{host, port}
}
