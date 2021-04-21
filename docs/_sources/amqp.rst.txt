AMQP Package
############

Introduction
============

The amqp package is a drop-in replacement for `streadway/amqp`_, and is designed to
enable automatic redialing to any code using the streadway package with no changes.

To begin using rogerRabbit/amqp, simply change your imports from:

.. code-block:: go

  import "github.com/streadway/amqp"

To:

.. code-block:: go

  import "github.com/peake100/rogerRabbit-go/amqp"

Under the hood, this library is a wrapper around `streadway/amqp`_ and would not be
possible without the amazing development team behind it.

Roger, Rabbit seeks to expand on the work done by `streadway/amqp`_, not replace it.

Basic Usage
-----------

As this package is a drop-in replacement for `streadway/amqp`_, basic usage will not be
covered by these docs.

The `official rabbitMQ documentation <https://www.rabbitmq.com/getstarted.html>`_ has
an excellent set of tutorials using the streadway library. Simply replacing the import
statements of the official examples to ``"github.com/peake100/rogerRabbit-go/pkg/amqp"``
will allow you to follow along as normal using this package.

.. note::

    If you find any examples that do NOT work after such a replacement, please open a
    PR!

For instance, the hello world receiver example would be simply changed from this:

.. code-block:: go

  package main

  import (
    "log"

    "github.com/streadway/amqp"
  )

  func failOnError(err error, msg string) {
    if err != nil {
      log.Fatalf("%s: %s", msg, err)
    }
  }

  func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, err := ch.QueueDeclare(
      "hello", // name
      false,   // durable
      false,   // delete when unused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )
    failOnError(err, "Failed to declare a queue")

    msgs, err := ch.Consume(
      q.Name, // queue
      "",     // consumer
      true,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    forever := make(chan bool)

    go func() {
      for d := range msgs {
        log.Printf("Received a message: %s", d.Body)
      }
    }()

    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever
  }

To this:

.. code-block:: go

  package main

  import (
    "log"

    // ONLY THIS CHANGES
    "github.com/peake100/rogerRabbit-go/pkg/amqp"
  )

  func failOnError(err error, msg string) {
    if err != nil {
      log.Fatalf("%s: %s", msg, err)
    }
  }

  func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, err := ch.QueueDeclare(
      "hello", // name
      false,   // durable
      false,   // delete when unused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )
    failOnError(err, "Failed to declare a queue")

    msgs, err := ch.Consume(
      q.Name, // queue
      "",     // consumer
      true,   // auto-ack
      false,  // exclusive
      false,  // no-local
      false,  // no-wait
      nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    forever := make(chan bool)

    go func() {
      for d := range msgs {
        log.Printf("Received a message: %s", d.Body)
      }
    }()

    log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
    <-forever
  }

This documentation will focus on the differences from -- and expansion upon -- the
streadway API, rather than retreading a primer on how to work with the basic API.

It is suggested that users new to amqp who have not used `streadway/amqp`_ start with
the basic RabbitMQ tutorials before continuing this documentation.

When To Use This Library
------------------------

Roger, Rabbit is designed to remove all the mental overhead involved with managing
unexpected broker disconnects, it features systems to automatically recreate
client / server topologies on reconnect, ensure consistent Delivery and Publish tags
over reconnection events, and more.

But there are some assumptions that need to be made or such features. In general, this
library should be used when:

- **Basic Client/Server Topology**: All calls on a channel to QueueDeclare, QueueBind,
  ExchangeDeclare and ExchangeUnbind will be re-made every time a channel drops it's
  connection and has to reconnect. In general, if you are using complex topology where
  Queues and Exchanges are being routinely shifted, deleted, and altered this library's
  behavior may result in the re-declaration of unwanted Queues and Exchanges.

- **Orphaned Publications Are Nacks**: Failed in-flight Publications on Channels in
  confirmation mode will be exposed to the end user as a Nacked publication. There is
  an additional extension of the streadway API to flag Orphaned publications, but such
  handling will require code tweaks and not be a drop-in replacement.

- **Delivery Acknowledgements do not Mix Per-Message and Multiple**: Roger, Rabbit will
  detect orphaned acknowledgements and return an error when orphans occur (the broker is
  disconnected before a delivery is acknowledged), but mixing Ack with multiple=true and
  multiple=false may confuse the library, and is currently not supported by Roger,
  Rabbit.

.. _streadway/amqp: https://github.com/streadway/amqp

Unsupported Features
--------------------

Roger, Rabbit strives to be a complete, drop-in replacement for `streadway/amqp`_, but
is still under construction. The following features have yet to be implemented:

- Transactions: Calling Channel.Tx(), Channel.TxCommit() and Channel.TxRollback() will
  result in a panic. Transactions are an interesting problem to solve for with robust
  channels and draft PRs for how to handle them are welcome!

Robust Features
===============

In this section, we will examine features unique to Roger, Rabbit.

Connection Recovery
-------------------

Both the ``Connection`` and ``Channel`` types are robust transport mirrors of the streadway
types by the same names, and will automatically re-connect when a connection is lost:

.. code-block::

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
  if err != nil {
    panic(err)
  }

  // Get a new channel from our robust connection.
  channel, err := connection.Channel()
  if err != nil {
    panic(err)
  }

  // We can use the test method to return an testing object with some additional
  // methods. ForceReconnect force-closes the underlying transport, causing the
  // robust connection to reconnect.
  //
  // We'll use a dummy *testing.T object here. These methods are designed for tests
  // only and should not be used in production code.
  channel.Test(new(testing.T)).ForceReconnect(context.Background())

  // We can see here our channel is still open.
  fmt.Println("IS CLOSED:", channel.IsClosed())

  // We can even declare a queue on it
  queue, err := channel.QueueDeclare(
    "example_channel_reconnect", // name
    false, // durable
    true, // autoDelete
    false, // exclusive
    false, // noWait
    nil, // args
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

  // Output:
  // IS CLOSED: false
  // QUEUE    : {Name:example_channel_reconnect Messages:0 Consumers:0}
  // IS CLOSED: true

Topology Recreation
-------------------

Roger, Rabbit's ``Channel`` type remembers all called to ``Channel.QueueDeclare()``,
``Channel.QueueBind()``, ``Channel.ExchangeDeclare()`` and ``Channel.ExchangeBind()``, and
replays those calls on a reconnection event:

.. code-block: go

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
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
    true, // autoDelete
    false, // exclusive
    false, // noWait
    nil, // args
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

  // Output:
  // INSPECT ERROR: Exception (404) Reason: "NOT_FOUND - no queue 'example_queue_declare_robust' in vhost '/'"
  // INSPECTION: example_queue_declare_robust
  // INSPECTION: example_queue_declare_robust

.. Note::

  Calling ``Channel.QueueDelete()``, ``Channel.QueueUnbind()``, ``Channel.ExchangeDelete``,
  and ``Channel.ExchangeUnbind()`` will remove relevant robust queues and bindings from
  the internally tracked lists. Queues invoked in these methods will NOT be recreated
  on a reconnection event.

Delivery Tag Continuity
-----------------------

Delivery tags remain continuous, even across unexpected disconnects. Roger, rabbit takes
care of all the internal logic of lining up the caller-facing delivery tag with the
actual delivery tag relative to the current underlying channel:

.. code-block::

  // Get a new connection to our test broker.
  connection, err := amqp.DialCtx(context.Background(), amqpTest.TestDialAddress)
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
    false, // durable
    false, // autoDelete
    false, // exclusive
    false, // noWait
    nil, // args
  )
  if err != nil {
    panic(err)
  }

  // Clean up the queue on exit,
  defer consumeChannel.QueueDelete(
    queue.Name, false, false, false,
  )

  // Set the prefetch count to 1, that way we are less likely to lose a message
  // that in in-flight from the broker in this example.
  err = consumeChannel.Qos(1, 0, false)
  if err != nil {
    panic(err)
  }

  // Start consuming the channel
  consume, err := consumeChannel.Consume(
    queue.Name,
    "example consumer", // consumer name
    false,              // autoAck
    false,              // exclusive
    false,              // no local
    false,              // no wait
    nil,                // args
  )

  // We'll close this channel when the consumer is exhausted
  consumeComplete := new(sync.WaitGroup)
  consumerClosed := make(chan struct{})

  // Launch a consumer
  go func() {
    // Close the consumeComplete to signal exit
    defer close(consumerClosed)

    fmt.Println("STARTING CONSUMER")

    // Range over the consume channel
    for delivery := range consume {
			// Ack the delivery.
			err = delivery.Ack(false)
			if err != nil {
				panic(err)
			}

      // Force-reconnect the channel after each delivery.
      consumeChannel.Test(new(testing.T)).ForceReconnect(context.Background())

      // Tick down the consumeComplete waitgroup
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

  // Output:
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

.. Warning::

  In the above example, is possible that we will drop the publishing of message during a
  reconnection event. If we want to be sure all messages reach the broker, we'll need to
  publish messages with the Channel in confirmation mode, which we will
  show in the next example.

Delivery Tag Orphans
--------------------

When manually acking Deliveries, it is possible that between the time we get a Delivery,
and the time that we ack it, a disconnection of the underlying channel has occurred and
the delivery is no longer acknowledgable. In such cases, an error will be returned
indicating this delivery has been orphaned:

.. code-block::

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
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
    false, // durable
    true, // autoDelete
    false, // exclusive
    false,  // noWait
    nil, // args
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
    nil, // args
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
  delivery := <- consume
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

  // Output:
  // DELIVERY: test message
  // ACK ERROR: 1 tags orphaned (1 - 1), 0 tags successfully acknowledged
  // FIRST ORPHAN TAG: 1
  // LAST ORPHAN TAG : 1

Publishing Tag Continuity
-------------------------

Just like with Delivery Tags, publishing tag continuity is maintained, even across
disconnection events.

.. code-block:: go

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
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
  queueName := "example_delivery_tag_continuity"
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
  for i := 0 ; i < 10 ; i++ {
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

  // Output:
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

.. note::

  The ``Confirmation.DisconnectOrphan`` is a new field for the ``Confirmation`` type and
  is unique to Roger, Rabbit.

  When ``DisconnectOrphan`` is true, it means that a Nack occurred not from a broker
  response, but because no confirmation positive or negative was received from the
  broker before the connection was disrupted. Orphaned messages may have reached the
  broker -- we have no way of knowing.

Channel Middleware
==================

Roger, Rabbit allows the registration of middleware on all ``Channel`` methods. In fact,
most of the robust ``Channel`` features are implemented through middleware defined in
the ``amqp/defaultMiddlewares`` package! It is a powerful tool and one of the biggest
API expansions over streadway/amqp.

Middleware signatures are defined in the ``amqp/amqpMiddleware`` package.

Registering Middleware
----------------------

.. code-block:: go

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

Middleware Providers
--------------------

For more complex middleware, you can implement middleware providers, which expose
methods that implement middleware. You can then register a factory method that will
generate a new provider value whenever a new Connection or Channel is created, and
the middleware methods will be automatically registered for you.

This can help build complex middlewares. Let's define a custom middleware provider and
factory method:

.. code-block:: go

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

We can register it on our Config so that every channel created from a connection
gets a fresh instance of our provider:

.. code-block:: go

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
        true, // autoDelete
        false, // exclusive
        false, // noWait
        nil, // args
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

if, for some reason, we wanted every Channel to share the *same* instance of the
provider, we could make the following adjustment:

.. code-block:: go

    // Create an instance of our provider:
    provider := NewCustomMiddlewareProvider()

    config := amqp.DefaultConfig()
    config.ChannelMiddleware.AddProviderMethods(provider)

This will add the methods once and re-use them for all Channels rather than making a
fresh provider every time a channel is generated.

.. note::

    Many of rogerRabbit's more complex features are implemented through middleware
    providers. Check the ``defaultmiddlewares`` package to see practical examples of
    middleware providers used in this library.

Testing
=======

Testing is a first-class citizen of the Roger, Rabbit package. Types expose a robust
number of testing methods, and the ``amqpTest`` offers a number of additional testing
utilities.

.. note::

  Testing methods and utilities are heavily integrated with
  `testify <https://godoc.org/github.com/stretchr/testify>`_. Testify is somewhat
  divisive within the Go community, but as the maintainers of this repository heavily
  leverage it, so does Roger, Rabbit's testing utilities.

Test() Methods
--------------

Both ``Connection`` and ``Channel`` expose a ``.Test()`` method, which returns a testing
harness type with additional methods for running tests on it's parent value.

Most test methods do not return an error, instead opting to report the error and
immediately fail the test.

Example:

.. code-block:: go

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
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

  // We can use the test harness to force the channel to reconnect. If a reconnection
  // does not occur before the passed context expires, the test will be failed.
  ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
  defer cancel()
  testHarness.ForceReconnect(ctx)

  // We can check how many times a reconnection has occurred. The first time we
  // connect to the broker is counted, so we should get '2':
  fmt.Println("RECONNECTION COUNT:", testHarness.ReconnectionCount())

  // exit.

  // Output:
  // RECONNECTION COUNT: 2

Testify Suite
-------------

The ``amqpTesting`` package makes an ``AmqpSuite`` type that is an extension of
`testify/suite.Suite <https://godoc.org/github.com/stretchr/testify/suite>`_

``AmqpSuite`` adds a number of QoL methods for quickly setting up an tearing down
integration tests with a test broker.

See the godoc documentation for more details.

Architecture
============

Overview
--------

.. figure:: _static/architecture.svg
    :align: center

    Schematic of the amqp transport (``Channel`` and ``Connection``) architecture.

Lets dig in to some of the specifics.

.. note::

    When we refer to an amqp.[Type], we are referring to the rogerRabbit-go/pkg/amqp
    implementation of that type. If we need to refer to underlying types provided by
    the streadway/amqp package, we will refer to them as streadway.[Type]

.. warning::

    The file organization of the ``amqp`` package is still in flux, though the API and
    underlying logical structure is likely to remain stable. This architecture guide
    will refrain from referencing specific files or paths for the time being, until
    the types and helpers are done playing musical chairs.

Streadway Transport
-------------------

The ``Streadway Transport`` is our underlying ``streadway.Connection`` or
``streadway.Channel`` and handles all of our actual communication with the
``AMQP Broker``.

It is manages with by a ``Robust Transport``.

Robust Transport
----------------

A Robust Transport is an ``amqp.Channel`` or ``amqp.Connection``. These types manage
their ``streadway.Channel`` or ``streadway.Connection`` counterparts, and expose
``Exported Methods`` to the caller that re-implement the methods supplied by the
``Streadway Transport`` they manage.

Each ``Robust Transport`` implements the ``reconnects`` interface, which in turn is
referenced by the ``Transport Manager`` and used to redial the underlying broker
connection.

So why are we talking about the ``Transport Manager`` here if it manages the
``Robust Transport``? Well, because the ``Transport Manager`` is then embedded into the
Robust Transport it is managing, and provides some of that transport's methods.

``Close()``, ``NotifyDisconnect()``, and all other methods that are shared between
``amqp.Channel`` and ``amqp.Connection`` are actually provided by an embedded
``transportManager`` value which, like an ouroboros, also contains a reference to it's
parent transport that is used to help manage the lifecycle of that transport's
underlying streadway transport through the shared ``reconnects`` interface.

Method Handlers
---------------

Each ``Robust Transport`` (and the ``Transport Manager``) contains a ``handlers`` field
which holds all the method handlers for that transport. These handlers are comprised of
the base streadway/amqp type methods wrapped in user-supplied ``Middleware``.

Middleware
----------

``Middleware`` is a core component of the Roger, Rabbit amqp package. Most of the Roger,
Rabbit's extended functionality is supplied by middleware, such as continuous Delivery
Tags over disconnects and auto-redeclaration of broker topology on reconnections.

In the early life of this package, robust features were implemented directly on the
``Channel``, resulting in features which were hard to isolate and maintain. Topology
re-declaration, for instance, involves 9 methods manipulating 6 data resources over
~700 lines of code. Debugging and tracking errors with these features was incredibly
cumbersome when they were spread out over the main ``Channel`` method calls.

By moving these features into middleware, all off the logic that supports a given
feature can be housed together, greatly reducing the complexity and increasing
maintainability.

The current default middleware that ships with Roger, Rabbit is:

- **ConfirmsMiddleware**: Tracks whether ``Channel.Confirms`` has been called, and sets
  any freshly reconnected streadway Channels into the correct state.

- **DeliveryTagsMiddleware**: Ensures that Delivery tags are continuous for the caller,
  even over disconnects.

- **FlowMiddleware**: Tracks the Flow state of channel and sets the correct state on
  Channel reconnections.

- **LoggingMiddlewareConnection**: Facilitates logging for all Connection Operations.

- **LoggingMiddlewareChannel**: Facilitates logging for all Channel Operations.

- **PublishTagsMiddleware**: Ensures that Publish Confirmation tags are continuous for
  the caller, even over disconnects.

- **QoSMiddleware**: Tracks QoS setting and sets up Channels on reconnect to match.

- **RouteDeclarationMiddleware**: Handles topology recreation on Channel reconnects.

Together these middlewares implement the bulk of our robust feature set.

Transport Lock
--------------

The ``TransportLock`` is a ``*sync.RWMutex`` contained on the ``transportManager`` used to
ensure there is no race conditions when a disconnection event occurs.

Any process that wishes to make use of the underlying streadway connection must acquire
the transport lock for read.

When the transport manager redials the broke, the transport lock is acquired for write,
and not released until a successful connection occurs. This blocks all other operations
(like channel and connection method calls) from interacting with the underlying
streadway transport until we are reconnected.

Transport Manager
-----------------

The ``Transport Manager`` implements all common functionality between ``Channel`` and
``Connection``. See the above section for more details.

The ``Transport Manager`` listens for a disconnection event to occur, the  grabs an
exclusive Write access to the ``Transport Lock``. Only then does it continuously attempt
to redial the broker using the methods exposed on out ``Robust Connection`` through the
``reconnects`` interface.

The transport manager also exposes a ``Retry On Disconnected`` method which all of our
exported ``Channel`` and ``Connection`` ``Exported Methods`` invoke to complete
requests.

Redial Routine
--------------

The ``Redial Routine`` is launched by the ``Transport Manager``. The redial routine
is responsible for reconnecting to the underlying ``Streadway Transport`` on a
disconnection event.

#. Register's a listener with the ``Streadway Transport``'s ``NotifyOnClose`` method.

#. Blocks until the listener signals that ``Streadway Connection`` has closed
   (disconnection event)

#. Exits if our ``Robust Transport`` has been closed by the caller.

#. Acquires the ``Transport Lock`` for write, blocking ``Exported Method`` calls until
   a successful reconnection occurs.

#. Calls the ``tryReconnect`` method on the ``Robust`` transport until we get a
   successful result.

#. Register's a listener with the new ``Streadway Connection``'s ``NotifyOnClose``
   method.

#. Releases the ``Transport Lock``.

#. Restarts at step 2.

Retry On Disconnected
---------------------

``Retry On Disconnected`` is a method exposed by ``Transport Manager`` that enables
``Robust Transports`` (``Channel`` and ``Connection``) to automatically retry an
operation if it failed because we were disconnected from the AMQP broker.

When invoked, ``Retry On Disconnected`` does the following:

#. Acquires the ``Transport Lock`` for read (so multiple methods can be called
   simultaneously).

#. Runs the operation handed to it by the ``Exported Method``, usually a
   ``Method Handler``.

#. Releases the ``Transport Lock``.

#. If an error occurred because the broker was not reachable, goes back to step 1.

#. Passes the result back to up to the caller.

Event Relays
------------

Many ``Channel`` methods like ``NotifyPublish`` or ``Consume`` involve sending
events along a provided or returned Go ``chan [Event]`` value to the caller.

When a ``streadway.Channel`` disconnects, it closes all event ``chan [Event]`` values
it is feeding. Because we want our event ``chan [Event]`` values to survive a
disconnect, we cannot pass the caller's ``chan [Event]`` directly to a
``streadway.Channel``. Instead, we need to create our own temporary ``chan [Event]``,
pass that to the ``Streadway Transport``, then relay the events from our temporary
``chan [Event]`` to the caller's ``chan [Event]``.

To do this, we launch an ``Event Relay``,

When a reconnection event occurs, the relay creates a new temporary ``chan [Event]``,
passes that to the new ``Streadway Transport``, and the continues to relay messages to
the caller as if nothing has happened.

Since ``Channel`` is the only transport that needs event relays, it manages their
lifecycle. ``Event Relays`` must be kept in careful sync with reconnection events to
ensure that data or logic races do not occur. That ``Event Relay`` lifecycle is as
follows:

#. A relay is started by the ``Channel.[Method]`` handler using the
   ``transportManager.retryOperationOnClosed`` function so it can acquire the
   ``Transport Lock`` for read when starting up (ensuring the ``streadway.Channel``
   is not swapped out from under it).

#. The relay runs it's first leg until the undeerlying ``streadway.Channel``
   disconnects.

#. On ``streadway.Channel`` disconnect, the temporary ``chan [Event]``
   feeding the relay dries up. The relay waits on a new ``*streadway.Channel`` to be
   sent by it's ``Channel`` on a successful reconnection.

#.  The ``transportManager`` acquires the ``Transport Lock`` for write and calls
    ``Channel.tryReconnect`` over and over until a new ``streadway.Channel`` is
    successfully connected.

#. When a new ``streadway.Channel`` is successfully established,
   ``Channel.tryReconnect`` does not return  until all the remaining steps are
   completed.

#. ``Channel.tryReconnect`` waits for all ``Event Relays`` to signal that their last
   leg has been successfully completed.

#. ``Channel.tryReconnect`` sends the new ``*streadway.Channel`` value to each relay
   for it to run any setup and start relaying events again.

#. ``Channel.tryReconnect`` waits for all relays to signal that their setup on
   ``*streadway.Channel`` is complete. If we were to return and release the
   ``Transport Lock`` before this, users might start taking actions that SHOULD generate
   events before our ``streadway.Channel`` had been set up to send them, resulting in
   dropped events.

#. ``Channel.tryReconnect`` returns with the new, connected ``*streadway.Channel``
   value releasing the ``Transport Lock`` so that callers can begin calling ``Channel``
   methods again.
