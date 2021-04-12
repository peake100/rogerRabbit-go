package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
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

// Active returns whether flow is currently active. For testing.
func (middleware *FlowMiddleware) Active() bool {
	return middleware.active
}

// Reconnect sets amqp.Channel.Flow(flow=false) on the underlying channel as soon as a
// reconnection occurs if the user has paused the flow on the channel.
func (middleware *FlowMiddleware) Reconnect(
	next amqpmiddleware.HandlerReconnect,
) (handler amqpmiddleware.HandlerReconnect) {
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

// Flow captures calls to *amqp.Channel.Flow() so channels can be paused on reconnect if
// the user has paused the channel.
func (middleware *FlowMiddleware) Flow(
	next amqpmiddleware.HandlerFlow,
) (handler amqpmiddleware.HandlerFlow) {
	return func(args amqpmiddleware.ArgsFlow) error {
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

// NewFlowMiddleware creates a new FlowMiddleware for a channel.
func NewFlowMiddleware() *FlowMiddleware {
	return &FlowMiddleware{
		active:     true,
		activeLock: new(sync.Mutex),
	}
}
