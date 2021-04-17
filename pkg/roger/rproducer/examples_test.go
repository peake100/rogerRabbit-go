package rproducer_test

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/amqptest"
	"github.com/peake100/rogerRabbit-go/pkg/roger/rproducer"
	"sync"
	"time"
)

func ExampleNew() {
	// Get a new connection to our test broker.
	connection, err := amqp.DialCtx(context.Background(), amqptest.TestDialAddress)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Get a new channel from our robust connection for publishing. This channel will
	// be put into confirmation mode by the producer.
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	// Declare a queue to produce to
	queue, err := channel.QueueDeclare(
		"example_confirmation_producer", // name
		false,                           // durable
		true,                            // autoDelete
		false,                           // exclusive
		false,                           // noWait
		nil,                             // args
	)

	// Create a new producer using our channel. Passing nil to opts will result in
	// default opts being used. By default, a Producer will put the passed channel in
	// confirmation mode, and each time publish is called, will block until a
	// confirmation from the server has been received.
	producer := rproducer.New(channel, nil)
	producerComplete := make(chan struct{})

	// Run the producer in it's own goroutine.
	go func() {
		// Signal this routine has exited on exit.
		defer close(producerComplete)

		err = producer.Run()
		if err != nil {
			panic(err)
		}
	}()

	messagesPublished := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {

		messagesPublished.Add(1)

		// Publish each message in it's own goroutine The producer handles the
		// boilerplate of tracking publication tags and receiving broker confirmations.
		go func() {
			// Release our WaitGroup on exit.
			defer messagesPublished.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Publish a message, this method will block until we get a publication
			// confirmation from the broker OR ctx expires.
			err = producer.Publish(
				ctx,
				"",         // exchange
				queue.Name, // queue
				true,       // mandatory
				false,      // immediate
				amqp.Publishing{
					Body: []byte("test message"),
				},
			)

			fmt.Println("Message Published and Confirmed!")

			if err != nil {
				panic(err)
			}
		}()
	}

	// Wait for all our messages to be published
	messagesPublished.Wait()

	// Start the shutdown of the producer
	producer.StartShutdown()

	// Wait for the producer to exit.
	<-producerComplete

	// exit.

	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
	// Message Published and Confirmed!
}
