package amqpconsumer_test

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqptest"
	"github.com/peake100/rogerRabbit-go/roger/amqpconsumer"
	"github.com/peake100/rogerRabbit-go/roger/amqpconsumer/middleware"
)

type BasicProcessor struct {
}

// ConsumeArgs returns the args to be made to the consumer's internal
// Channel.Consume() method.
func (processor *BasicProcessor) AmqpArgs() amqpconsumer.AmqpArgs {
	return amqpconsumer.AmqpArgs{
		ConsumerName: "example_consumer_queue",
		AutoAck:      false,
		Exclusive:    false,
		Args:         nil,
	}
}

// SetupChannel is called before consuming begins, and allows the handler to declare
// any routes, bindings, etc, necessary to handle it's route.
func (processor *BasicProcessor) SetupChannel(
	ctx context.Context, amqpChannel middleware.AmqpRouteManager,
) error {
	_, err := amqpChannel.QueueDeclare(
		"example_consumer_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error declaring Queue: %w", err)
	}

	return nil
}

// HandleDelivery is the business logic invoked for each delivery.
func (processor *BasicProcessor) HandleDelivery(
	ctx context.Context, delivery amqp.Delivery,
) (err error, requeue bool) {
	// Print the message
	fmt.Println("BODY:", delivery.Body)

	// Returning no error will result in an ACK of the message.
	return nil, false
}

// Cleanup allows the route handler to remove any resources necessary on close.
func (processor *BasicProcessor) CleanupChannel(
	ctx context.Context, amqpChannel middleware.AmqpRouteManager,
) error {
	_, err := amqpChannel.QueueDelete(
		"example_consumer_queue", false, false, false,
	)
	if err != nil {
		return fmt.Errorf("error deleting Queue: %w", err)
	}

	return nil
}

func ExampleConsumer() {
	// Get a new connection to our test broker.
	connection, err := amqp.Dial(amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Get a new channel from our robust connection.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Create a new consumer that uses our robust channel.
	consumer := amqpconsumer.New(channel, amqpconsumer.DefaultOpts())
	defer consumer.StartShutdown()

	// Create a new delivery processor and register it.
	processor := new(BasicProcessor)
	err = consumer.RegisterProcessor(processor)
	if err != nil {
		panic(err)
	}

	// This method will block forever as the consumer runs.
	err = consumer.Run()
	if err != nil {
		panic(err)
	}
}
