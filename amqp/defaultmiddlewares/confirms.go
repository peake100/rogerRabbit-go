package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// ConfirmsMiddlewareID can be used to retrieve the running instance of
// ConfirmsMiddleware during testing.
const ConfirmsMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultConfirms"

// ConfirmsMiddleware saves most recent amqp.Channel.Confirm() settings and re-applies
// them on restart.
type ConfirmsMiddleware struct {
	// Whether qosArgs has been isSet.
	confirmsOn bool
}

// TypeID implements amqpmiddleware.ProvidesMiddleware and returns a static type ID for
// retrieving the active middleware value during testing.
func (middleware *ConfirmsMiddleware) TypeID() amqpmiddleware.ProviderTypeID {
	return ConfirmsMiddlewareID
}

// ConfirmsOn returns whether Confirm() has been called on this channel. For testing.
func (middleware *ConfirmsMiddleware) ConfirmsOn() bool {
	return middleware.confirmsOn
}

// ChannelReconnect puts the new, underlying connection into confirmation mode if
// Confirm() has been called.
func (middleware *ConfirmsMiddleware) ChannelReconnect(
	next amqpmiddleware.HandlerChannelReconnect,
) (handler amqpmiddleware.HandlerChannelReconnect) {
	return func(ctx context.Context, args amqpmiddleware.ArgsChannelReconnect) (*streadway.Channel, error) {
		channel, err := next(ctx, args)
		// If there was an error or QoS() has not been called, return results.
		if err != nil || !middleware.confirmsOn {
			return channel, err
		}

		err = channel.Confirm(false)
		if err != nil {
			return nil, fmt.Errorf(
				"error setting channel to confirms mode: %w", err,
			)
		}
		return channel, nil
	}
}

// Confirm captures called to amqp.Channel.Confirm() and remembers that all subsequent
// underlying channels should be put into confirmation mode upon reconnect.
func (middleware *ConfirmsMiddleware) Confirm(
	next amqpmiddleware.HandlerConfirm,
) (handler amqpmiddleware.HandlerConfirm) {
	return func(ctx context.Context, args amqpmiddleware.ArgsConfirms) error {
		err := next(ctx, args)
		if err != nil {
			return err
		}

		// If this method is called, turned confirms to true
		middleware.confirmsOn = true
		return nil
	}
}

// NewConfirmMiddleware creates a new *ConfirmsMiddleware to register with a channel.
func NewConfirmMiddleware() amqpmiddleware.ProvidesMiddleware {
	return &ConfirmsMiddleware{
		confirmsOn: false,
	}
}
