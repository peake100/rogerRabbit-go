=============
Roger, Rabbit
=============

.. image:: _static/logo-tophat-tall.svg
    :width: 40%
    :align: center

.. toctree::
   :maxdepth: 2
   :caption: Contents:

   ./overview.rst
   ./amqp.rst
   ./roger.rst

Demo
----

Examples:

**Survive a Disconnection Event Using the amqp Package:**

.. code-block:: go

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

    // We can use the Test method to return a testing harness with some additional
    // methods. ForceReconnect force-closes the underlying streadway Channel, causing
    // the robust Channel to reconnect.
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

**Effortless Publishing with Confirmations using the roger Package:**

.. code-block:: go

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
				"", // exchange
				queue.Name, // queue
				true, // mandatory
				false, // immediate
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

API Documentation
-----------------

API Documentation is hosted on pkg.go.dev and can be
`found here <https://pkg.go.dev/github.com/peake100/rogerRabbit-go?readme=expanded#section-documentation>`_.

.. _streadway/amqp: https://github.com/streadway/amqp