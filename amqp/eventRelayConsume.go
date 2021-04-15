package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	streadway "github.com/streadway/amqp"
)

// consumeArgs holds args for consume operation
type consumeArgs struct {
	queue, consumer                     string
	autoAck, exclusive, noLocal, noWait bool
	args                                Table
	callerDeliveryChan                  chan datamodels.Delivery
}

// consumeRelay implements eventRelay for Channel.Consume.
type consumeRelay struct {
	// ConsumeArgs are arguments to call on Channel.Consume
	ConsumeArgs consumeArgs
	// CallerDeliveries is the delivery channel to pass deliveries back to the client.
	CallerDeliveries chan<- datamodels.Delivery

	// Acknowledger is the robust Channel this relay is sending events for.
	Acknowledger streadway.Acknowledger

	// brokerDeliveries is the current delivery channel coming from the broker.
	brokerDeliveries <-chan streadway.Delivery

	// handler with middleware we will call to pass events to the caller.
	handler amqpmiddleware.HandlerConsumeEvents
}

// baseHandler returns the base handler to be wrapped in middleware.
func (relay *consumeRelay) baseHandler() amqpmiddleware.HandlerConsumeEvents {
	return func(_ amqpmiddleware.EventMetadata, event amqpmiddleware.EventConsume) {
		relay.CallerDeliveries <- event.Delivery
	}
}

// SetupForRelayLeg implements eventRelay and sets up a new broker Consume channel to
// relay events from.
func (relay *consumeRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerDeliveries, err := newChannel.Consume(
		relay.ConsumeArgs.queue,
		relay.ConsumeArgs.consumer,
		relay.ConsumeArgs.autoAck,
		relay.ConsumeArgs.exclusive,
		relay.ConsumeArgs.noLocal,
		relay.ConsumeArgs.noWait,
		relay.ConsumeArgs.args,
	)

	if err != nil {
		return err
	}

	relay.brokerDeliveries = brokerDeliveries

	return nil
}

// RunRelayLeg implements eventRelay and relays all deliveries from the current
// underlying streadway/amqp.Channel to the caller.
func (relay *consumeRelay) RunRelayLeg(legNum int) (done bool) {
	// Drain consumer events
	eventNum := int64(0)
	for brokerDelivery := range relay.brokerDeliveries {
		// Wrap the delivery and send on our way.
		relay.handler(
			createEventMetadata(legNum, eventNum),
			amqpmiddleware.EventConsume{
				Delivery: datamodels.NewDelivery(brokerDelivery, relay.Acknowledger),
			},
		)
		eventNum++
	}

	return false
}

// Shutdown implements eventRelay and closes the caller channel.
func (relay *consumeRelay) Shutdown() error {
	close(relay.CallerDeliveries)
	return nil
}

// newConsumeRelay creates a new consumeRelay to relay Channel.Consume events to the
// caller.
func newConsumeRelay(
	consumeArgs *consumeArgs,
	acknowledger Acknowledger,
	middleware []amqpmiddleware.ConsumeEvents,
) *consumeRelay {
	relay := &consumeRelay{
		ConsumeArgs:      *consumeArgs,
		CallerDeliveries: consumeArgs.callerDeliveryChan,
		Acknowledger:     acknowledger,
		brokerDeliveries: nil,
		handler:          nil,
	}

	relay.handler = relay.baseHandler()
	for _, thisMiddleware := range middleware {
		relay.handler = thisMiddleware(relay.handler)
	}

	return relay
}
