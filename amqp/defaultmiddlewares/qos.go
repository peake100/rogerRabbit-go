package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// QoSMiddlewareID can be used to retrieve the running instance of QoSMiddleware during
// testing.
const QoSMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultQoS"

// QoSMiddleware saves most recent QoS settings and re-applies them on restart.
//
// This method is currently racy if multiple goroutines call QoS with different
// settings.
type QoSMiddleware struct {
	// qosArgs is the latest args passed to QoS().
	qosArgs amqpmiddleware.ArgsQoS
	// isSet is whether qosArgs has been set.
	isSet bool
}

// TypeID implements amqpmiddleware.ProvidesMiddleware and returns a static type ID for
// retrieving the active middleware value during testing.
func (middleware *QoSMiddleware) TypeID() amqpmiddleware.ProviderTypeID {
	return QoSMiddlewareID
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
func (middleware *QoSMiddleware) ChannelReconnect(
	next amqpmiddleware.HandlerChannelReconnect,
) amqpmiddleware.HandlerChannelReconnect {
	return func(ctx context.Context, args amqpmiddleware.ArgsChannelReconnect) (*streadway.Channel, error) {
		channel, err := next(ctx, args)
		// If there was an error or QoS() has not been called, return results.
		if err != nil || !middleware.isSet {
			return channel, err
		}

		// Otherwise re-apply the QoS settings.
		qosArgs := middleware.qosArgs
		err = channel.Qos(
			qosArgs.PrefetchCount,
			qosArgs.PrefetchSize,
			qosArgs.Global,
		)
		if err != nil {
			return nil, fmt.Errorf("error re-applying QoS args: %w", err)
		}
		return channel, nil
	}
}

// QoS is called in amqp.Channel.QoS(). Saves the QoS settings passed to the QoS
// function
func (middleware *QoSMiddleware) QoS(next amqpmiddleware.HandlerQoS) amqpmiddleware.HandlerQoS {
	return func(ctx context.Context, args amqpmiddleware.ArgsQoS) error {
		err := next(ctx, args)
		if err != nil {
			return err
		}

		middleware.qosArgs = args
		middleware.isSet = true
		return nil
	}
}

// NewQosMiddleware creates a new QoSMiddleware.
func NewQosMiddleware() amqpmiddleware.ProvidesMiddleware {
	return &QoSMiddleware{
		qosArgs: amqpmiddleware.ArgsQoS{},
		isSet:   false,
	}
}
