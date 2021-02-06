AMQP Package
############

Introduction
============

The amqp package is a drop-in replacement for `streadway/amqp`_, and is designed to enable
automatic redialing to any code using the streadway package with no changes.

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
statements of the official examples to `"github.com/peake100/rogerRabbit-go/amqp"`
will result in working code that handles unexpected broker disconnects.

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
    "github.com/peake100/rogerRabbit-go/amqp"
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

Both the `Connection` and `Channel` types are robust transport mirrors of the streadway
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

Roger, Rabbit's `Channel` type remembers all called to `Channel.QueueDeclare()`,
`Channel.QueueBind()`, `Channel.ExchangeDeclare()` and `Channel.ExchangeBind()`, and
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

  Calling `Channel.QueueDelete()`, `Channel.QueueUnbind()`, `Channel.ExchangeDelete`,
  and `Channel.ExchangeUnbind()` will remove relevant robust queues and bindings from
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

  The `Confirmation.DisconnectOrphan` is a new field for the `Confirmation` type and
  is unique to Roger, Rabbit.

  When `DisconnectOrphan` is true, it means that a Nack occurred not from a broker
  response, but because no confirmation positive or negative was received from the
  broker before the connection was disrupted. Orphaned messages may have reached the
  broker -- we have no way of knowing.

Channel Middleware
==================

Roger, Rabbit allows the registration of middleware on all `Channel` methods. In fact,
most of the robust `Channel` features are implemented through middleware defined in
the `amqp/defaultMiddlewares` package! It is a powerful tool and one of the biggest
API expansions over streadway/amqp.

Middleware signatures are defined in the `amqp/amqpMiddleware` package.

Registering a new middleware:

.. code-block::

  // define our new middleware
  queueDeclareMiddleware := func(
    next amqpMiddleware.HandlerQueueDeclare,
  ) amqpMiddleware.HandlerQueueDeclare {
    return func(args *amqpMiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
      fmt.Println("MIDDLEWARE INVOKED FOR QUEUE")
      fmt.Println("QUEUE NAME :", args.Name)
      fmt.Println("AUTO-DELETE:", args.AutoDelete)
      return next(args)
    }
  }

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

  // Register our middleware for queue declare.
  channel.Middleware().AddQueueDeclare(queueDeclareMiddleware)

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

  // Output:
  // MIDDLEWARE INVOKED FOR QUEUE
  // QUEUE NAME : example_middleware
  // AUTO-DELETE: true

.. Note::

  Middleware is currently only implemented for methods required for use by the default
  middleware that supplies most of the `Channel` type's features.

  If you wish to have middleware support added to a method currently missing it, please
  open an issue and it will be added!

Testing
=======

Testing is a first-class citizen of the Roger, Rabbit package. Types expose a robust
number of testing methods, and the `amqpTest` offers a number of additional testing
utilities.

.. note::

  Testing methods and utilities are heavily integrated with
  `testify <https://godoc.org/github.com/stretchr/testify>`_. Testify is somewhat
  divisive within the Go community, but as the maintainers of this repository heavily
  leverage it, so does Roger, Rabbit's testing utilities.

Test() Methods
--------------

Both `Connection` and `Channel` expose a `.Test()` method, which returns a testing
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

The `amqpTesting` package makes an `AmqpSuite` type that is an extension of
`testify/suite.Suite <https://godoc.org/github.com/stretchr/testify/suite>`_

`AmqpSuite` adds a number of QoL methods for quickly setting up an tearing down
integration tests with a test broker.

See the godoc documentation for more details.
