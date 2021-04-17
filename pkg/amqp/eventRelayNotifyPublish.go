package amqp

import (
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/internal"
	streadway "github.com/streadway/amqp"
)

// notifyPublishRelay implements eventRelay for Channel.NotifyPublish.
type notifyPublishRelay struct {
	// CallerConfirmations is the Delivery channel to pass deliveries back to the
	// client.
	CallerConfirmations chan<- internal.Confirmation

	// brokerConfirmations is the current delivery channel coming from the broker.
	brokerConfirmations <-chan streadway.Confirmation

	// handler for notify publish events.
	handler amqpmiddleware.HandlerNotifyPublishEvents
}

// baseHandler creates the innermost handler for the event send.
func (relay *notifyPublishRelay) baseHandler() (
	handler amqpmiddleware.HandlerNotifyPublishEvents,
) {
	return func(_ amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyPublish) {
		relay.CallerConfirmations <- event.Confirmation
	}
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

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyPublish
// events to the original caller.
func (relay *notifyPublishRelay) RunRelayLeg(legNum int) (done bool) {
	eventNum := int64(0)
	// Range over the confirmations from the broker.
	for brokerConf := range relay.brokerConfirmations {
		// Apply the offset to the delivery tag.
		confirmation := internal.Confirmation{
			Confirmation:     brokerConf,
			DisconnectOrphan: false,
		}
		relay.handler(
			createEventMetadata(legNum, eventNum),
			amqpmiddleware.EventNotifyPublish{Confirmation: confirmation},
		)
		eventNum++
	}

	// Otherwise continue to the next channel.
	return false
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyPublishRelay) Shutdown() error {
	close(relay.CallerConfirmations)
	return nil
}

// newNotifyPublishRelay creates a new notifyPublishRelay.
func newNotifyPublishRelay(
	callerConfirmations chan<- internal.Confirmation,
	middleware []amqpmiddleware.NotifyPublishEvents,
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
