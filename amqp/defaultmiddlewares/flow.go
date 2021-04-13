package defaultmiddlewares

import (
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
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
) (handler amqpmiddleware.HandlerChannelReconnect) {
	return func(args amqpmiddleware.ArgsChannelReconnect) (*streadway.Channel, error) {
		channel, err := next(args)
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
func NewFlowMiddleware() amqpmiddleware.ProvidesMiddleware {
	return &FlowMiddleware{
		active:     true,
		activeLock: new(sync.Mutex),
	}
}
