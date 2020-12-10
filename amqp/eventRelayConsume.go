package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/dataModels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Holds args for consume operation
type consumeArgs struct {
	queue, consumer                     string
	autoAck, exclusive, noLocal, noWait bool
	args                                Table
	callerDeliveryChan                  chan dataModels.Delivery
}

// Relays Deliveries across channel disconnects.
type consumeRelay struct {
	// Arguments to call on Consume
	ConsumeArgs consumeArgs
	// Delivery channel to pass deliveries back to the client.
	CallerDeliveries chan<- dataModels.Delivery

	Acknowledger streadway.Acknowledger

	// The current delivery channel coming from the broker.
	brokerDeliveries <-chan streadway.Delivery

	// The handler with middleware we will call to pass events to the caller.
	handler amqpMiddleware.HandlerConsumeEvent
}

func (relay *consumeRelay) baseHandler() amqpMiddleware.HandlerConsumeEvent {
	return func(event dataModels.Delivery) {
		relay.CallerDeliveries <- event
	}
}

func (relay *consumeRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	return parent.With().
		Str("EVENT_TYPE", "CONSUME").
		Str("CONSUMER_QUEUE", relay.ConsumeArgs.queue).
		Logger()
}

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

func (relay *consumeRelay) RunRelayLeg() (done bool, err error) {
	// Drain consumer events
	for brokerDelivery := range relay.brokerDeliveries {
		// Wrap the delivery and send on our way.
		relay.handler(dataModels.NewDelivery(brokerDelivery, relay.Acknowledger))
	}

	return false, nil
}

func (relay *consumeRelay) Shutdown() error {
	close(relay.CallerDeliveries)
	return nil
}

func newConsumeRelay(
	consumeArgs *consumeArgs,
	acknowledger Acknowledger,
	middleware []amqpMiddleware.ConsumeEvent,
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
