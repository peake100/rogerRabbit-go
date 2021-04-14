Roger Package
#############

The roger package contains a number of higher-level abstractions for working with
the amqp package.

Producer
========

The `Producer` type makes it dead-simple to publish messages with broker confirmations
enabled.

By default, the `Producer.Publish()` method will not return until a confirmation for
the publication has been received from the broker, an error occurs, or the context
passed to it expires, like so:

.. code-block:: go

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
  if err != nil {
    panic(err)
  }
  defer connection.Close()

  // Get a new channel from our robust connection for publishing.
  channel, err := connection.Channel()
  if err != nil {
    panic(err)
  }

  // Declare a queue to produce to.
  queue, err := channel.QueueDeclare(
    "example_confirmation_producer", // name
    false, // durable
    true, // autoDelete
    false, // exclusive
    false, // noWait
    nil, // args
  )

  // Create a new producer using our channel. Passing nil to opts will result in
  // default opts being used. By default, a Producer will put the passed channel in
  // confirmation mode, and each time publish is called, will block until a
  // confirmation from the server has been received.
  producer := roger.NewProducer(channel, nil)
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
  for i := 0 ; i < 10 ; i++ {

    messagesPublished.Add(1)

    // Publish each message in it's own goroutine.
    go func() {
      // Release our WaitGroup on exit.
      defer messagesPublished.Done()

      ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
      defer cancel()

      // Publish a message, this method will block until we get a publication
      // confirmation from the broker OR ctx expires.
      err = producer.Publish(
        ctx,
        "",
        queue.Name,
        false,
        false,
        amqp.Publishing{
          Body: []byte("test message"),
        },
      )

      fmt.Println("Message Published!")

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

  // Output:
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!
  // Message Published!

Consumer
========

The `Consumer` type allows registration of consumer handlers that take in a delivery and
return error information.

Delivery ACK, NACK, requeue and other boilerplate is handled for you behind the scenes.
Additional options like max concurrent processors are made available for setting up
robust consumer programs with as little boilerplate as possible.

A Route handler might looks something like this:

.. code-block:: go

  type BasicProcessor struct {
  }

  // ConsumeArgs returns the args to be made to the consumer's internal
  // Channel.Consume() method.
  func (processor *BasicProcessor) ConsumeArgs() *ConsumeArgs {
    return &ConsumeArgs{
      ConsumerName: "example_consumer_queue",
      AutoAck:      false,
      Exclusive:    false,
      Args:         nil,
    }
  }

  // SetupChannel is called before consuming begins, and allows the handler to declare
  // any routes, bindings, etc, necessary to handle it's route.
  func (processor *BasicProcessor) SetupChannel(
    ctx context.Context, amqpChannel AmqpRouteManager,
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
    ctx context.Context, delivery dataModels.Delivery, logger zerolog.Logger,
  ) (err error, requeue bool) {
    // Print the message
    fmt.Println("BODY:", delivery.Body)

    // Returning no error will result in an ACK of the message.
    return nil, false
  }

  // Cleanup allows the route handler to remove any resources necessary on close.
  func (processor *BasicProcessor) Cleanup(amqpChannel AmqpRouteManager) error {
    _, err := amqpChannel.QueueDelete(
      consumer.QueueName, false, false, false,
    )
    if err != nil {
      return fmt.Errorf("error deleting Queue: %w", err)
    }

    return nil
  }

Registering our handler and running our consumer might look something like this:

.. code-block:: go

  // Get a new connection to our test broker.
  connection, err := amqp.Dial(amqpTest.TestDialAddress)
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
  consumer := roger.NewConsumer(channel, nil)
  defer consumer.StartShutdown()

  // Create a new delivery processor and register it.
  processor := new(BasicProcessor)
  consumer.RegisterProcessor(processor)

  // This method will block forever as the consumer runs.
  err = consumer.Run()
  if err != nil {
    panic(err)
  }
