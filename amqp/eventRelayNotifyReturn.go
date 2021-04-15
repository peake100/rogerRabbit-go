package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
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
}

func (relay *notifyReturnRelay) baseHandler() amqpmiddleware.HandlerNotifyReturnEvents {
	return func(_ amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyReturn) {
		relay.CallerReturns <- event.Return
	}
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyReturn.
func (relay *notifyReturnRelay) SetupForRelayLeg(newChannel *streadway.Channel) {
	brokerChannel := make(chan Return, cap(relay.CallerReturns))
	relay.brokerReturns = brokerChannel
	newChannel.NotifyReturn(brokerChannel)
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyReturn
// events to the original caller.
func (relay *notifyReturnRelay) RunRelayLeg(newChan *streadway.Channel, legNum int) (done bool) {
	relay.SetupForRelayLeg(newChan)

	eventNum := int64(0)
	for thisReturn := range relay.brokerReturns {
		relay.handler(
			createEventMetadata(legNum, eventNum),
			amqpmiddleware.EventNotifyReturn{Return: thisReturn},
		)
		eventNum++
	}

	return false
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
