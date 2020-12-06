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
type QoSMiddleware struct {
	// The latest args passed to QoS()
	qosArgs amqpMiddleware.ArgsQoS
	// Whether qosArgs has been isSet.
	isSet bool
}

// Get current args. For testing.
func (middleware *QoSMiddleware) QosArgs() amqpMiddleware.ArgsQoS {
	return middleware.qosArgs
}

// Whether the QoS has been set. For testing.
func (middleware *QoSMiddleware) IsSet() bool {
	return middleware.isSet
}

// Re-applies QoS settings on reconnect
func (middleware *QoSMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	return func(
		ctx context.Context,
		logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := next(ctx, logger)
		// If there was an error or QoS() has not been called, return results.
		if err != nil || !middleware.isSet {
			return channel, err
		}

		// Otherwise re-apply the QoS settings.
		args := middleware.qosArgs
		err = channel.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
		if err != nil {
			return nil, fmt.Errorf("error re-applying QoS args: %w", err)
		}
		return channel, nil
	}
}

// Saves the QoS settings passed to the QoS function
func (middleware *QoSMiddleware) Qos(
	next amqpMiddleware.HandlerQoS,
) (handler amqpMiddleware.HandlerQoS) {
	return func(args *amqpMiddleware.ArgsQoS) error {
		err := next(args)
		if err != nil {
			return err
		}

		middleware.qosArgs = *args
		middleware.isSet = true
		return nil
	}
}

func NewQosMiddleware() *QoSMiddleware {
	return &QoSMiddleware{
		qosArgs: amqpMiddleware.ArgsQoS{},
		isSet:   false,
	}
}
