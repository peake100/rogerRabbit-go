package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Relays Deliveries across channel disconnects.
type notifyPublishRelay struct {
	// Delivery channel to pass deliveries back to the client.
	CallerConfirmations chan<- data.Confirmation

	// The current delivery channel coming from the broker.
	brokerConfirmations <-chan streadway.Confirmation

	// Middleware for notify publish.
	middleware []amqpMiddleware.NotifyPublishEvent
	// Handler for notify publish events.
	handler amqpMiddleware.HandlerNotifyPublishEvent

	// Logger
	logger zerolog.Logger
}

// Creates the innermost handler for the event send
func (relay *notifyPublishRelay) baseHandler() (
	handler amqpMiddleware.HandlerNotifyPublishEvent,
) {
	return func(event *amqpMiddleware.EventNotifyPublish) {
		relay.CallerConfirmations <- event.Confirmation
	}
}

func (relay *notifyPublishRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_PUBLISH").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyPublishRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	// Get our broker channel. We will make it with the same capacity of the channel the
	// caller sent into it.
	brokerConfirmations := make(
		chan streadway.Confirmation, cap(relay.CallerConfirmations),
	)
	relay.brokerConfirmations = newChannel.NotifyPublish(brokerConfirmations)

	return nil
}

func (relay *notifyPublishRelay) logConfirmation(confirmation data.Confirmation) {
	relay.logger.Debug().
		Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
		Bool("ACK", confirmation.Ack).
		Bool("ORPHAN", confirmation.DisconnectOrphan).
		Msg("publish confirmation event sent")
}

func (relay *notifyPublishRelay) RunRelayLeg() (done bool, err error) {
	// Range over the confirmations from the broker.
	for brokerConf := range relay.brokerConfirmations {
		// Apply the offset to the delivery tag.
		confirmation := data.Confirmation{
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

func (relay *notifyPublishRelay) Shutdown() error {
	close(relay.CallerConfirmations)
	return nil
}

func newNotifyPublishRelay(
	callerConfirmations chan<- data.Confirmation,
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
