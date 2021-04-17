# Roger, Rabbit
<p align="center">
<img src="https://goreportcard.com/badge/github.com/peake100/rogerRabbit-go" /></a>
</p>

Roger, Rabbit is a drop-in replacement for 
[streadway/amqp](https://github.com/streadway/amqp) that offers the following
enhancements:

- **Automatic Connection Recoveries:** Both Connection and Channel objects will 
  automatically recover from unexpected broker disconnections, and restore channel and
  connection states so you don't miss a beat.
- **Custom Middleware**: For all Channel methods.
- **Testing utilities**: Force reconnects, and testify methods for common setup.
- **Convenience Types**: 
    - **Confirmation Producers**: handles broker confirmation on publish before
      returning.
    - **Consumer Framework**: Register http-like handlers to consume from queues, with
      nack-on-error, max concurrent worker, and other boilerplate
      
Getting Started
---------------

For quickstart and full API documentation:
[read the docs](https://peake100.github.io/rogerRabbit-go/).

For library development guide,
[read the docs](https://illuscio-dev.github.io/islelib-go/).

Demo
----

**Survive a Disconnection Event Using the amqp Package:**

```go
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
```

**Effortless Publishing with Confirmations using the roger Package:**

```go
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
```

Motivations
-----------

[streadway/amqp](https://github.com/streadway/amqp), the official rabbitMQ driver for 
Go, is an excellent library with a great API but limited scope. By design, It offers a 
full implementation of the AMQP spec, but comes with very few quality-of-life featured
beyond that. 

From its documentation:

>## Goals
>
>   - Provide a functional interface that closely represents the AMQP 0.9.1 model
    targeted to RabbitMQ as a server. This includes the minimum necessary to
    interact the semantics of the protocol.
>
>## Things not intended to be supported:
>
>   - Auto reconnect and re-synchronization of client and server topologies.
>
>      - Reconnection would require understanding the error paths when the topology
    cannot be declared on reconnect. This would require a new set of types and code
    paths that are best suited at the call-site of this package. AMQP has a dynamic
    topology that needs all peers to agree. If this doesn't happen, the behavior
    is undefined. Instead of producing a possible interface with undefined
    behavior, this package is designed to be simple for the caller to implement the
    necessary connection-time topology declaration so that reconnection is trivial
    and encapsulated in the caller's application code.

Without a supplied way to handle reconnections, 
[bespoke](https://ninefinity.org/post/ensuring-rabbitmq-connection-in-golang/>)
[solutions](<https://medium.com/@dhanushgopinath/automatically-recovering-rabbitmq-connections-in-go-applications-7795a605ca59>)
[abound](<https://www.ribice.ba/golang-rabbitmq-client/>).

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

Roger, Rabbit is inspired by [aio-pika's](https://aio-pika.readthedocs.io/en/latest/index.html)
[robust connections and channels](https://aio-pika.readthedocs.io/en/latest/apidoc.html#aio_pika.connect_robust)
which abstract away connection management with an identical API to their non-robust
connection and channel API's, allowing robust AMQP broker interactions with minimal fuss
and very few limitations.

**NOTE:**

This library is not meant to supplant streadway/amqp (Roger, Rabbit is built on
top of it!), but an extension with quality of life features. Roger, Rabbit would not
be possible without the amazing groundwork laid down by streadway/amqp.

Goals
-----

The goals of the Roger, Rabbit package are as follows:

- **Offer a Drop-in Replacement for streadway/amqp**: APIs may be extended (adding
  fields to `amqp.Config` or additional methods to `*amqp.Channel`, for instance) but
  must not break existing code unless absolutely necessary.

- **Add as few Additional Error Paths as Possible**: Errors may be *extended* with
  additional information concerning disconnect scenarios, but new error type returns
  from `*Connection` or `*amqp.Channel` should be an absolute last resort.

- **Be Highly Extensible**: Roger, Rabbit seeks to offer a high degree of extensibility
  via features like middleware, in an effort to reduce the balkanization of amqp client
  solutions.

Current Limitations & Warnings
------------------------------

- **Performance**: Roger, Rabbit has not been extensively benchmarked against
  `streadway/amqp`. To see preliminary benchmarks, take a look at the next section.

- **Transaction Support**: Roger, Rabbit does not currently support amqp Transactions,
  as the author does not use them. Draft PR's with possible implementations are welcome!

- **Reliability**: While the author uses this library in production, it is still early
  days, and more battle-testing will be needed before this library is promoted to
  version 1.0. PR's are welcome for Bug Fixes, code coverage, or new features.

Benchmarks
----------

Because of Roger, Rabbit's middleware-driven design, some overhead is expected vs
streadway proper. However, initial benchmarks are promising, and show only minimal
impact. For most applications, the overhead cost is likely worth the cost for ease of
development and flexibility.

Still, if absolute peak throughput is critical to an application, a less general and
more tailored approach may be warranted.

Benchmarks can be found in `./amqp/benchmark_test.go`.

Machine: Intel(R) Core(TM) i9-8950HK CPU @ 2.90GHz


| OPERATION          | LIB  | EXECUTIONS  |     NS/OP  |  COMPARISON
| -------------------|------|-------------|------------|------------
| QueueInspect       | sw   |     2,838   |  812,594   |         --
|                    | rr   |     2,470   |  813,269   |      +0.1%
| Publish            | sw   |    7,4559   |   28,882   |         --
|                    | rr   |    7,0665   |   30,031   |      +4.0%
| Publish & Confirm  | sw   |    3,4528   |   59,703   |         --
|                    | rr   |    3,5481   |   62,198   |      +4.2%


The above numbers were calculated by running each benchmark 4 times, then taking the
fastest result for each library.

The benchmarks were run with the following command:

Acknowledgements
----------------

This library is built on top of [streadway/amqp](https://github.com/streadway/amqp) and
would not be possible without such an amazing foundation.

Prerequisites
-------------

Golang 1.6+, Python 3.6+
