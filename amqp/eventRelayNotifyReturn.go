package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// notifyPublishRelay implements eventRelay for Channel.NotifyReturn.
type notifyReturnRelay struct {
	// The channel we are relaying returns to from the broker
	CallerReturns chan<- Return

	// The current broker channel we are pulling from.
	brokerReturns <-chan Return

	// handler is the base handler wrapped in all caller-supplied middleware to be
	// evoked on each event.
	handler amqpmiddleware.HandlerNotifyReturnEvents

	// Logger
	logger zerolog.Logger
}

func (relay *notifyReturnRelay) baseHandler() amqpmiddleware.HandlerNotifyReturnEvents {
	return func(event amqpmiddleware.EventNotifyReturn) {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Str("EXCHANGE", event.Return.Exchange).
				Str("ROUTING_KEY", event.Return.MessageId).
				Str("MESSAGE_ID", event.Return.MessageId).
				Bytes("BODY", event.Return.Body).
				Msg("return notification sent")
		}

		relay.CallerReturns <- event.Return
	}
}

// Logger implements eventRelay and sets up our logger.
func (relay *notifyReturnRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_RETURN").Logger()
	relay.logger = logger
	return relay.logger
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyReturn.
func (relay *notifyReturnRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerChannel := make(chan Return, cap(relay.CallerReturns))
	relay.brokerReturns = brokerChannel
	newChannel.NotifyReturn(brokerChannel)
	return nil
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyReturn
// events to the original caller.
func (relay *notifyReturnRelay) RunRelayLeg() (done bool, err error) {
	for thisReturn := range relay.brokerReturns {
		relay.handler(amqpmiddleware.EventNotifyReturn{Return: thisReturn})
	}

	return false, nil
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyReturnRelay) Shutdown() error {
	defer close(relay.CallerReturns)
	return nil
}

// newNotifyReturnRelay creates a new notifyReturnRelay.
func newNotifyReturnRelay(
	callerReturns chan<- Return,
	middleware []amqpmiddleware.NotifyReturnEvents,
) *notifyReturnRelay {
	// Create the relay
	relay := &notifyReturnRelay{
		CallerReturns: callerReturns,
	}

	// Apply all middleware to the handler
	relay.handler = relay.baseHandler()
	for _, thisMiddleware := range middleware {
		relay.handler = thisMiddleware(relay.handler)
	}

	return relay
}
