# Roger, Rabbit

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

## Acknowledgements

This library is built on top of [streadway/amqp](https://github.com/streadway/amqp) and
would not be possible without such an amazing foundation.

## Getting Started
For quickstart and full API documentation:
[read the docs](https://peake100.github.io/rogerRabbit-go/).

For library development guide, 
[read the docs](https://illuscio-dev.github.io/islelib-go/).

### Prerequisites

Golang 1.3+, Python 3.6+

## Authors

* **Billy Peake** - *Initial work*
