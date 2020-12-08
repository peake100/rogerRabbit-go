package defaultMiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Saves most recent QoS settings and re-applies them on restart.
//
// This method is currently racy if multiple goroutines call QoS with different
// settings.
type ConfirmsMiddleware struct {
	// Whether qosArgs has been isSet.
	confirmsOn bool
}

// Whether the QoS has been set. For testing.
func (middleware *ConfirmsMiddleware) ConfirmsOn() bool {
	return middleware.confirmsOn
}

// Re-applies QoS settings on reconnect
func (middleware *ConfirmsMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	return func(
		ctx context.Context,
		logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := next(ctx, logger)
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

// Saves the QoS settings passed to the QoS function
func (middleware *ConfirmsMiddleware) Confirm(
	next amqpMiddleware.HandlerConfirm,
) (handler amqpMiddleware.HandlerConfirm) {
	return func(args *amqpMiddleware.ArgsConfirms) error {
		err := next(args)
		if err != nil {
			return err
		}

		// If this method is called, turned confirms to true
		middleware.confirmsOn = true
		return nil
	}
}

func NewConfirmMiddleware() *ConfirmsMiddleware {
	return &ConfirmsMiddleware{
		confirmsOn: false,
	}
}
