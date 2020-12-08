package defaultMiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
)

// FlowMiddleware tracks the Flow state of the channel, and if it is set to false by the
// user, re-established channels to flow=false.
type FlowMiddleware struct {
	// Whether flow is currently active
	active bool
	// Lock for active in cae multiple goroutines try to set the flow simultaneously.
	activeLock *sync.Mutex
}

func (middleware *FlowMiddleware) Active() bool {
	return middleware.active
}

func (middleware *FlowMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	return func(
		ctx context.Context, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := next(ctx, logger)
		// New channels start out active, so if flow is active we can keep chugging.
		if err != nil || middleware.active {
			return channel, err
		}

		err = channel.Flow(middleware.active)
		if err != nil {
			return nil, fmt.Errorf("error setting flow to false: %w", err)
		}

		return channel, err
	}
}

func (middleware *FlowMiddleware) Flow(
	next amqpMiddleware.HandlerFlow,
) (handler amqpMiddleware.HandlerFlow) {
	return func(args *amqpMiddleware.ArgsFlow) error {
		middleware.activeLock.Lock()
		defer middleware.activeLock.Unlock()

		err := next(args)
		if err != nil {
			return err
		}

		middleware.active = args.Active
		return nil
	}
}

func NewFlowMiddleware() *FlowMiddleware {
	return &FlowMiddleware{
		active:     true,
		activeLock: new(sync.Mutex),
	}
}
