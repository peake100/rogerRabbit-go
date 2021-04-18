Project
=======

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
  `streadway/amqp`_. To see preliminary benchmarks, take a look at the next section.

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

.. table:: Machine: Intel(R) Core(TM) i9-8950HK CPU @ 2.90GHz || Calculated by taking the fastest of four runs.

    =================  ====  =========== ==========   ===========
            OPERATION  LIB   EXECUTIONS       NS/OP    COMPARISON
    =================  ====  =========== ==========   ===========
         QueueInspect   sw        2,838     812,594            --
                   --   rr        2,470     813,269         +0.1%
              Publish   sw       7,4559      28,882            --
                   --   rr       7,0665      30,031         +4.0%
    Publish & Confirm   sw       3,4528      59,703            --
                   --   rr       3,5481      62,198         +4.2%
    Consume (QoS 100)   sw       75,433      27,206            --
                   --   rr       73,957      29,846         +9.7%
    =================  ====  =========== ==========   ===========

Run with the following command:

.. code-block:: shell

    go test -p 1 -count 4 -bench=Comparison -run=NoTests -benchtime=2s ./...

.. _streadway/amqp: https://github.com/streadway/amqp