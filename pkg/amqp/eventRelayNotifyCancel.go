package amqp

import (
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// notifyCancelRelay implements eventRelay for Channel.NotifyCancel.
type notifyCancelRelay struct {
	// The channel we are relaying returns to from the broker
	CallerCancellations chan<- string

	// The current broker channel we are pulling from.
	brokerCancellations <-chan string

	// handler is the base handler wrapped in all caller-supplied middleware to be
	// evoked on each event.
	handler amqpmiddleware.HandlerNotifyCancelEvents
}

func (relay *notifyCancelRelay) baseHandler() amqpmiddleware.HandlerNotifyCancelEvents {
	return func(_ amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyCancel) {
		relay.CallerCancellations <- event.Cancellation
	}
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyCancel
func (relay *notifyCancelRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerChannel := make(chan string, cap(relay.CallerCancellations))
	relay.brokerCancellations = brokerChannel
	newChannel.NotifyCancel(brokerChannel)
	return nil
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyCancel
// events to the original caller.
func (relay *notifyCancelRelay) RunRelayLeg(legNum int) (done bool) {
	eventNum := int64(0)
	for thisCancellation := range relay.brokerCancellations {
		relay.handler(
			createEventMetadata(legNum, eventNum),
			amqpmiddleware.EventNotifyCancel{Cancellation: thisCancellation},
		)
		eventNum++
	}

	return false
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyCancelRelay) Shutdown() error {
	defer close(relay.CallerCancellations)
	return nil
}

// newNotifyReturnRelay creates a new notifyReturnRelay.
func newNotifyCancelRelay(
	callerCancellations chan<- string,
	middleware []amqpmiddleware.NotifyCancelEvents,
) *notifyCancelRelay {
	// Create the relay
	relay := &notifyCancelRelay{
		CallerCancellations: callerCancellations,
	}

	// Apply all middleware to the handler
	relay.handler = relay.baseHandler()
	for _, thisMiddleware := range middleware {
		relay.handler = thisMiddleware(relay.handler)
	}

	return relay
}
