package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// QoSMiddleware saves most recent QoS settings and re-applies them on restart.
//
// This method is currently racy if multiple goroutines call QoS with different
// settings.
type QoSMiddleware struct {
	// qosArgs is the latest args passed to QoS()
	qosArgs amqpmiddleware.ArgsQoS
	// isSet is whether qosArgs has been set.
	isSet bool
}

// QosArgs returns current args. For testing.
func (middleware *QoSMiddleware) QosArgs() amqpmiddleware.ArgsQoS {
	return middleware.qosArgs
}

// IsSet returns whether the QoS has been set. For testing.
func (middleware *QoSMiddleware) IsSet() bool {
	return middleware.isSet
}

// Reconnect is called whenever the underlying channel is reconnected. This middleware
// re-applies any QoS calls to the channel.
func (middleware *QoSMiddleware) Reconnect(
	next amqpmiddleware.HandlerReconnect,
) (handler amqpmiddleware.HandlerReconnect) {
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

// Qos is called in amqp.Channel.Qos(). Saves the QoS settings passed to the QoS
// function
func (middleware *QoSMiddleware) Qos(
	next amqpmiddleware.HandlerQoS,
) (handler amqpmiddleware.HandlerQoS) {
	return func(args amqpmiddleware.ArgsQoS) error {
		err := next(args)
		if err != nil {
			return err
		}

		middleware.qosArgs = args
		middleware.isSet = true
		return nil
	}
}

// NewQosMiddleware creates a new QoSMiddleware.
func NewQosMiddleware() *QoSMiddleware {
	return &QoSMiddleware{
		qosArgs: amqpmiddleware.ArgsQoS{},
		isSet:   false,
	}
}
