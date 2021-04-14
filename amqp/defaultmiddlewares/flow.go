package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"sync"
)

// FlowMiddlewareID can be used to retrieve the running instance of FlowMiddleware
// during testing.
const FlowMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultFlow"

// FlowMiddleware tracks the Flow state of the channel, and if it is set to false by the
// user, re-established channels to flow=false.
type FlowMiddleware struct {
	// Whether flow is currently active
	active bool
	// Lock for active in cae multiple goroutines try to set the flow simultaneously.
	activeLock *sync.Mutex
}

// TypeID implements amqpmiddleware.ProvidesMiddleware and returns a static type ID for
// retrieving the active middleware value during testing.
func (middleware *FlowMiddleware) TypeID() amqpmiddleware.ProviderTypeID {
	return FlowMiddlewareID
}

// Active returns whether flow is currently active. For testing.
func (middleware *FlowMiddleware) Active() bool {
	return middleware.active
}

// Reconnect sets amqp.Channel.Flow(flow=false) on the underlying channel as soon as a
// reconnection occurs if the user has paused the flow on the channel.
func (middleware *FlowMiddleware) ChannelReconnect(
	next amqpmiddleware.HandlerChannelReconnect,
) amqpmiddleware.HandlerChannelReconnect {
	return func(
		ctx context.Context, args amqpmiddleware.ArgsChannelReconnect,
	) (amqpmiddleware.ResultsChannelReconnect, error) {
		results, err := next(ctx, args)
		// New channels start out active, so if flow is active we can keep chugging.
		if err != nil || middleware.active {
			return results, err
		}

		err = results.Channel.Flow(middleware.active)
		if err != nil {
			return results, fmt.Errorf("error setting flow to false: %w", err)
		}

		return results, err
	}
}

// Flow captures calls to *amqp.Channel.Flow() so channels can be paused on reconnect if
// the user has paused the channel.
func (middleware *FlowMiddleware) Flow(next amqpmiddleware.HandlerFlow) amqpmiddleware.HandlerFlow {
	return func(ctx context.Context, args amqpmiddleware.ArgsFlow) error {
		middleware.activeLock.Lock()
		defer middleware.activeLock.Unlock()

		err := next(ctx, args)
		if err != nil {
			return err
		}

		middleware.active = args.Active
		return nil
	}
}

// NewFlowMiddleware creates a new FlowMiddleware for a channel.
func NewFlowMiddleware() amqpmiddleware.ProvidesMiddleware {
	return &FlowMiddleware{
		active:     true,
		activeLock: new(sync.Mutex),
	}
}
