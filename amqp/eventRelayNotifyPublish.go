package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/dataModels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// notifyPublishRelay implements eventRelay for Channel.NotifyPublish.
type notifyPublishRelay struct {
	// CallerConfirmations is the Delivery channel to pass deliveries back to the
	// client.
	CallerConfirmations chan<- dataModels.Confirmation

	// brokerConfirmations is the current delivery channel coming from the broker.
	brokerConfirmations <-chan streadway.Confirmation

	// middleware is Middleware for notify publish.
	middleware []amqpMiddleware.NotifyPublishEvent
	// handler for notify publish events.
	handler amqpMiddleware.HandlerNotifyPublishEvent

	// logger is the Logger for our event relay.
	logger zerolog.Logger
}

// baseHandler creates the innermost handler for the event send.
func (relay *notifyPublishRelay) baseHandler() (
	handler amqpMiddleware.HandlerNotifyPublishEvent,
) {
	return func(event *amqpMiddleware.EventNotifyPublish) {
		relay.CallerConfirmations <- event.Confirmation
	}
}

// Logger implements eventRelay and sets up our logger.
func (relay *notifyPublishRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_PUBLISH").Logger()
	relay.logger = logger
	return relay.logger
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyPublish.
func (relay *notifyPublishRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	// Get our broker channel. We will make it with the same capacity of the channel the
	// caller sent into it.
	brokerConfirmations := make(
		chan streadway.Confirmation, cap(relay.CallerConfirmations),
	)
	relay.brokerConfirmations = newChannel.NotifyPublish(brokerConfirmations)

	return nil
}

// logConfirmation logs a confirmation message in debug mode.
func (relay *notifyPublishRelay) logConfirmation(confirmation dataModels.Confirmation) {
	relay.logger.Debug().
		Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
		Bool("ACK", confirmation.Ack).
		Bool("ORPHAN", confirmation.DisconnectOrphan).
		Msg("publish confirmation event sent")
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyPublish
// events to the original caller.
func (relay *notifyPublishRelay) RunRelayLeg() (done bool, err error) {
	// Range over the confirmations from the broker.
	for brokerConf := range relay.brokerConfirmations {
		// Apply the offset to the delivery tag.
		confirmation := dataModels.Confirmation{
			Confirmation:     brokerConf,
			DisconnectOrphan: false,
		}
		if relay.logger.Debug().Enabled() {
			relay.logConfirmation(confirmation)
		}
		relay.handler(&amqpMiddleware.EventNotifyPublish{Confirmation: confirmation})
	}

	// Otherwise continue to the next channel.
	return false, nil
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyPublishRelay) Shutdown() error {
	close(relay.CallerConfirmations)
	return nil
}

// newNotifyPublishRelay creates a new notifyPublishRelay.
func newNotifyPublishRelay(
	callerConfirmations chan<- dataModels.Confirmation,
	middleware []amqpMiddleware.NotifyPublishEvent,
) *notifyPublishRelay {
	// Create the relay
	relay := &notifyPublishRelay{
		CallerConfirmations: callerConfirmations,
	}

	// Apply all middleware to the handler
	relay.handler = relay.baseHandler()
	for _, thisMiddleware := range middleware {
		relay.handler = thisMiddleware(relay.handler)
	}

	return relay
}
