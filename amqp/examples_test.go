package amqp_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqptest"
	streadway "github.com/streadway/amqp"
	"sync"
	"testing"
	"time"
)

// Channel channelReconnect examples.
func ExampleChannel_reconnect() {
	// Get a new connection to our test broker.
	//
	// DialCtx is a new function that allows the Dial function to keep attempting
	// re-dials to the broker until the passed context expires.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}

	// Get a new channel from our robust connection.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// We can use the test method to return an testing object with some additional
	// methods. ForceReconnect force-closes the underlying livesOnce, causing the
	// robust connection to channelReconnect.
	//
	// We'll use a dummy *testing.T object here. These methods are designed for tests
	// only and should not be used in production code.
	channel.Test(new(testing.T)).ForceReconnect(context.Background())

	// We can see here our channel is still open.
	fmt.Println("IS CLOSED:", channel.IsClosed())

	// We can even declare a queue on it
	queue, err := channel.QueueDeclare(
		"example_channel_reconnect", // name
		false,                       // durable
		true,                        // autoDelete
		false,                       // exclusive
		false,                       // noWait
		nil,                         // args
	)
	if err != nil {
		panic(err)
	}

	// Here is the result
	fmt.Printf("QUEUE    : %+v\n", queue)

	// Explicitly close the connection. This will also close all child channels.
	err = connection.Close()
	if err != nil {
		panic(err)
	}

	// Now that we have explicitly closed the connection, the channel will be closed.
	fmt.Println("IS CLOSED:", channel.IsClosed())

	// IS CLOSED: false
	// QUEUE    : {Name:example_channel_reconnect Messages:0 Consumers:0}
	// IS CLOSED: true
}

func ExampleChannel_Consume_deliveryTags() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}

	// Get a new channel from our robust connection for consuming.
	consumeChannel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Get a new channel from our robust connection for publishing.
	publishChannel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	queueName := "example_delivery_tag_continuity"

	// Declare the queue we are going to use.
	queue, err := consumeChannel.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		panic(err)
	}

	// Clean up the queue on exit,
	defer consumeChannel.QueueDelete(
		queue.Name, false, false, false,
	)

	// Start consuming the channel
	consume, err := consumeChannel.Consume(
		queue.Name,
		"example consumer", // consumer name
		true,               // autoAck
		false,              // exclusive
		false,              // no local
		false,              // no wait
		nil,                // args
	)

	// We'll close this channel when the consumer is exhausted
	consumeComplete := new(sync.WaitGroup)
	consumerClosed := make(chan struct{})

	// Set the prefetch count to 1, that way we are less likely to lose a message
	// that in in-flight from the broker in this example.
	err = consumeChannel.Qos(1, 0, false)
	if err != nil {
		panic(err)
	}

	// Launch a consumer
	go func() {
		// Close the consumeComplete to signal exit
		defer close(consumerClosed)

		fmt.Println("STARTING CONSUMER")

		// Range over the consume channel
		for delivery := range consume {
			// Force-channelReconnect the channel after each delivery.
			consumeChannel.Test(new(testing.T)).ForceReconnect(context.Background())

			// Tick down the consumeComplete WaitGroup
			consumeComplete.Done()

			// Print the delivery. Even though we are forcing a new underlying channel
			// to be connected each time, the delivery tags will still be continuous.
			fmt.Printf(
				"DELIVERY %v: %v\n", delivery.DeliveryTag, string(delivery.Body),
			)
		}

		fmt.Println("DELIVERIES EXHAUSTED")
	}()

	// We'll publish 10 test messages.
	for i := 0; i < 10; i++ {
		// Add one to the consumeComplete WaitGroup.
		consumeComplete.Add(1)

		// Publish a message. Even though the consumer may be force re-connecting the
		// connection each time, we can keep using the channel.
		//
		// NOTE: it is possible that we will drop a message here during a reconnection
		// event. If we want to be sure all messages reach the broker, we'll need to
		// publish messages with the Channel in confirmation mode, which we will
		// show in another example.
		err = publishChannel.Publish(
			"",
			queue.Name,
			false,
			false,
			amqp.Publishing{
				Body: []byte(fmt.Sprintf("message %v", i)),
			},
		)
		if err != nil {
			panic(err)
		}
	}

	// Wait for all messages to be received
	consumeComplete.Wait()

	// Close the connection
	err = connection.Close()
	if err != nil {
		panic(err)
	}

	// Wait for the consumer to exit
	<-consumerClosed

	// exit

	// STARTING CONSUMER
	// DELIVERY 1: message 0
	// DELIVERY 2: message 1
	// DELIVERY 3: message 2
	// DELIVERY 4: message 3
	// DELIVERY 5: message 4
	// DELIVERY 6: message 5
	// DELIVERY 7: message 6
	// DELIVERY 8: message 7
	// DELIVERY 9: message 8
	// DELIVERY 10: message 9
	// DELIVERIES EXHAUSTED
}

func ExampleChannel_Consume_orphan() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}

	// Get a new channel from our robust connection for consuming.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	queueName := "example_delivery_ack_orphan"

	// Declare the queue we are going to use.
	_, err = channel.QueueDeclare(
		queueName, // name
		false,     // durable
		true,      // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		panic(err)
	}

	// Cleanup channel on exit.
	defer channel.QueueDelete(queueName, false, false, false)

	// Start consuming the channel
	consume, err := channel.Consume(
		queueName,
		"example consumer", // consumer name
		// Auto-ack is set to false
		false, // autoAck
		false, // exclusive
		false, // no local
		false, // no wait
		nil,   // args
	)

	// publish a message
	err = channel.Publish(
		"", // exchange
		queueName,
		false,
		false,
		amqp.Publishing{
			Body: []byte("test message"),
		},
	)
	if err != nil {
		panic(err)
	}

	// get the delivery of our published message
	delivery := <-consume
	fmt.Println("DELIVERY:", string(delivery.Body))

	// Force-close the channel.
	channel.Test(new(testing.T)).ForceReconnect(context.Background())

	// Now that the original underlying channel is closed, it is impossible to ack
	// the delivery. We will get an error when we try.
	err = delivery.Ack(false)
	fmt.Println("ACK ERROR:", err)

	// This error is an orphan error
	var orphanErr amqp.ErrCantAcknowledgeOrphans
	if !errors.As(err, &orphanErr) {
		panic("error not orphan error")
	}

	fmt.Println("FIRST ORPHAN TAG:", orphanErr.OrphanTagFirst)
	fmt.Println("LAST ORPHAN TAG :", orphanErr.OrphanTagLast)

	// DELIVERY: test message
	// ACK ERROR: 1 tags orphaned (1 - 1), 0 tags successfully acknowledged
	// FIRST ORPHAN TAG: 1
	// LAST ORPHAN TAG : 1
}

// Publication tags remain continuous, even across disconnection events.
func ExampleChannel_Publish_tagContinuity() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}

	// Get a new channel from our robust connection for publishing.
	publishChannel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Put the channel into confirmation mode
	err = publishChannel.Confirm(false)
	if err != nil {
		panic(err)
	}

	confirmationsReceived := new(sync.WaitGroup)
	confirmationsComplete := make(chan struct{})

	// Create a channel to consume publication confirmations.
	publishEvents := publishChannel.NotifyPublish(make(chan amqp.Confirmation))
	go func() {
		// Close to signal exit.
		defer close(confirmationsComplete)

		// Range over the confirmation channel.
		for confirmation := range publishEvents {
			// Mark 1 confirmation as done.
			confirmationsReceived.Done()

			// Print confirmation.
			fmt.Printf(
				"CONFIRMATION TAG %02d: ACK: %v ORPHAN: %v\n",
				confirmation.DeliveryTag,
				confirmation.Ack,
				// If the confirmation was never received because the channel was
				// disconnected, then confirmation.Ack will be false, and
				// confirmation.DisconnectOrphan will be true.
				confirmation.DisconnectOrphan,
			)
		}
	}()

	// Declare the message queue
	queueName := "example_publish_tag_continuity"
	_, err = publishChannel.QueueDeclare(
		queueName,
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	// We'll publish 10 test messages.
	for i := 0; i < 10; i++ {
		// We want to wait here to make sure we got the confirmation from the last
		// publication before force-closing the connection to show we can handle it.
		confirmationsReceived.Wait()

		// Force a reconnection of the underlying channel.
		publishChannel.Test(new(testing.T)).ForceReconnect(context.Background())

		// Increment the confirmation WaitGroup
		confirmationsReceived.Add(1)

		// Publish a message. Even though the consumer may be force re-connecting the
		// connection each time, we can keep using the channel.
		err = publishChannel.Publish(
			"",
			queueName,
			false,
			false,
			amqp.Publishing{
				Body: []byte(fmt.Sprintf("message %v", i)),
			},
		)
		if err != nil {
			panic(err)
		}
	}

	// Wait for all confirmations to be received.
	confirmationsReceived.Wait()

	// Close the connection.
	err = connection.Close()
	if err != nil {
		panic(err)
	}

	// Wait for the confirmation routine to exit.
	<-confirmationsComplete

	// Exit.

	// CONFIRMATION TAG 01: ACK: true ORPHAN: false
	// CONFIRMATION TAG 02: ACK: true ORPHAN: false
	// CONFIRMATION TAG 03: ACK: true ORPHAN: false
	// CONFIRMATION TAG 04: ACK: true ORPHAN: false
	// CONFIRMATION TAG 05: ACK: true ORPHAN: false
	// CONFIRMATION TAG 06: ACK: true ORPHAN: false
	// CONFIRMATION TAG 07: ACK: true ORPHAN: false
	// CONFIRMATION TAG 08: ACK: true ORPHAN: false
	// CONFIRMATION TAG 09: ACK: true ORPHAN: false
	// CONFIRMATION TAG 10: ACK: true ORPHAN: false
}

// Declared queues are re-declared on disconnect.
func ExampleChannel_QueueDeclare_reDeclare() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}

	// Close the connection on exit.
	defer connection.Close()

	// Get a new channel from our robust connection for publishing.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	queueName := "example_queue_declare_robust"

	// If we try to inspect this queue before declaring it, we will get an error.
	_, err = channel.QueueInspect(queueName)
	if err == nil {
		panic("expected queue inspect error")
	}
	fmt.Println("INSPECT ERROR:", err)

	// Declare the queue.
	_, err = channel.QueueDeclare(
		queueName,
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)

	// Delete the queue to clean up
	defer channel.QueueDelete(queueName, false, false, false)

	// Inspect the queue.
	queue, err := channel.QueueInspect(queueName)
	if err != nil {
		panic(err)
	}
	fmt.Println("INSPECTION:", queue.Name)

	// Force a re-connection
	channel.Test(new(testing.T)).ForceReconnect(context.Background())

	// Inspect the queue again, it will already have been re-declared
	queue, err = channel.QueueInspect(queueName)
	if err != nil {
		panic(err)
	}
	fmt.Println("INSPECTION:", queue.Name)

	// Delete the queue to clean up
	_, err = channel.QueueDelete(queueName, false, false, false)
	if err != nil {
		panic(err)
	}

	// INSPECT ERROR: Exception (404) Reason: "NOT_FOUND - no queue 'example_queue_declare_robust' in vhost '/'"
	// INSPECTION: example_queue_declare_robust
	// INSPECTION: example_queue_declare_robust
}

func ExampleChannel_Test() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Get a new channel from our robust connection for publishing. The channel is
	// created with our default middleware.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Get a channel testing harness. In a real test function, you would pass the
	// test's *testing.T value. Here, we will just pass a dummy one.
	testHarness := channel.Test(new(testing.T))

	// We can use the test harness to force the channel to channelReconnect. If a reconnection
	// does not occur before the passed context expires, the test will be failed.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testHarness.ForceReconnect(ctx)

	// We can check how many times a reconnection has occurred. The first time we
	// connect to the broker is counted, so we should get '2':
	fmt.Println("RECONNECTION COUNT:", testHarness.ReconnectionCount())

	// exit.

	// RECONNECTION COUNT: 2
}

// Add a custom middleware to channels created by a connection.
func ExampleChannel_addMiddleware() {
	// define our new middleware
	queueDeclareMiddleware := func(
		next amqpmiddleware.HandlerQueueDeclare,
	) amqpmiddleware.HandlerQueueDeclare {
		return func(args amqpmiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
			fmt.Println("MIDDLEWARE INVOKED FOR QUEUE")
			fmt.Println("QUEUE NAME :", args.Name)
			fmt.Println("AUTO-DELETE:", args.AutoDelete)
			return next(args)
		}
	}

	// Create a config and add our middleware to it.
	config := amqp.DefaultConfig()
	config.ChannelMiddleware.AddQueueDeclare(queueDeclareMiddleware)

	// Get a new connection to our test broker.
	connection, err := amqp.DialConfigCtx(
		context.Background(), amqptest.TestDialAddress, config,
	)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Get a new channel from our robust connection for publishing. The channel is
	// created with our default middleware.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Declare our queue, our middleware will be invoked and print a message.
	_, err = channel.QueueDeclare(
		"example_middleware",
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	// MIDDLEWARE INVOKED FOR QUEUE
	// QUEUE NAME : example_middleware
	// AUTO-DELETE: true
}

// CustomMiddlewareProvider exposes methods for middlewares that need to coordinate.
type CustomMiddlewareProvider struct {
	InvocationCount int
}

// TypeID implements amqpmiddleware.ProvidesMiddleware and returns a unique type ID
// that can be used to fetch middleware values when testing.
func (middleware *CustomMiddlewareProvider) TypeID() amqpmiddleware.ProviderTypeID {
	return "CustomMiddleware"
}

// QueueDeclare implements amqpmiddleware.ProvidesQueueDeclare.
func (middleware *CustomMiddlewareProvider) QueueDeclare(
	next amqpmiddleware.HandlerQueueDeclare,
) amqpmiddleware.HandlerQueueDeclare {
	return func(args amqpmiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
		middleware.InvocationCount++
		fmt.Printf(
			"DECLARED: %v, TOTAL: %v\n", args.Name, middleware.InvocationCount,
		)
		return next(args)
	}
}

// QueueDelete implements amqpmiddleware.ProvidesQueueDelete.
func (middleware *CustomMiddlewareProvider) QueueDelete(
	next amqpmiddleware.HandlerQueueDeclare,
) amqpmiddleware.HandlerQueueDeclare {
	return func(args amqpmiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
		middleware.InvocationCount++
		fmt.Printf(
			"DELETED: %v, TOTAL: %v\n", args.Name, middleware.InvocationCount,
		)
		return next(args)
	}
}

// NewCustomMiddlewareProvider creates a new CustomMiddlewareProvider.
func NewCustomMiddlewareProvider() amqpmiddleware.ProvidesMiddleware {
	return new(CustomMiddlewareProvider)
}

func ExampleChannel_customMiddlewareProvider() {
	// Create a config and add our middleware provider factory to it.
	config := amqp.DefaultConfig()
	config.ChannelMiddleware.AddProviderFactory(NewCustomMiddlewareProvider)

	// Get a new connection to our test broker.
	connection, err := amqp.DialConfigCtx(
		context.Background(), amqptest.TestDialAddress, config,
	)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Get a new channel from our robust connection for publishing. The channel is
	// created with our default middleware.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Declare our queue, our middleware will be invoked and print a message.
	_, err = channel.QueueDeclare(
		"example_middleware",
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		panic(err)
	}

	// Delete our queue, our middleware will be invoked and print a message.
	_, err = channel.QueueDelete(
		"example_middleware",
		false, // ifUnused
		false, // ifEmpty
		false, // noWait
	)
	if err != nil {
		panic(err)
	}

	// MIDDLEWARE INVOKED FOR QUEUE
	// DECLARED: example_middleware, TOTAL: 1
	// DELETED: example_middleware, TOTAL: 2
	// AUTO-DELETE: true
}
