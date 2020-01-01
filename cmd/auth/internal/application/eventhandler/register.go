package eventhandler

import (
	"context"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/eventbus"
	grpc_utils "github.com/vardius/go-api-boilerplate/internal/grpc"
	"github.com/vardius/gollback"
	"google.golang.org/grpc"
)

// Register registers event handlers for topics
// will panic after timeout if unable to register handlers
func Register(conn *grpc.ClientConn, eventBus eventbus.EventBus, topicToHandlerMap map[string]eventbus.EventHandler, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	connName := "pubsub"

	// Will retry infinitely until timeouts by context (after 5 seconds)
	_, err := gollback.New(ctx).Retry(0, func(ctx context.Context) (interface{}, error) {
		if !grpc_utils.IsConnectionServing(connName, conn) {
			return nil, errors.Newf(" %s gRPC connection is not serving", connName)
		}
    
		for topic, handler := range topicToHandlerMap {
			// Will resubscribe to handler on error infinitely
			go func(topic string, handler eventbus.EventHandler) {
				gollback.New(context.Background()).Retry(0, func(ctx context.Context) (interface{}, error) {
					err := eventBus.Subscribe(ctx, topic, handler)

					return nil, errors.Newf(errors.INTERNAL, "EventHandler %s unsubscribed (%v)", topic, err)
				})
			}(topic, handler)
		}

		return nil, nil
	})

		return nil, errors.Newf(" %s gRPC connection is not serving", connName)
	})

	if err != nil {
		panic(err)
	}
}
