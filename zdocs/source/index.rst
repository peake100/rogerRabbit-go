Roger, Rabbit
=============

Roger, Rabbit is broken into two packages:

- *rogerRabbit/amqp*: A drop-in replacement for streadway/amqp with automatic re-dials,
  method middleware, and more.

- *rogerRabbit/roger*: Higher level abstractions for publishing and consuming messages.

.. toctree::
   :maxdepth: 2
   :caption: Contents:

   ./amqp.rst
   ./roger.rst

Demo
----

**Survive a Disconnection Event Using the amqp Package:**

.. code-block::

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

.. code-block::

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
    producer := amqpproducer.New(channel, nil)
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

Motivations
-----------

`streadway/amqp`_, the official rabbitMQ driver for go is an excellent library with a
great API, but  limited scope. By design, It offers a full implementation of the AMQP
spec, but comes with very few quality-of-life featured beyond that. From it's
documentation:

.. code-block:: text

    Goals

        Provide a functional interface that closely represents the AMQP 0.9.1 model
        targeted to RabbitMQ as a server. This includes the minimum necessary to
        interact the semantics of the protocol.

    Things not intended to be supported:

        Auto reconnect and re-synchronization of client and server topologies.

        Reconnection would require understanding the error paths when the topology
        cannot be declared on reconnect. This would require a new set of types and code
        paths that are best suited at the call-site of this package. AMQP has a dynamic
        topology that needs all peers to agree. If this doesn't happen, the behavior
        is undefined. Instead of producing a possible interface with undefined
        behavior, this package is designed to be simple for the caller to implement the
        necessary connection-time topology declaration so that reconnection is trivial
        and encapsulated in the caller's application code.

Without a supplied way to handle reconnections, `bespoke <https://ninefinity.org/post/ensuring-rabbitmq-connection-in-golang/>`_
`solutions <https://medium.com/@dhanushgopinath/automatically-recovering-rabbitmq-connections-in-go-applications-7795a605ca59>`_
`abound <https://www.ribice.ba/golang-rabbitmq-client/>`_.

Most of these solutions are overly-fitted to a specific problem (consumer vs producer or
involve domain-specific logic), that is prone to data races (can you spot them in the
first link?), cumbersome to inject into a production code (do we abort the business
logic on an error or try to recover in-place?), and bugs (each solution has its own
redial bugs rather than finding them in a single lib where fixes can benefit everyone
and community code coverage is high).

Nome of this is meant to disparage the above solutions, they likely work great in the
code they were created for, but they point to a need that is not being filled by the
official driver. The nature of the default `*Channel` API encourages solutions that
are ill-suited to stateless handlers OR require you to handle retries every place you
must interact with the broker. Such implementation details can be annoying when writing
higher-level business logic and can lead to either unnecessary error returns, bespoke
solutions in every project, or messy calling code at sites which need to interact with
an AMQP broker.

Roger, Rabbit is inspired by `aio-pika's <https://aio-pika.readthedocs.io/en/latest/index.html>`_
`robust connections and channels <https://aio-pika.readthedocs.io/en/latest/apidoc.html#aio_pika.connect_robust>`_,
which abstract away connection management with an identical API to their non-robust
connection and channel API's.

.. note::

    This library is not meant to supplant `streadway/amqp`_ (Roger, Rabbit is built on
    top of it!), but an extension with quality of life features. Roger, Rabbit would not
    be possible without the amazing groundwork laid down by `streadway/amqp`_.

Goals
-----

The goals of the Roger, Rabbit package are as follows:

- **Offer a drop-in replacement for rogerRabbit/amqp**: APIs may be extended (adding
  fields to `amqp.Config` or additional methods to `*amqp.Channel`, for instance) but
  must not break existing code unless absolutely necessary.

- **Add as few additional error paths as possible**: Errors may be *extended* with
  additional information concerning disconnect scenarios, but new error type returns
  from `*Connection` or `*amqp.Channel` should be an absolute last resort.

- **Be Highly Extensible**: Roger, Rabbit seeks to offer a high degree of extensibility
  via features like middleware, in an effort to reduce the balkanization of amqp client
  solutions.

Current Limitations & Warnings
------------------------------

- **Performance**: Roger, Rabbit's implementation is handled primarily through
  middleware and a `*sync.RWMutex` on transports that handles blocking methods on
  reconnection events. This increases the overhead allows for an enormous amount of
  extensibility and robustness, but may be a limiting factor for applications that need
  the absolute maximum throughput possible.

- **Transaction Support**: Roger, Rabbit does not currently support amqp Transactions,
  as the author does not use them. Draft PR's with possible implementations are welcome!

- **Reliability**: While the author uses this library in production, it is still early
  days, and more battle-testing will be needed before this library is promoted to
  version 1.0. PR's are welcome for Bug Fixes, code coverage, or new features.

.. note::

    Currently, there are not side-by-side benchmarks for Roger, Rabbit and
    `streadway/amqp`_, but PR's to add them are welcome!

API Documentation
-----------------

API Documentation is hosted on pkg.go.dev and can be
`found here <https://pkg.go.dev/github.com/peake100/rogerRabbit-go?readme=expanded#section-documentation>`_.

.. _streadway/amqp: https://github.com/streadway/amqp