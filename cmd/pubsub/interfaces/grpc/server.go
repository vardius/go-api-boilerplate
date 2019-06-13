/*
Package grpc provides grpc messagebus server
*/
package grpc

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
	pubsub_messagebus "github.com/vardius/go-api-boilerplate/cmd/pubsub/application/messagebus"
	"github.com/vardius/go-api-boilerplate/cmd/pubsub/infrastructure/proto"
)

type server struct {
	bus pubsub_messagebus.MessageBus
}

// NewServer returns new messagebus server object
func NewServer(bus pubsub_messagebus.MessageBus) proto.MessageBusServer {
	return &server{bus}
}

// Publish publishes message payload to all topic handlers
func (s *server) Publish(ctx context.Context, r *proto.PublishRequest) (*empty.Empty, error) {
	log.Printf("[grpc|Publish] %s %s", r.GetTopic(), r.GetPayload())

	s.bus.Publish(r.GetTopic(), ctx, r.GetPayload())

	return new(empty.Empty), ctx.Err()
}

// Subscribe subscribes to a topic
func (s *server) Subscribe(r *proto.SubscribeRequest, stream proto.MessageBus_SubscribeServer) error {
	done := make(chan error)
	defer close(done)

	handler := func(_ context.Context, payload pubsub_messagebus.Payload) {
		log.Printf("[grpc|Subscribe] %s %s", r.GetTopic(), payload)

		err := stream.Send(&proto.SubscribeResponse{
			Payload: payload,
		})

		if err != nil {
			done <- err
		}
	}

	s.bus.Subscribe(r.GetTopic(), handler)

	err := <-done

	s.bus.Unsubscribe(r.GetTopic(), handler)

	return err
}
