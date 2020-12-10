package amqp

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/dataModels"
	"github.com/peake100/rogerRabbit-go/amqp/defaultMiddlewares"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"testing"
)

// Implements transport for *streadway.Channel.
type transportChannel struct {
	// The current, underlying channel object.
	*streadway.Channel

	// The roger connection we will use to re-establish dropped channels.
	rogerConn *Connection

	// Holds all registered handlers.
	handlers           *channelHandlers
	defaultMiddlewares *ChannelTestingDefaultMiddlewares

	// Sync object for communicating with relays
	relaySync channelRelaySync

	// Logger for the channel transport
	logger zerolog.Logger
}

func (transport *transportChannel) cleanup() error {
	// Release this lock so event processors can close.
	defer transport.relaySync.shared.runSetup.Done()
	return nil
}

func (transport *transportChannel) tryReconnect(ctx context.Context) error {
	// Wait for all event processors processing events from the previous channel to be
	// ready.
	logger := transport.logger.With().Str("STATUS", "CONNECTING").Logger()
	debugEnabled := logger.Debug().Enabled()

	if debugEnabled {
		logger.Debug().Msg("waiting for event relays to finish")
	}
	transport.relaySync.WaitForRelayLegComplete()

	if debugEnabled {
		logger.Debug().Msg("getting new channel")
	}

	// Invoke all our reconnection middleware and reconnect the channel.
	channel, err := transport.handlers.reconnect(ctx, logger)
	if err != nil {
		return err
	}

	// Set up the new channel
	transport.Channel = channel
	// Allow the relays to advance to the setup stage.
	if debugEnabled {
		logger.Debug().Msg("advancing event relays to setup")
	}
	transport.relaySync.StartSetup()
	transport.relaySync.WaitForRelaySetupComplete()

	if debugEnabled {
		logger.Debug().Msg("restarting event relay processing")
	}
	transport.relaySync.StartRelays()

	return nil
}

// Channel is a drop-in replacement for streadway/amqp.Channel, with the exception
// that it automatically recovers from unexpected disconnections.
//
// Unless otherwise noted at the beginning of their descriptions, all methods work
// exactly as their streadway counterparts, but will automatically re-attempt on
// ErrClosed errors. All other errors will be returned as normal. Descriptions have
// been copy-pasted from the streadway library for convenience.
//
// Unlike streadway/amqp.Channel, this channel will remain open when an error is
// returned. Under the hood, the old, closed, channel will be replaced with a new,
// fresh, one -- so operation will continue as normal.
type Channel struct {
	// A transport object that contains our current underlying connection.
	transportChannel *transportChannel

	// Manages the lifetime of the connection: such as automatic reconnects, connection
	// status events, and closing.
	*transportManager
}

/*
ROGER NOTE: Tags will bew continuous, even in the event of a re-connect. The channel
will take care of matching up caller-facing delivery tags to the current channel's
underlying tag.

--

Confirm puts this channel into confirm mode so that the client can ensure all
publishings have successfully been received by the server.  After entering this
mode, the server will send a basic.ack or basic.nack message with the deliver
tag set to a 1 based incremental index corresponding to every publishing
received after the this method returns.

Add a listener to Channel.NotifyPublish to respond to the Confirmations. If
Channel.NotifyPublish is not called, the Confirmations will be silently
ignored.

The order of acknowledgments is not bound to the order of deliveries.

Ack and Nack confirmations will arrive at some point in the future.

Unroutable mandatory or immediate messages are acknowledged immediately after
any Channel.NotifyReturn listeners have been notified.  Other messages are
acknowledged when all queues that should have the message routed to them have
either received acknowledgment of delivery or have enqueued the message,
persisting the message if necessary.

When NoWait is true, the client will not wait for a response.  A channel
exception could occur if the server does not support this method.

*/
func (channel *Channel) Confirm(noWait bool) error {
	args := &amqpMiddleware.ArgsConfirms{NoWait: noWait}

	op := func() error {
		return channel.transportChannel.handlers.confirm(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, op, true)
}

/*
Qos controls how many messages or how many bytes the server will try to keep on
the network for consumers before receiving delivery acks.  The intent of Qos is
to make sure the network buffers stay full between the server and client.

With a prefetch publishCount greater than zero, the server will deliver that many
messages to consumers before acknowledgments are received.  The server ignores
this option when consumers are started with noAck because no acknowledgments
are expected or sent.

With a prefetch size greater than zero, the server will try to keep at least
that many bytes of deliveries flushed to the network before receiving
acknowledgments from the consumers.  This option is ignored when consumers are
started with noAck.

When global is true, these Qos settings apply to all existing and future
consumers on all channels on the same connection.  When false, the Channel.Qos
settings will apply to all existing and future consumers on this channel.

Please see the RabbitMQ Consumer Prefetch documentation for an explanation of
how the global flag is implemented in RabbitMQ, as it differs from the
AMQP 0.9.1 specification in that global Qos settings are limited in scope to
channels, not connections (https://www.rabbitmq.com/consumer-prefetch.html).

To get round-robin behavior between consumers consuming from the same queue on
different connections, set the prefetch publishCount to 1, and the next available
message on the server will be delivered to the next available consumer.

If your consumer work time is reasonably consistent and not much greater
than two times your network round trip time, you will see significant
throughput improvements starting with a prefetch publishCount of 2 or slightly
greater as described by benchmarks on RabbitMQ.

http://www.rabbitmq.com/blog/2012/04/25/rabbitmq-performance-measurements-part-2/
*/
func (channel *Channel) Qos(prefetchCount, prefetchSize int, global bool) error {
	args := &amqpMiddleware.ArgsQoS{
		PrefetchCount: prefetchCount,
		PrefetchSize:  prefetchSize,
		Global:        global,
	}

	op := func() error {
		return channel.transportChannel.handlers.qos(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, op, true)
}

/*
Flow pauses the delivery of messages to consumers on this channel.  Channels
are opened with flow control active, to open a channel with paused
deliveries immediately call this method with `false` after calling
Connection.Channel.

When active is `false`, this method asks the server to temporarily pause deliveries
until called again with active as `true`.

Channel.Get methods will not be affected by flow control.

This method is not intended to act as window control.  Use Channel.Qos to limit
the number of unacknowledged messages or bytes in flight instead.

The server may also send us flow methods to throttle our publishings.  A well
behaving publishing client should add a listener with Channel.NotifyFlow and
pause its publishings when `false` is sent on that channel.

Note: RabbitMQ prefers to use TCP push back to control flow for all channels on
a connection, so under high volume scenarios, it's wise to open separate
Connections for publishings and deliveries.

*/
func (channel *Channel) Flow(active bool) error {
	args := &amqpMiddleware.ArgsFlow{
		Active: active,
	}

	op := func() error {
		return channel.transportChannel.handlers.flow(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, op, true)
}

/*
ROGER NOTE: Queues declared with a roger channel will be automatically re-declared
upon channel disconnect recovery. Calling channel.QueueDelete() will remove the queue
from the list of queues to be re-declared in case of a disconnect.

This may cases where queues deleted by other producers / consumers are
automatically re-declared. Future updates will introduce more control over when and
how queues are re-declared on reconnection.

--

QueueDeclare declares a queue to hold messages and deliver to consumers.
Declaring creates a queue if it doesn't already exist, or ensures that an
existing queue matches the same parameters.

Every queue declared gets a default binding to the empty exchange "" which has
the type "direct" with the routing Key matching the queue's Name.  With this
default binding, it is possible to publish messages that route directly to
this queue by publishing to "" with the routing Key of the queue Name.

  QueueDeclare("alerts", true, false, false, false, nil)
  Publish("", "alerts", false, false, Publishing{Body: []byte("...")})

  Delivery       Exchange  Key       Queue
  -----------------------------------------------
  Key: alerts -> ""     -> alerts -> alerts

The queue Name may be empty, in which case the server will generate a unique Name
which will be returned in the Name field of Queue struct.

Durable and Non-Auto-Deleted queues will survive server restarts and remain
when there are no remaining consumers or bindings. Persistent publishings will
be restored in this queue on server restart.  These queues are only able to be
bound to Durable exchanges.

Non-Durable and Auto-Deleted queues will not be redeclared on server restart
and will be deleted by the server after a short time when the last consumer is
canceled or the last consumer's channel is closed.  Queues with this lifetime
can also be deleted normally with QueueDelete.  These Durable queues can only
be bound to non-Durable exchanges.

Non-Durable and Non-Auto-Deleted queues will remain declared as long as the
server is running regardless of how many consumers.  This lifetime is useful
for temporary topologies that may have long delays between consumer activity.
These queues can only be bound to non-Durable exchanges.

Durable and Auto-Deleted queues will be restored on server restart, but without
active consumers will not survive and be removed.  This Lifetime is unlikely
to be useful.

Exclusive queues are only accessible by the connection that declares them and
will be deleted when the connection closes.  Channels on other connections
will receive an error when attempting  to declare, bind, consume, purge or
delete a queue with the same Name.

When NoWait is true, the queue will assume to be declared on the server.  A
channel exception will arrive if the conditions are met for existing queues
or attempting to modify an existing queue from a different connection.

When the error return value is not nil, you can assume the queue could not be
declared with these parameters, and the channel will be closed.

*/
func (channel *Channel) QueueDeclare(
	name string,
	durable bool,
	autoDelete bool,
	exclusive bool,
	noWait bool,
	args streadway.Table,
) (queue Queue, err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	// We need to remember to re-declare this queue on reconnectMiddleware
	queueArgs := &amqpMiddleware.ArgsQueueDeclare{
		Name:       name,
		Durable:    durable,
		AutoDelete: autoDelete,
		Exclusive:  exclusive,
		// We are always going to wait on re-declares, but we should save the
		// NoWait value for posterity.
		NoWait: noWait,
		// Copy the Args so if the user mutates them for a future call we have
		// an un-changed version of the originals.
		Args: copyTable(args),
	}

	// Wrap the hook runner in a closure.
	operation := func() error {
		var opErr error
		queue, opErr = channel.transportChannel.handlers.queueDeclare(queueArgs)
		return opErr
	}

	// Hand off to the retry operation method.
	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return queue, err
}

/*

QueueDeclarePassive is functionally and parametrically equivalent to
QueueDeclare, except that it sets the "passive" attribute to true. A passive
queue is assumed by RabbitMQ to already exist, and attempting to connect to a
non-existent queue will cause RabbitMQ to throw an exception. This function
can be used to test for the existence of a queue.

*/
func (channel *Channel) QueueDeclarePassive(
	name string, durable, autoDelete, exclusive, noWait bool, args Table,
) (queue Queue, err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		queue, opErr = channel.transportChannel.QueueDeclarePassive(
			name, durable, autoDelete, exclusive, noWait, args,
		)
		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return queue, err
}

/*
QueueInspect passively declares a queue by Name to inspect the current message
publishCount and consumer publishCount.

Use this method to check how many messages ready for delivery reside in the queue,
how many consumers are receiving deliveries, and whether a queue by this
Name already exists.

If the queue by this Name exists, use Channel.QueueDeclare check if it is
declared with specific parameters.

If a queue by this Name does not exist, an error will be returned and the
channel will be closed.

*/
func (channel *Channel) QueueInspect(name string) (queue Queue, err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		queue, opErr = channel.transportChannel.QueueInspect(name)
		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return queue, err
}

/*
QueueBind binds an exchange to a queue so that publishings to the exchange will
be routed to the queue when the publishing routing Key matches the binding
routing Key.

  QueueBind("pagers", "alert", "log", false, nil)
  QueueBind("emails", "info", "log", false, nil)

  Delivery       Exchange  Key       Queue
  -----------------------------------------------
  Key: alert --> log ----> alert --> pagers
  Key: info ---> log ----> info ---> emails
  Key: debug --> log       (none)    (dropped)

If a binding with the same Key and arguments already exists between the
exchange and queue, the attempt to rebind will be ignored and the existing
binding will be retained.

In the case that multiple bindings may cause the message to be routed to the
same queue, the server will only route the publishing once.  This is possible
with topic exchanges.

  QueueBind("pagers", "alert", "amq.topic", false, nil)
  QueueBind("emails", "info", "amq.topic", false, nil)
  QueueBind("emails", "#", "amq.topic", false, nil) // match everything

  Delivery       Exchange        Key       Queue
  -----------------------------------------------
  Key: alert --> amq.topic ----> alert --> pagers
  Key: info ---> amq.topic ----> # ------> emails
                           \---> info ---/
  Key: debug --> amq.topic ----> # ------> emails

It is only possible to bind a Durable queue to a Durable exchange regardless of
whether the queue or exchange is auto-deleted.  Bindings between Durable queues
and exchanges will also be restored on server restart.

If the binding could not complete, an error will be returned and the channel
will be closed.

When NoWait is false and the queue could not be bound, the channel will be
closed with an error.

*/
func (channel *Channel) QueueBind(
	name, key, exchange string, noWait bool, args Table,
) error {
	bindArgs := &amqpMiddleware.ArgsQueueBind{
		Name:     name,
		Key:      key,
		Exchange: exchange,
		NoWait:   noWait,
		// Copy the arg table so if the caller re-uses it we preserve the original
		// values.
		Args: copyTable(args),
	}

	// Wrap the method handler
	operation := func() error {
		return channel.transportChannel.handlers.queueBind(bindArgs)
	}

	err := channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
QueueUnbind removes a binding between an exchange and queue matching the Key and
arguments.

It is possible to send and empty string for the exchange Name which means to
unbind the queue from the default exchange.

*/
func (channel *Channel) QueueUnbind(name, key, exchange string, args Table) error {
	unbindArgs := &amqpMiddleware.ArgsQueueUnbind{
		Name:     name,
		Key:      key,
		Exchange: exchange,
		Args:     copyTable(args),
	}

	operation := func() error {
		return channel.transportChannel.handlers.queueUnbind(unbindArgs)
	}

	err := channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
QueuePurge removes all messages from the named queue which are not waiting to
be acknowledged.  Messages that have been delivered but have not yet been
acknowledged will not be removed.

When successful, returns the number of messages purged.

If NoWait is true, do not wait for the server response and the number of
messages purged will not be meaningful.
*/
func (channel *Channel) QueuePurge(name string, noWait bool) (
	count int, err error,
) {
	operation := func() error {
		var opErr error
		count, opErr = channel.transportChannel.QueuePurge(name, noWait)
		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return count, err
}

/*
QueueDelete removes the queue from the server including all bindings then
purges the messages based on server configuration, returning the number of
messages purged.

When ifUnused is true, the queue will not be deleted if there are any
consumers on the queue.  If there are consumers, an error will be returned and
the channel will be closed.

When ifEmpty is true, the queue will not be deleted if there are any messages
remaining on the queue.  If there are messages, an error will be returned and
the channel will be closed.

When NoWait is true, the queue will be deleted without waiting for a response
from the server.  The purged message publishCount will not be meaningful. If the queue
could not be deleted, a channel exception will be raised and the channel will
be closed.

*/
func (channel *Channel) QueueDelete(
	name string, ifUnused, ifEmpty, noWait bool,
) (count int, err error) {
	deleteArgs := &amqpMiddleware.ArgsQueueDelete{
		Name:     name,
		IfUnused: ifUnused,
		IfEmpty:  ifEmpty,
		NoWait:   noWait,
	}

	operation := func() error {
		var opErr error
		count, opErr = channel.transportChannel.handlers.queueDelete(deleteArgs)
		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return count, err
}

/*
ROGER NOTE: Exchanges declared with a roger channel will be automatically re-declared
upon channel disconnect recovery. Calling channel.ExchangeDelete() will remove the
exchange from the list of exchanges to be re-declared in case of a disconnect.

This may cases where exchanges deleted by other producers / consumers are
automatically re-declared. Future updates will introduce more control over when and
how exchanges are re-declared on reconnection.

--

ExchangeDeclare declares an exchange on the server. If the exchange does not
already exist, the server will create it.  If the exchange exists, the server
verifies that it is of the provided type, durability and auto-delete flags.

Errors returned from this method will close the channel.

Exchange names starting with "amq." are reserved for pre-declared and
standardized exchanges. The client MAY declare an exchange starting with
"amq." if the passive option is set, or the exchange already exists.  Names can
consist of a non-empty sequence of letters, digits, hyphen, underscore,
period, or colon.

Each exchange belongs to one of a set of exchange kinds/types implemented by
the server. The exchange types define the functionality of the exchange - i.e.
how messages are routed through it. Once an exchange is declared, its type
cannot be changed.  The common types are "direct", "fanout", "topic" and
"headers".

Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
declared when there are no remaining bindings.  This is the best lifetime for
long-lived exchange configurations like stable routes and default exchanges.

Non-Durable and Auto-Deleted exchanges will be deleted when there are no
remaining bindings and not restored on server restart.  This lifetime is
useful for temporary topologies that should not pollute the virtual host on
failure or after the consumers have completed.

Non-Durable and Non-Auto-deleted exchanges will remain as long as the server is
running including when there are no remaining bindings.  This is useful for
temporary topologies that may have long delays between bindings.

Durable and Auto-Deleted exchanges will survive server restarts and will be
removed before and after server restarts when there are no remaining bindings.
These exchanges are useful for robust temporary topologies or when you require
binding Durable queues to auto-deleted exchanges.

Note: RabbitMQ declares the default exchange types like 'amq.fanout' as
Durable, so queues that bind to these pre-declared exchanges must also be
Durable.

Exchanges declared as `internal` do not accept accept publishings. Internal
exchanges are useful when you wish to implement inter-exchange topologies
that should not be exposed to users of the broker.

When NoWait is true, declare without waiting for a confirmation from the server.
The channel may be closed as a result of an error.  Add a NotifyClose listener
to respond to any exceptions.

Optional amqp.Table of arguments that are specific to the server's implementation of
the exchange can be sent for exchange types that require extra parameters.
*/
func (channel *Channel) ExchangeDeclare(
	name, kind string, durable, autoDelete, internal, noWait bool, args Table,
) (err error) {
	exchangeArgs := &amqpMiddleware.ArgsExchangeDeclare{
		Name:       name,
		Kind:       kind,
		Durable:    durable,
		AutoDelete: autoDelete,
		Internal:   internal,
		// We will always use wait on re-establishments, but preserve the original
		// setting here for posterity.
		NoWait: noWait,
		// Copy the table so if the caller re-uses it we dont have it mutated
		// between re-declarations.
		Args: copyTable(args),
	}

	operation := func() error {
		return channel.transportChannel.handlers.exchangeDeclare(exchangeArgs)
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*

ExchangeDeclarePassive is functionally and parametrically equivalent to
ExchangeDeclare, except that it sets the "passive" attribute to true. A passive
exchange is assumed by RabbitMQ to already exist, and attempting to connect to a
non-existent exchange will cause RabbitMQ to throw an exception. This function
can be used to detect the existence of an exchange.

*/
func (channel *Channel) ExchangeDeclarePassive(
	name, kind string, durable, autoDelete, internal, noWait bool, args Table,
) (err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.ExchangeDeclarePassive(
			name, kind, durable, autoDelete, internal, noWait, args,
		)
		if opErr != nil {
			return opErr
		}

		return nil
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
ExchangeDelete removes the named exchange from the server. When an exchange is
deleted all queue bindings on the exchange are also deleted.  If this exchange
does not exist, the channel will be closed with an error.

When ifUnused is true, the server will only delete the exchange if it has no queue
bindings.  If the exchange has queue bindings the server does not delete it
but close the channel with an exception instead.  Set this to true if you are
not the sole owner of the exchange.

When NoWait is true, do not wait for a server confirmation that the exchange has
been deleted.  Failing to delete the channel could close the channel.  Add a
NotifyClose listener to respond to these channel exceptions.
*/
func (channel *Channel) ExchangeDelete(
	name string, ifUnused, noWait bool,
) (err error) {
	deleteArgs := &amqpMiddleware.ArgsExchangeDelete{
		Name:     name,
		IfUnused: ifUnused,
		NoWait:   noWait,
	}

	operation := func() error {
		return channel.transportChannel.handlers.exchangeDelete(deleteArgs)
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
ExchangeBind binds an exchange to another exchange to create inter-exchange
routing topologies on the server.  This can decouple the private topology and
routing exchanges from exchanges intended solely for publishing endpoints.

Binding two exchanges with identical arguments will not create duplicate
bindings.

Binding one exchange to another with multiple bindings will only deliver a
message once.  For example if you bind your exchange to `amq.fanout` with two
different binding keys, only a single message will be delivered to your
exchange even though multiple bindings will match.

Given a message delivered to the source exchange, the message will be forwarded
to the destination exchange when the routing Key is matched.

  ExchangeBind("sell", "MSFT", "trade", false, nil)
  ExchangeBind("buy", "AAPL", "trade", false, nil)

  Delivery       Source      Key      Destination
  example        exchange             exchange
  -----------------------------------------------
  Key: AAPL  --> trade ----> MSFT     sell
                       \---> AAPL --> buy

When NoWait is true, do not wait for the server to confirm the binding.  If any
error occurs the channel will be closed.  Add a listener to NotifyClose to
handle these errors.

Optional arguments specific to the exchanges bound can also be specified.
*/
func (channel *Channel) ExchangeBind(
	destination, key, source string, noWait bool, args Table,
) (err error) {
	bindArgs := &amqpMiddleware.ArgsExchangeBind{
		Destination: destination,
		Key:         key,
		Source:      source,
		NoWait:      noWait,
		Args:        copyTable(args),
	}

	operation := func() error {
		return channel.transportChannel.handlers.exchangeBind(bindArgs)
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
ExchangeUnbind unbinds the destination exchange from the source exchange on the
server by removing the routing Key between them.  This is the inverse of
ExchangeBind.  If the binding does not currently exist, an error will be
returned.

When NoWait is true, do not wait for the server to confirm the deletion of the
binding.  If any error occurs the channel will be closed.  Add a listener to
NotifyClose to handle these errors.

Optional arguments that are specific to the type of exchanges bound can also be
provided.  These must match the same arguments specified in ExchangeBind to
identify the binding.
*/
func (channel *Channel) ExchangeUnbind(
	destination, key, source string, noWait bool, args Table,
) (err error) {
	unbindArgs := &amqpMiddleware.ArgsExchangeUnbind{
		Destination: destination,
		Key:         key,
		Source:      source,
		NoWait:      noWait,
		Args:        copyTable(args),
	}

	operation := func() error {
		return channel.transportChannel.handlers.exchangeUnbind(unbindArgs)
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
Publish sends a Publishing from the client to an exchange on the server.

When you want a single message to be delivered to a single queue, you can
publish to the default exchange with the routingKey of the queue Name.  This is
because every declared queue gets an implicit route to the default exchange.

Since publishings are asynchronous, any undeliverable message will get returned
by the server.  Add a listener with Channel.NotifyReturn to handle any
undeliverable message when calling publish with either the mandatory or
immediate parameters as true.

Publishings can be undeliverable when the mandatory flag is true and no queue is
bound that matches the routing Key, or when the immediate flag is true and no
consumer on the matched queue is ready to accept the delivery.

This can return an error when the channel, connection or socket is closed.  The
error or lack of an error does not indicate whether the server has received this
publishing.

It is possible for publishing to not reach the broker if the underlying socket
is shut down without pending publishing packets being flushed from the kernel
buffers.  The easy way of making it probable that all publishings reach the
server is to always call Connection.Close before terminating your publishing
application.  The way to ensure that all publishings reach the server is to add
a listener to Channel.NotifyPublish and put the channel in confirm mode with
Channel.Confirm.  Publishing delivery tags and their corresponding
confirmations start at 1.  Exit when all publishings are confirmed.

When Publish does not return an error and the channel is in confirm mode, the
internal counter for DeliveryTags with the first confirmation starts at 1.

*/
func (channel *Channel) Publish(
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	msg Publishing,
) (err error) {
	args := &amqpMiddleware.ArgsPublish{
		Exchange:  exchange,
		Key:       key,
		Mandatory: mandatory,
		Immediate: immediate,
		Msg:       msg,
	}

	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		return channel.transportChannel.handlers.publish(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, operation, true)
}

/*
Get synchronously receives a single Delivery from the head of a queue from the
server to the client.  In almost all cases, using Channel.Consume will be
preferred.

If there was a delivery waiting on the queue and that delivery was received, the
second return value will be true.  If there was no delivery waiting or an error
occurred, the ok bool will be false.

All deliveries must be acknowledged including those from Channel.Get.  Call
Delivery.Ack on the returned delivery when you have fully processed this
delivery.

When autoAck is true, the server will automatically acknowledge this message so
you don't have to.  But if you are unable to fully process this message before
the channel or connection is closed, the message will not get requeued.

*/
func (channel *Channel) Get(
	queue string,
	autoAck bool,
) (msg dataModels.Delivery, ok bool, err error) {
	args := &amqpMiddleware.ArgsGet{
		Queue:   queue,
		AutoAck: autoAck,
	}

	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		msg, ok, opErr = channel.transportChannel.handlers.get(args)
		if opErr != nil {
			return opErr
		}

		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return msg, ok, err
}

/*
ROGER NOTE: Unlike the normal consume method, re-connections are handled automatically
on channel errors. Delivery tags will appear un-interrupted to the consumer, and the
roger channel will track the lineup of caller-facing delivery tags to the delivery
tags of the current underlying channel.

--

Consume immediately starts delivering queued messages.

Begin receiving on the returned chan Delivery before any other operation on the
Connection or Channel.

Continues deliveries to the returned chan Delivery until Channel.Cancel,
Connection.Close, Channel.Close, or an AMQP exception occurs.  Consumers must
range over the chan to ensure all deliveries are received.  Unreceived
deliveries will block all methods on the same connection.

All deliveries in AMQP must be acknowledged.  It is expected of the consumer to
call Delivery.Ack after it has successfully processed the delivery.  If the
consumer is cancelled or the channel or connection is closed any unacknowledged
deliveries will be requeued at the end of the same queue.

The consumer is identified by a string that is unique and scoped for all
consumers on this channel.  If you wish to eventually cancel the consumer, use
the same non-empty identifier in Channel.Cancel.  An empty string will cause
the library to generate a unique identity.  The consumer identity will be
included in every Delivery in the ConsumerTag field

When autoAck (also known as noAck) is true, the server will acknowledge
deliveries to this consumer prior to writing the delivery to the network.  When
autoAck is true, the consumer should not call Delivery.Ack. Automatically
acknowledging deliveries means that some deliveries may get lost if the
consumer is unable to process them after the server delivers them.
See http://www.rabbitmq.com/confirms.html for more details.

When Exclusive is true, the server will ensure that this is the sole consumer
from this queue. When Exclusive is false, the server will fairly distribute
deliveries across multiple consumers.

The noLocal flag is not supported by RabbitMQ.

It's advisable to use separate connections for
Channel.Publish and Channel.Consume so not to have TCP pushback on publishing
affect the ability to consume messages, so this parameter is here mostly for
completeness.

When NoWait is true, do not wait for the server to confirm the request and
immediately begin deliveries.  If it is not possible to consume, a channel
exception will be raised and the channel will be closed.

Optional arguments can be provided that have specific semantics for the queue
or server.

Inflight messages, limited by Channel.Qos will be buffered until received from
the returned chan.

When the Channel or Connection is closed, all buffered and inflight messages will
be dropped.

When the consumer tag is cancelled, all inflight messages will be delivered until
the returned chan is closed.

*/
func (channel *Channel) Consume(
	queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args Table,
) (deliveryChan <-chan dataModels.Delivery, err error) {
	consumeArgs := &consumeArgs{
		queue:     queue,
		consumer:  consumer,
		autoAck:   autoAck,
		exclusive: exclusive,
		noLocal:   noLocal,
		noWait:    noWait,
		args:      args,
		// Make a buffered channel so we don't cause latency from waiting for queues to
		// be ready
		callerDeliveryChan: make(chan dataModels.Delivery, 16),
	}
	deliveryChan = consumeArgs.callerDeliveryChan

	// Create our consumer relay
	consumeRelay := newConsumeRelay(
		consumeArgs,
		channel,
		channel.transportChannel.handlers.consumeEventMiddleware,
	)

	// Pass it to our relay handler.
	err = channel.setupAndLaunchEventRelay(consumeRelay)
	if err != nil {
		return nil, err
	}

	// If no error, pass the channel back to the caller
	return deliveryChan, nil
}

/*
ROGER NOTE: If a tag (or a tag range when acking multiple tags) is from a previously
disconnected channel, a ErrCantAcknowledgeOrphans will be returned, which is a new error
type specific to the Roger implementation.

When acking multiple tags, it is possible that some of the tags will be from a closed
underlying channel, and therefore be orphaned, and some will be from the current
channel, and therefore be successful. In such cases, the ErrCantAcknowledgeOrphans will
still be returned, and can be checked for which tag ranges could not be acked.

--

Ack acknowledges a delivery by its delivery tag when having been consumed with
Channel.Consume or Channel.Get.

Ack acknowledges all message received prior to the delivery tag when multiple
is true.

See also Delivery.Ack
*/
func (channel *Channel) Ack(tag uint64, multiple bool) error {
	args := &amqpMiddleware.ArgsAck{
		Tag:      tag,
		Multiple: multiple,
	}

	operation := func() error {
		return channel.transportChannel.handlers.ack(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, operation, true)
}

/*
ROGER NOTE: If a tag (or a tag range when nacking multiple tags) is from a previously
disconnected channel, a ErrCantAcknowledgeOrphans will be returned, which is a new error
type specific to the Roger implementation.

When nacking multiple tags, it is possible that some of the tags will be from a closed
underlying channel, and therefore be orphaned, and some will be from the current
channel, and therefore be successful. In such cases, the ErrCantAcknowledgeOrphans will
still be returned, and can be checked for which tag ranges could not be nacked.

--

Nack negatively acknowledges a delivery by its delivery tag.  Prefer this
method to notify the server that you were not able to process this delivery and
it must be redelivered or dropped.

See also Delivery.Nack
*/
func (channel *Channel) Nack(tag uint64, multiple bool, requeue bool) error {
	args := &amqpMiddleware.ArgsNack{
		Tag:      tag,
		Multiple: multiple,
		Requeue:  requeue,
	}

	operation := func() error {
		return channel.transportChannel.handlers.nack(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, operation, true)
}

/*
ROGER NOTE: If a tag (or a tag range when rejecting multiple tags) is from a previously
disconnected channel, a ErrCantAcknowledgeOrphans will be returned, which is a new error
type specific to the Roger implementation.

When nacking multiple tags, it is possible that some of the tags will be from a closed
underlying channel, and therefore be orphaned, and some will be from the current
channel, and therefore be successful. In such cases, the ErrCantAcknowledgeOrphans will
still be returned, and can be checked for which tag ranges could not be rejected.

--

Reject negatively acknowledges a delivery by its delivery tag.  Prefer Nack
over Reject when communicating with a RabbitMQ server because you can Nack
multiple messages, reducing the amount of protocol messages to exchange.

See also Delivery.Reject
*/
func (channel *Channel) Reject(tag uint64, requeue bool) error {
	args := &amqpMiddleware.ArgsReject{
		Tag:     tag,
		Requeue: requeue,
	}

	operation := func() error {
		return channel.transportChannel.handlers.reject(args)
	}

	return channel.retryOperationOnClosed(channel.ctx, operation, true)
}

/*
ROGER NOTE: It is possible that if a channel is disconnected unexpectedly, there may
have been confirmations in flight that did not reach the client. In cases where a
channel connection is re-established, ant missing delivery tags will be reported nacked,
but an additional DisconnectOrphan field will be set to `true`. It is possible that
such messages DID reach the broker, but the Ack messages were lost in the disconnect
event.

It's also possible that an orphan is caused from a problem with publishing the message
in the first place. For instance, publishing with the ``immediate`` flag set to false
if the broker does not support it, or if a queue was not declared correctly. If you are
getting a lot of orphaned messages, make sure to check what disconnect errors you are
receiving.

--

NotifyPublish registers a listener for reliable publishing. Receives from this
chan for every publish after Channel.Confirm will be in order starting with
DeliveryTag 1.

There will be one and only one Confirmation Publishing starting with the
delivery tag of 1 and progressing sequentially until the total number of
Publishings have been seen by the server.

Acknowledgments will be received in the order of delivery from the
NotifyPublish channels even if the server acknowledges them out of order.

The listener chan will be closed when the Channel is closed.

The capacity of the chan Confirmation must be at least as large as the
number of outstanding publishings.  Not having enough buffered chans will
create a deadlock if you attempt to perform other operations on the Connection
or Channel while confirms are in-flight.

It's advisable to wait for all Confirmations to arrive before calling
Channel.Close() or Connection.Close().

*/
func (channel *Channel) NotifyPublish(
	confirm chan dataModels.Confirmation,
) chan dataModels.Confirmation {
	// Setup and launch the event relay that will handle these events across multiple
	// connections.
	args := &amqpMiddleware.ArgsNotifyPublish{Confirm: confirm}
	return channel.transportChannel.handlers.notifyPublish(args)
}

// Closes confirmation tag channels for NotifyConfirm and NotifyConfirmOrOrphaned.
func notifyConfirmCloseConfirmChannels(tagChannels ...chan uint64) {
mainLoop:
	// Iterate over the channels and close them. We'll need to make sure we don't close
	// the same channel twice.
	for i, thisChannel := range tagChannels {
		// Whenever we get a new channel, compare it against all previously closed
		// channels. If it is one of our previous channels, advance the main loop
		for i2 := 0; i2 < i; i2++ {
			previousChan := tagChannels[i2]
			if previousChan == thisChannel {
				continue mainLoop
			}
		}
		close(thisChannel)
	}
}

func notifyConfirmHandleAckAndNack(
	confirmation dataModels.Confirmation, ack, nack chan uint64, logger zerolog.Logger,
) {
	if confirmation.Ack {
		if logger.Debug().Enabled() {
			logger.Debug().
				Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
				Bool("ACK", confirmation.Ack).
				Str("CHANNEL", "ACK").
				Msg("ack confirmation sent")
		}
		ack <- confirmation.DeliveryTag
	} else {
		if logger.Debug().Enabled() {
			logger.Debug().
				Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
				Bool("ACK", confirmation.Ack).
				Str("CHANNEL", "NACK").
				Msg("nack confirmation sent")
		}
		nack <- confirmation.DeliveryTag
	}
}

/*
ROGER NOTE: the nack channel will receive both tags that were explicitly nacked by the
server AND tags that were orphaned due to a connection loss. If you wish to handle
Orphaned tags separately, use the new method NotifyConfirmOrOrphaned.

--

NotifyConfirm calls NotifyPublish and starts a goroutine sending
ordered Ack and Nack DeliveryTag to the respective channels.

For strict ordering, use NotifyPublish instead.
*/
func (channel *Channel) NotifyConfirm(
	ack, nack chan uint64,
) (chan uint64, chan uint64) {
	confirmsEvents := channel.NotifyPublish(
		make(chan dataModels.Confirmation, cap(ack)+cap(nack)),
	)
	logger := channel.logger.With().
		Str("EVENT_TYPE", "NOTIFY_CONFIRM").
		Logger()

	go func() {
		defer notifyConfirmCloseConfirmChannels(ack, nack)

		// range over confirmation events and place them in the ack and nack channels.
		for confirmation := range confirmsEvents {
			notifyConfirmHandleAckAndNack(confirmation, ack, nack, logger)
		}
	}()

	return ack, nack
}

/*
As NotifyConfirm, but with a third queue for delivery tags that were orphaned from
a disconnect, these tags are routed to the nack channel in NotifyConfirm.
*/
func (channel *Channel) NotifyConfirmOrOrphaned(
	ack, nack, orphaned chan uint64,
) (chan uint64, chan uint64, chan uint64) {
	confirmsEvents := channel.NotifyPublish(
		make(chan dataModels.Confirmation, cap(ack)+cap(nack)+cap(orphaned)),
	)
	logger := channel.logger.With().
		Str("EVENT_TYPE", "NOTIFY_CONFIRM_OR_ORPHAN").
		Logger()

	go channel.runNotifyConfirmOrOrphaned(ack, nack, orphaned, confirmsEvents, logger)

	return ack, nack, orphaned
}

func (channel *Channel) runNotifyConfirmOrOrphaned(
	ack, nack, orphaned chan uint64,
	confirmEvents <-chan dataModels.Confirmation,
	logger zerolog.Logger,
) {
	// Close channels on exit
	defer notifyConfirmCloseConfirmChannels(ack, nack, orphaned)

	// range over confirmation events and place them in the ack and nack channels.
	for confirmation := range confirmEvents {
		if confirmation.DisconnectOrphan {
			if logger.Debug().Enabled() {
				logger.Debug().
					Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
					Bool("ACK", confirmation.Ack).
					Str("CHANNEL", "ORPHANED").
					Msg("orphaned confirmation sent")
			}
			orphaned <- confirmation.DeliveryTag
		} else {
			notifyConfirmHandleAckAndNack(confirmation, ack, nack, logger)
		}
	}
}

/*
ROGER NOTE: Because this channel survives over unexpected server disconnects, it is
possible that returns in-flight to the client from the broker will be dropped, and
therefore will be missed. You can subscribe to disconnection events through

--

NotifyReturn registers a listener for basic.return methods.  These can be sent
from the server when a publish is undeliverable either from the mandatory or
immediate flags.

A return struct has a copy of the Publishing along with some error
information about why the publishing failed.

*/
func (channel *Channel) NotifyReturn(returns chan Return) chan Return {
	relay := &notifyReturnRelay{
		CallerReturns: returns,
	}

	err := channel.setupAndLaunchEventRelay(relay)
	if err != nil {
		close(returns)
		channel.logger.Err(err).Msg("error setting up notify return relay")
	}
	return returns
}

/*
NotifyCancel registers a listener for basic.cancel methods.  These can be sent
from the server when a queue is deleted or when consuming from a mirrored queue
where the master has just failed (and was moved to another node).

The subscription tag is returned to the listener.

*/
func (channel *Channel) NotifyCancel(cancellations chan string) chan string {
	relay := &notifyCancelRelay{
		CallerCancellations: cancellations,
	}

	err := channel.setupAndLaunchEventRelay(relay)
	if err != nil {
		close(cancellations)
		channel.logger.Err(err).Msg("error setting up notify cancel relay")
	}

	return cancellations
}

/*
ROGER NOTE: Flow notification will be received when an unexpected disconnection occurs.
If the broker disconnects, a ``false`` notification will be sent unless the last
notification from the broker was ``false``, and when the connection is re-acquired, a
``true`` notification will be sent before resuming relaying notification from the
broker. This means that NotifyFlow can be a useful tool for dealing with disconnects,
even when using RabbitMQ.

--

NotifyFlow registers a listener for basic.flow methods sent by the server.
When `false` is sent on one of the listener channels, all publishers should
pause until a `true` is sent.

The server may ask the producer to pause or restart the flow of Publishings
sent by on a channel. This is a simple flow-control mechanism that a server can
use to avoid overflowing its queues or otherwise finding itself receiving more
messages than it can process. Note that this method is not intended for window
control. It does not affect contents returned by basic.get-ok methods.

When a new channel is opened, it is active (flow is active). Some
applications assume that channels are inactive until started. To emulate
this behavior a client MAY open the channel, then pause it.

Publishers should respond to a flow messages as rapidly as possible and the
server may disconnect over producing channels that do not respect these
messages.

basic.flow-ok methods will always be returned to the server regardless of
the number of listeners there are.

To control the flow of deliveries from the server, use the Channel.Flow()
method instead.

Note: RabbitMQ will rather use TCP pushback on the network connection instead
of sending basic.flow.  This means that if a single channel is producing too
much on the same connection, all channels using that connection will suffer,
including acknowledgments from deliveries.  Use different Connections if you
desire to interleave consumers and producers in the same process to avoid your
basic.ack messages from getting rate limited with your basic.publish messages.

*/
func (channel *Channel) NotifyFlow(flowNotifications chan bool) chan bool {
	relay := &notifyFlowRelay{
		CallerFlow: flowNotifications,
		ChannelCtx: channel.ctx,
	}

	err := channel.setupAndLaunchEventRelay(relay)
	if err != nil {
		close(flowNotifications)
		channel.logger.Err(err).Msg("error setting up notify cancel relay")
	}

	return flowNotifications
}

// returns error we should pass to transaction panics until they are implemented.
func panicTransactionMessage(methodName string) error {
	return fmt.Errorf(
		"%v and other transaction methods not implemented, pull requests are"+
			" welcome for this functionality",
		methodName,
	)
}

/*
ROGER NOTE: transactions are not currently implemented, and calling any of the Tx
methods will result in a panic. The author of this library is not familiar with the
intricacies of amqp transactions, and how a channel in a transaction state should
behave over a disconnect.

PRs to add this functionality are welcome.

--

Tx puts the channel into transaction mode on the server.  All publishings and
acknowledgments following this method will be atomically committed or rolled
back for a single queue.  Call either Channel.TxCommit or Channel.TxRollback to
leave a this transaction and immediately start a new transaction.

The atomicity across multiple queues is not defined as queue declarations and
bindings are not included in the transaction.

The behavior of publishings that are delivered as mandatory or immediate while
the channel is in a transaction is not defined.

Once a channel has been put into transaction mode, it cannot be taken out of
transaction mode.  Use a different channel for non-transactional semantics.

*/
func (channel *Channel) Tx() error {
	panic(panicTransactionMessage("Tx"))
}

/*
ROGER NOTE: transactions are not currently implemented, and calling any of the Tx
methods will result in a panic. The author of this library is not familiar with the
intricacies of amqp transactions, and how a channel in a transaction state should
behave over a disconnect.

PRs to add this functionality are welcome.

--

TxCommit atomically commits all publishings and acknowledgments for a single
queue and immediately start a new transaction.

Calling this method without having called Channel.Tx is an error.

*/
func (channel *Channel) TxCommit() error {
	panic(panicTransactionMessage("TxCommit"))
}

/*
ROGER NOTE: transactions are not currently implemented, and calling any of the Tx
methods will result in a panic. The author of this library is not familiar with the
intricacies of amqp transactions, and how a channel in a transaction state should
behave over a disconnect.

PRs to add this functionality are welcome.

--

TxRollback atomically rolls back all publishings and acknowledgments for a
single queue and immediately start a new transaction.

Calling this method without having called Channel.Tx is an error.

*/
func (channel *Channel) TxRollback() error {
	panic(panicTransactionMessage("TxRollback"))
}

/*
Namespace object with methods for registering middleware. Middleware will be called
in the order it is registered.
*/
func (channel *Channel) Middleware() *channelHandlers {
	return channel.transportChannel.handlers
}

type ChannelTestingDefaultMiddlewares struct {
	QoS     *defaultMiddlewares.QoSMiddleware
	Flow    *defaultMiddlewares.FlowMiddleware
	Confirm *defaultMiddlewares.ConfirmsMiddleware

	RouteDeclaration *defaultMiddlewares.RouteDeclarationMiddleware

	PublishTags  *defaultMiddlewares.PublishTagsMiddleware
	DeliveryTags *defaultMiddlewares.DeliveryTagsMiddleware
}

// Holds testing information and methods for channels.
type ChannelTesting struct {
	*transportTesting

	channel *Channel
	// The middleware objects registered to the channel during channel creation.
	DefaultMiddlewares *ChannelTestingDefaultMiddlewares
}

// Returns a tester for the RogerConnection feeding this channel.
func (testing *ChannelTesting) ConnTest() *transportTesting {
	blocks := int32(0)

	return &transportTesting{
		t:       testing.t,
		manager: testing.channel.transportChannel.rogerConn.transportManager,
		blocks:  &blocks,
	}
}

// Returns the current underlying amqp/streadway channel being used.
func (testing *ChannelTesting) UnderlyingChannel() *streadway.Channel {
	if testing.channel.transportChannel == nil {
		return nil
	}
	return testing.channel.transportChannel.Channel
}

// Returns the current underlying amqp/streadway connection being used.
func (testing *ChannelTesting) UnderlyingConnection() *streadway.Connection {
	return testing.channel.transportChannel.rogerConn.transportConn.Connection
}

// Returns the current underlying amqp/streadway connection being used.
func (testing *ChannelTesting) ReconnectionCount() uint64 {
	return testing.channel.reconnectCount
}

// Test methods for the transport
func (channel *Channel) Test(t *testing.T) *ChannelTesting {
	blocks := int32(0)

	chanTester := &ChannelTesting{
		transportTesting: &transportTesting{
			t:       t,
			manager: channel.transportManager,
			blocks:  &blocks,
		},
		channel:            channel,
		DefaultMiddlewares: channel.transportChannel.defaultMiddlewares,
	}

	t.Cleanup(chanTester.cleanup)
	return chanTester
}
