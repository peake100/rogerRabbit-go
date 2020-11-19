package amqp

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
	"sync/atomic"
)

// This object holds the current settings for our channel. We break this into it's own
// struct so that we can pass the current settings to methods like
// eventRelay.SetupForRelayLeg() without exposing objects such methods should not have
// access to.
type channelSettings struct {
	// Whether the channel is in publisher confirms mode. When true, all re-connected
	// channels will be put into publisher confirms mode.
	publisherConfirms bool

	// The current delivery tag for this robust connection. Each time a message is
	// successfully published, this value should be atomically incremented. This tag
	// will function like the normal channel tag, AND WILL NOT RESET when the underlying
	// channel is re-established. Whenever we reconnect, the broker will reset and begin
	// delivery tags at 1. That means that we are going to need to track how the current
	// underlying channel's delivery tag matches up against our user-facing tags.
	//
	// The goal is to simulate the normal channel's behavior and continue to send an
	// unbroken stream of incrementing delivery tags, even during multiple connection
	// interruptions.
	//
	// We use a pointer here to support atomic operations.
	tagPublishCount *uint64
	// Offset to add to a given tag to get it's actual broker delivery tag for the
	// current channel.
	tagPublishOffset *uint64
	// As tagPublishCount, but for delivery tags of delivered messages.
	tagConsumeCount *uint64
	// As tagConsumeCount, but for consumption tags.
	tagConsumeOffset *uint64
}

// Implements transport for *streadway.Channel.
type transportChannel struct {
	// The current, underlying channel object.
	*streadway.Channel

	// The roger connection we will use to re-establish dropped channels.
	rogerConn *Connection

	// Current settings for the channel.
	settings channelSettings

	// List of queues that must be declared upon re-establishing the channel. We use a
	// map so we can remove queues from this list on queue delete.
	declareQueues *sync.Map

	// Event processors should grab this WaitGroup when they spin up and release it
	// when they have finished processing all available events after a channel close.
	// This will block a reconnection from being available for new work until all
	// work from the previous channel has been finished.
	//
	// This will also block the updating of values like tagPublishCount until all
	// eventProcessors have finished work.
	eventRelaysRunning *sync.WaitGroup
	// This WaitGroup will close when a new channel is opened but not yet serving
	// requests, this allows event processors to grab information about the channel
	// and remain confident that we did not miss a new channel while wrapping up event
	// processing from a previous channel in a situation where we are rapidly connecting
	// and disconnecting
	eventRelaysRunSetup *sync.WaitGroup
	// This WaitGroup should be added to before a processor releases
	// eventRelaysRunning and released after a processor has acquired all the info
	// it needs from a new channel. Once this WaitGroup is fully released, the write
	// lock on the channel will be released and channel methods will become available to
	// callers again.
	eventRelaysSetupComplete *sync.WaitGroup
	// This WaitGroup will be released when all relays are ready. Event relays will
	// wait on this group after releasing eventRelaysSetupComplete so that they don't error out
	// and re-enter the beginning of their lifecycle before eventRelaysRunSetup has
	// been re-acquired by the transport manager.
	eventRelaysGo *sync.WaitGroup

	// Logger for the channel transport
	logger zerolog.Logger
}

func (transport *transportChannel) cleanup() error {
	// Release this lock so event processors can close.
	defer transport.eventRelaysRunSetup.Done()
	return nil
}

// Sets up a channel with all the settings applied to the last one. This method will
// get called whenever a channel connection is re-established with a fresh channel.
func (transport *transportChannel) reconnectApplyChannelSettings() error {
	if transport.logger.Debug().Enabled() {
		transport.logger.Debug().
			Bool("CONFIRMS", transport.settings.publisherConfirms).
			Msg("applying channel settings")
	}

	if transport.settings.publisherConfirms {
		err := transport.Channel.Confirm(false)
		if err != nil {
			return fmt.Errorf("error putting into confirm mode: %w", err)
		}
	}

	return nil
}

func (transport *transportChannel) reconnectUpdateTagTrackers() {
	currentPublished := *transport.settings.tagPublishCount
	currentConsumed := *transport.settings.tagConsumeCount

	transport.settings.tagPublishOffset = &currentPublished
	transport.settings.tagConsumeOffset = &currentConsumed
}

func (transport *transportChannel) reconnectDeclareQueues() error {
	var err error

	redeclareQueues := func(key, value interface{}) bool {
		thisQueue := value.(*queueDeclareArgs)
		_, err = transport.Channel.QueueDeclare(
			thisQueue.name,
			thisQueue.durable,
			thisQueue.autoDelete,
			thisQueue.exclusive,
			// We need to wait and confirm this gets received before moving on
			false,
			thisQueue.args,
		)
		if err != nil {
			err = fmt.Errorf(
				"error re-declaring queue '%v': %w", thisQueue.name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	transport.declareQueues.Range(redeclareQueues)

	return err
}

func (transport *transportChannel) tryReconnect(ctx context.Context) error {
	// Wait for all event processors processing events from the previous channel to be
	// ready.
	logger := transport.logger.With().Str("STATUS", "CONNECTING").Logger()
	debugEnabled := logger.Debug().Enabled()

	if debugEnabled {
		logger.Debug().Msg("waiting for event relays to finish")
	}
	transport.eventRelaysRunning.Wait()

	if debugEnabled {
		logger.Debug().Msg("getting new channel")
	}
	channel, err := transport.rogerConn.getStreadwayChannel(ctx)
	if err != nil {
		return err
	}

	if transport.logger.Debug().Enabled() {
		transport.logger.Debug().
			Str("STATUS", "CONNECTING").
			Msg("setting up channel")
	}

	// Set up the new channel
	transport.Channel = channel
	err = transport.reconnectApplyChannelSettings()
	if err != nil {
		return err
	}

	if debugEnabled {
		logger.Debug().Msg("updating tag trackers")
	}
	transport.reconnectUpdateTagTrackers()

	if debugEnabled {
		logger.Debug().Msg("re-declaring queues")
	}
	err = transport.reconnectDeclareQueues()
	if err != nil {
		return err
	}

	// Acquire the final WaitGroup to release when we are ready to let the relays start
	// processing again.
	transport.eventRelaysGo.Add(1)

	// Allow the relays to advance to the setup stage.
	if debugEnabled {
		logger.Debug().Msg("advancing event relays to setup")
	}
	transport.eventRelaysRunSetup.Done()

	// Wait for the relays to finish their setup.
	transport.eventRelaysSetupComplete.Wait()

	// Block the relays from setting up again until we have a new channel.
	transport.eventRelaysRunSetup.Add(1)

	// Release the relays, allowing them to start processing events on our new channel.
	if debugEnabled {
		logger.Debug().Msg("restarting event relay processing")
	}
	transport.eventRelaysGo.Done()

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

// Store queue declare information for re-establishing queues on disconnect.
type queueDeclareArgs struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       streadway.Table
}

/*
ROGER NOTE: Queues declared with a roger channel will be automatically re-declared
upon channel disconnect recovery. Calling channel.QueueDelete() will remove the queue
from the list of queues to be re-declared in case of a disconnect.

--

QueueDeclare declares a queue to hold messages and deliver to consumers.
Declaring creates a queue if it doesn't already exist, or ensures that an
existing queue matches the same parameters.

Every queue declared gets a default binding to the empty exchange "" which has
the type "direct" with the routing key matching the queue's name.  With this
default binding, it is possible to publish messages that route directly to
this queue by publishing to "" with the routing key of the queue name.

  QueueDeclare("alerts", true, false, false, false, nil)
  Publish("", "alerts", false, false, Publishing{Body: []byte("...")})

  Delivery       Exchange  Key       Queue
  -----------------------------------------------
  key: alerts -> ""     -> alerts -> alerts

The queue name may be empty, in which case the server will generate a unique name
which will be returned in the Name field of Queue struct.

Durable and Non-Auto-Deleted queues will survive server restarts and remain
when there are no remaining consumers or bindings. Persistent publishings will
be restored in this queue on server restart.  These queues are only able to be
bound to durable exchanges.

Non-Durable and Auto-Deleted queues will not be redeclared on server restart
and will be deleted by the server after a short time when the last consumer is
canceled or the last consumer's channel is closed.  Queues with this lifetime
can also be deleted normally with QueueDelete.  These durable queues can only
be bound to non-durable exchanges.

Non-Durable and Non-Auto-Deleted queues will remain declared as long as the
server is running regardless of how many consumers.  This lifetime is useful
for temporary topologies that may have long delays between consumer activity.
These queues can only be bound to non-durable exchanges.

Durable and Auto-Deleted queues will be restored on server restart, but without
active consumers will not survive and be removed.  This Lifetime is unlikely
to be useful.

Exclusive queues are only accessible by the connection that declares them and
will be deleted when the connection closes.  Channels on other connections
will receive an error when attempting  to declare, bind, consume, purge or
delete a queue with the same name.

When noWait is true, the queue will assume to be declared on the server.  A
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
	operation := func() error {
		var opErr error
		queue, opErr = channel.transportChannel.QueueDeclare(
			name, durable, autoDelete, exclusive, noWait, args,
		)

		// We need to remember to re-declare this queue on reconnect
		if opErr == nil {
			var argsCopy Table
			if args != nil {
				argsCopy = make(Table)
				for key, value := range args {
					argsCopy[key] = value
				}
			}

			queueArgs := &queueDeclareArgs{
				name:       name,
				durable:    durable,
				autoDelete: autoDelete,
				exclusive:  exclusive,
				noWait:     noWait,
				args:       argsCopy,
			}

			channel.transportChannel.declareQueues.Store(name, queueArgs)
		}

		return opErr
	}

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
QueueInspect passively declares a queue by name to inspect the current message
count and consumer count.

Use this method to check how many messages ready for delivery reside in the queue,
how many consumers are receiving deliveries, and whether a queue by this
name already exists.

If the queue by this name exists, use Channel.QueueDeclare check if it is
declared with specific parameters.

If a queue by this name does not exist, an error will be returned and the
channel will be closed.

*/
func (channel *Channel) QueueInspect(
	name string,
) (queue Queue, err error) {
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
be routed to the queue when the publishing routing key matches the binding
routing key.

  QueueBind("pagers", "alert", "log", false, nil)
  QueueBind("emails", "info", "log", false, nil)

  Delivery       Exchange  Key       Queue
  -----------------------------------------------
  key: alert --> log ----> alert --> pagers
  key: info ---> log ----> info ---> emails
  key: debug --> log       (none)    (dropped)

If a binding with the same key and arguments already exists between the
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
  key: alert --> amq.topic ----> alert --> pagers
  key: info ---> amq.topic ----> # ------> emails
                           \---> info ---/
  key: debug --> amq.topic ----> # ------> emails

It is only possible to bind a durable queue to a durable exchange regardless of
whether the queue or exchange is auto-deleted.  Bindings between durable queues
and exchanges will also be restored on server restart.

If the binding could not complete, an error will be returned and the channel
will be closed.

When noWait is false and the queue could not be bound, the channel will be
closed with an error.

*/
func (channel *Channel) QueueBind(
	name, key, exchange string, noWait bool, args Table,
) error {
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.QueueBind(name, key, exchange, noWait, args)
		return opErr
	}

	err := channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
QueueUnbind removes a binding between an exchange and queue matching the key and
arguments.

It is possible to send and empty string for the exchange name which means to
unbind the queue from the default exchange.

*/
func (channel *Channel) QueueUnbind(name, key, exchange string, args Table) error {
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.QueueUnbind(name, key, exchange, args)
		return opErr
	}

	err := channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
QueuePurge removes all messages from the named queue which are not waiting to
be acknowledged.  Messages that have been delivered but have not yet been
acknowledged will not be removed.

When successful, returns the number of messages purged.

If noWait is true, do not wait for the server response and the number of
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

When noWait is true, the queue will be deleted without waiting for a response
from the server.  The purged message count will not be meaningful. If the queue
could not be deleted, a channel exception will be raised and the channel will
be closed.

*/
func (channel *Channel) QueueDelete(
	name string, ifUnused, ifEmpty, noWait bool,
) (count int, err error) {
	operation := func() error {
		var opErr error
		count, opErr = channel.transportChannel.QueueDelete(
			name, ifUnused, ifEmpty, noWait,
		)
		if opErr != nil {
			return opErr
		}

		channel.transportChannel.declareQueues.Delete(name)

		return nil
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return count, err
}

/*
Publish sends a Publishing from the client to an exchange on the server.

When you want a single message to be delivered to a single queue, you can
publish to the default exchange with the routingKey of the queue name.  This is
because every declared queue gets an implicit route to the default exchange.

Since publishings are asynchronous, any undeliverable message will get returned
by the server.  Add a listener with Channel.NotifyReturn to handle any
undeliverable message when calling publish with either the mandatory or
immediate parameters as true.

Publishings can be undeliverable when the mandatory flag is true and no queue is
bound that matches the routing key, or when the immediate flag is true and no
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
	operation := func() error {
		opErr := channel.transportChannel.Publish(
			exchange,
			key,
			mandatory,
			immediate,
			msg,
		)
		if opErr != nil {
			return opErr
		}

		if channel.logger.Debug().Enabled() {
			channel.logger.Debug().
				Str("EXCHANGE", exchange).
				Str("ROUTING_KEY", key).
				Bytes("BODY", msg.Body).
				Str("MESSAGE_ID", msg.MessageId).
				Msg("published message")
		}

		// Return if publishing confirms is not turned on
		if !channel.transportChannel.settings.publisherConfirms {
			return nil
		}

		// If there was no error, and we are in confirms mode, increment the current
		// delivery tag. We need to do this atomically so if publish is getting called
		// in more than one goroutine, we don't have a data race condition and
		// under-count our tags.
		atomic.AddUint64(channel.transportChannel.settings.tagPublishCount, 1)

		return opErr
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
) (msg Delivery, ok bool, err error) {
	operation := func() error {
		var opErr error
		var msgStreadway streadway.Delivery
		msgStreadway, ok, opErr = channel.transportChannel.Get(
			queue,
			autoAck,
		)
		// If there was no error, atomically increment the delivery tag.
		if opErr == nil {
			atomic.AddUint64(channel.transportChannel.settings.tagConsumeCount, 1)
			msg = newDelivery(
				msgStreadway, channel.transportChannel.settings.tagConsumeOffset,
			)
		}
		return opErr
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return msg, ok, err
}

/*
ROGER NOTE: unlike normal consume channels, this channel's delivery tags will reset
whenever the underlying connection is re-established. It is on the roadmap to bring
the behavior completely in line with the normal Consume behavior, where each delivery
gets an incremented tag, but doing so is depended on handling acks across channel
re-connects.

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

When exclusive is true, the server will ensure that this is the sole consumer
from this queue. When exclusive is false, the server will fairly distribute
deliveries across multiple consumers.

The noLocal flag is not supported by RabbitMQ.

It's advisable to use separate connections for
Channel.Publish and Channel.Consume so not to have TCP pushback on publishing
affect the ability to consume messages, so this parameter is here mostly for
completeness.

When noWait is true, do not wait for the server to confirm the request and
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
) (deliveryChan <-chan Delivery, err error) {
	// Make a buffered channel so we don't cause latency from waiting for queues to be
	// ready
	callerDeliveries := make(chan Delivery, 16)
	deliveryChan = callerDeliveries

	// Create our consumer relay
	consumerRelay := &ConsumerRelay{
		ConsumeArgs: consumeArgs{
			queue:     queue,
			consumer:  consumer,
			autoAck:   autoAck,
			exclusive: exclusive,
			noLocal:   noLocal,
			noWait:    noWait,
			args:      args,
		},
		CallerDeliveries: callerDeliveries,
	}

	// Pass it to our relay handler.
	err = channel.setupAndLaunchEventRelay(consumerRelay)
	if err != nil {
		return nil, err
	}

	// If no error, pass the channel back to the caller
	return deliveryChan, nil
}

/*
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

When noWait is true, the client will not wait for a response.  A channel
exception could occur if the server does not support this method.

*/
func (channel *Channel) Confirm(noWait bool) error {
	op := func() error {
		var opErr error
		opErr = channel.transportChannel.Confirm(noWait)
		if opErr != nil {
			return opErr
		}

		// If the channel was already in confirms mode, there is nothing more to do
		if channel.transportChannel.settings.publisherConfirms {
			return nil
		}

		// otherwise set the setting to true and create a new context.
		channel.transportChannel.settings.publisherConfirms = true

		return nil
	}

	return channel.retryOperationOnClosed(channel.ctx, op, true)
}

/*
ROGER NOTE: It is possible that if a channel is disconnected unexpectedly, there may
have been confirmations in flight that did not reach the client. In cases where a
channel connection is re-established, ant missing delivery tags will be reported nacked,
but an additional DisconnectOrphan field will be set to `true`. It is possible that
such messages DID reach the broker, but the Ack messages were lost in the disconnect
event.

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
func (channel *Channel) NotifyPublish(confirm chan Confirmation) chan Confirmation {
	// Setup and launch the event relay that will handle these events across multiple
	// connections.
	relay := &notifyPublishRelay{
		CallerConfirmations: confirm,
	}

	err := channel.setupAndLaunchEventRelay(relay)
	// On an error, close the channel.
	if err != nil {
		channel.logger.Err(err).Msg("error setting up NotifyPublish event relay")
		close(confirm)
	}
	return confirm
}

// Closes confirmation tag channels for NotifyConfirm and NotifyConfirmOrOrphan.
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
	confirmation Confirmation, ack, nack chan uint64, logger zerolog.Logger,
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
Orphaned tags separately, use the new method NotifyConfirmOrOrphan.

--

NotifyConfirm calls NotifyPublish and starts a goroutine sending
ordered Ack and Nack DeliveryTag to the respective channels.

For strict ordering, use NotifyPublish instead.
*/
func (channel *Channel) NotifyConfirm(
	ack, nack chan uint64,
) (chan uint64, chan uint64) {
	confirmsEvents := channel.NotifyPublish(make(chan Confirmation, cap(ack)+cap(nack)))
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
func (channel *Channel) NotifyConfirmOrOrphan(
	ack, nack, orphaned chan uint64,
) (chan uint64, chan uint64, chan uint64) {
	confirmsEvents := channel.NotifyPublish(make(chan Confirmation, cap(ack)+cap(nack)))
	logger := channel.logger.With().
		Str("EVENT_TYPE", "NOTIFY_CONFIRM_OR_ORPHAN").
		Logger()

	go func() {
		// Close channels on exit
		defer notifyConfirmCloseConfirmChannels(ack, nack, orphaned)

		// range over confirmation events and place them in the ack and nack channels.
		for confirmation := range confirmsEvents {
			if confirmation.DisconnectOrphan {

			} else {
				if logger.Debug().Enabled() {
					logger.Debug().
						Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
						Bool("ACK", confirmation.Ack).
						Str("CHANNEL", "ORPHANED").
						Msg("orphaned confirmation sent")
				}
				orphaned <- confirmation.DeliveryTag
				notifyConfirmHandleAckAndNack(confirmation, ack, nack, logger)
			}
		}
	}()

	return ack, nack, orphaned
}
