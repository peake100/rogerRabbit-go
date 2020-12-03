package amqp

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
	"sync/atomic"
)

// Store queue declare information for re-establishing queues on disconnect.
type queueDeclareArgs struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       Table
}

// Store queue bind information for re-establishing bindings on disconnect.
type queueBindArgs struct {
	name     string
	key      string
	exchange string
	noWait   bool
	args     Table
}

// Store exchange declare information for re-establishing queues on disconnect.
type exchangeDeclareArgs struct {
	name       string
	kind       string
	durable    bool
	autoDelete bool
	internal   bool
	noWait     bool
	args       Table
}

// Store exchange bind information for re-establishing bindings on disconnect.
type exchangeBindArgs struct {
	destination string
	key         string
	source      string
	noWait      bool
	args        Table
}

// Holds current qos settings for the channel so they can be re-instated on reconnect.
type qosSettings struct {
	prefetchCount int
	prefetchSize  int
	global        bool
}

// Enum-like for acknowledgement types.
type ackMethod int

const (
	ack    ackMethod = 0
	nack   ackMethod = 1
	reject ackMethod = 2
)

// Ack command for acknowledge routine.
type ackInfo struct {
	// Delivery tag to acknowledge
	deliveryTag uint64
	// The method to call when acknowledging
	method ackMethod
	// Passed to `multiple` arg on ack and nack methods
	multiple bool
	// Passed to `requeue` arg on nack and reject methods
	requeue bool
	// result passed to this channel so it can be returned to the initial caller.
	resultChan chan error
}

func newAckInfo(
	deliveryTag uint64, method ackMethod, multiple bool, requeue bool,
) ackInfo {
	return ackInfo{
		deliveryTag: deliveryTag,
		method:      method,
		multiple:    multiple,
		requeue:     requeue,
		// use a buffer as 1 so we don't block the acknowledge routine when it sends a
		// result
		resultChan: make(chan error, 1),
	}
}

// This object holds the current settings for our channel. We break this into it's own
// struct so that we can pass the current settings to methods like
// eventRelay.SetupForRelayLeg() without exposing objects such methods should not have
// access to.
type channelSettings struct {
	// Whether the channel is in publisher confirms mode. When true, all re-connected
	// channels will be put into publisher confirms mode.
	publisherConfirms bool

	// If not nil, these are the latest qos settings passed to the QoS() method for the
	// channel, and will be-reapplied to any new channel created due to a disconnection
	// event.
	qos *qosSettings
	// Whether consumer flow to this channel is open. When false, re-established
	// channels will immediately be put into pause mode.
	flowActive bool

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
	tagPublishOffset uint64
	// As tagPublishCount, but for delivery tags of delivered messages.
	tagConsumeCount *uint64
	// As tagConsumeCount, but for consumption tags.
	tagConsumeOffset uint64
	// The highest ack we have received
	tagLatestDeliveryAck uint64
}

// Implements transport for *streadway.Channel.
type transportChannel struct {
	// The current, underlying channel object.
	*streadway.Channel

	// The roger connection we will use to re-establish dropped channels.
	rogerConn *Connection

	// Current settings for the channel.
	settings channelSettings
	// locks the flow state from being changed until release
	flowActiveLock *sync.Mutex

	// List of queues that must be declared upon re-establishing the channel. We use a
	// map so we can remove queues from this list on queue delete.
	declareQueues *sync.Map
	// List of exchanges that must be declared upon re-establishing the channel. We use
	// a map so we can remove exchanges from this list on exchange delete.
	declareExchanges *sync.Map
	// List of bindings to re-build on channel re-establishment.
	bindQueues []*queueBindArgs
	// Lock that must be acquired to alter bindQueues.
	bindQueuesLock *sync.Mutex
	// List of bindings to re-build on channel re-establishment.
	bindExchanges []*exchangeBindArgs
	// Lock that must be acquired to alter bindQueues.
	bindExchangesLock *sync.Mutex

	// We're going to handle all nacks and acks in a goroutine so we can track the
	// latest without heavy lock contention. This channels will be used to send
	// them to the processor.
	//
	// TODO: It may be faster to have a parking lot channel of ackInfo values we can
	// 	recycle rather than creating them fresh each time. Should benchmark both
	//	 approaches
	ackChan chan ackInfo

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
	// wait on this group after releasing eventRelaysSetupComplete so that they don't
	// error out and re-enter the beginning of their lifecycle before
	// eventRelaysRunSetup has been re-acquired by the transport manager.
	eventRelaysGo *sync.WaitGroup

	// Logger for the channel transport
	logger zerolog.Logger
}

func (transport *transportChannel) cleanup() error {
	// Release this lock so event processors can close.
	defer transport.eventRelaysRunSetup.Done()
	defer close(transport.ackChan)
	return nil
}

// Removed a queue from the list of queues to be redeclared on reconnect
func (transport *transportChannel) removeQueue(queueName string) {
	// Remove the queue.
	transport.declareQueues.Delete(queueName)
	// Remove all binding commands associated with this queue from the re-bind on
	// reconnect list.
	transport.removeQueueBindings(
		queueName,
		"",
		"",
	)
}

// Remove a re-connection queue binding when a queue, exchange, or binding is removed.
func (transport *transportChannel) removeQueueBindings(
	queueName string,
	exchangeName string,
	routingKey string,
) {
	removeQueueMatch := false
	removeExchangeMatch := false
	removeRouteMatch := false

	if queueName != "" {
		removeQueueMatch = true
	}
	if exchangeName != "" {
		removeExchangeMatch = true
	}
	if routingKey != "" {
		removeRouteMatch = true
	}

	transport.bindQueuesLock.Lock()
	defer transport.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range transport.bindQueues {
		// If there is a routing key to match, then the queue and exchange must match
		// too (so we don't end up removing a binding with the same routing key between
		// a different queue-exchange pair).
		if removeRouteMatch &&
			thisBinding.key == routingKey &&
			thisBinding.name == queueName &&
			thisBinding.exchange == exchangeName {
			// then:
			continue
		} else if removeQueueMatch && thisBinding.name == queueName {
			continue
		} else if removeExchangeMatch && thisBinding.exchange == exchangeName {
			continue
		}

		transport.bindQueues[i] = thisBinding
		i++
	}

	transport.bindQueues = transport.bindQueues[0:i]
}

// Remove an exchange from the re-declaration list, as well as all queue and
// inter-exchange bindings it was a part of.
func (transport *transportChannel) removeExchange(exchangeName string) {
	// Remove the exchange from the list of exchanges we need to re-declare
	transport.declareExchanges.Delete(exchangeName)
	// Remove all bindings associated with this exchange from the list of bindings
	// to re-declare on re-connections.
	transport.removeQueueBindings("", exchangeName, "")
	transport.removeExchangeBindings(
		"", "", "", exchangeName,
	)
}

// Remove a re-connection binding when a binding or exchange is removed.
func (transport *transportChannel) removeExchangeBindings(
	destination string,
	key string,
	source string,
	// When a queue is deleted we need to remove any binding where the source or
	// destination matches
	destinationOrSource string,
) {
	removeDestinationMatch := false
	removeKeyMatch := false
	removeSourceMatch := false

	if destination != "" {
		removeDestinationMatch = true
	}
	if key != "" {
		removeKeyMatch = true
	}
	if source != "" {
		removeSourceMatch = true
	}
	if destinationOrSource != "" {
		removeSourceMatch = true
		removeDestinationMatch = true
	}

	transport.bindQueuesLock.Lock()
	defer transport.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range transport.bindExchanges {
		// If there is a routing key to match, then the source and destination exchanges
		// must match too (so we don't end up removing a binding with the same routing
		// key between a different set of exchanges).
		if removeKeyMatch &&
			thisBinding.key == key &&
			thisBinding.source == source &&
			thisBinding.destination == destination {
			// then:
			continue
		} else if removeDestinationMatch && thisBinding.destination == destination {
			continue
		} else if removeSourceMatch && thisBinding.source == source {
			continue
		}

		transport.bindExchanges[i] = thisBinding
		i++
	}

	transport.bindQueues = transport.bindQueues[0:i]
}

// Sets up a channel with all the settings applied to the last one. This method will
// get called whenever a channel connection is re-established with a fresh channel.
func (transport *transportChannel) reconnectApplyChannelSettings() error {
	if transport.logger.Debug().Enabled() {
		transport.logger.Debug().
			Bool("CONSUMER_FLOW_ACTIVE", transport.settings.flowActive).
			Bool("CONFIRMS", transport.settings.publisherConfirms).
			Interface("QOS", transport.settings.qos).
			Msg("applying channel settings")
	}

	// If flow was paused, immediately pause it again
	if !transport.settings.flowActive {
		transport.flowActiveLock.Lock()
		defer transport.flowActiveLock.Unlock()

		err := transport.Channel.Flow(transport.settings.flowActive)
		if err != nil {
			return fmt.Errorf("error pausing consumer flow: %w", err)
		}
	}

	// If in publisher confirms mode, set up the new channel to be so.
	if transport.settings.publisherConfirms {
		err := transport.Channel.Confirm(false)
		if err != nil {
			return fmt.Errorf("error putting into confirm mode: %w", err)
		}
	}

	// If qos settings were given, apply them.
	qos := transport.settings.qos
	if qos != nil {
		err := transport.Channel.Qos(qos.prefetchCount, qos.prefetchSize, qos.global)
		if err != nil {
			return fmt.Errorf("error applying qos settings: %w", err)
		}
	}

	return nil
}

func (transport *transportChannel) reconnectUpdateTagTrackers() {
	transport.settings.tagPublishOffset = *transport.settings.tagPublishCount
	transport.settings.tagConsumeOffset = *transport.settings.tagConsumeCount
}

func (transport *transportChannel) reconnectDeclareQueues() error {
	var err error

	redeclareQueues := func(key, value interface{}) bool {
		thisQueue := value.(*queueDeclareArgs)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := transport.Channel.QueueDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnect.

		// TODO: add ability to configure whether or not a passive declare should be
		//   used.
		if true {
			method = transport.Channel.QueueDeclare
		}

		_, err = method(
			thisQueue.name,
			thisQueue.durable,
			thisQueue.autoDelete,
			thisQueue.exclusive,
			// We need to wait and confirm this gets received before moving on
			false,
			thisQueue.args,
		)
		if err != nil {
			var streadwayErr *Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == NotFound {
				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				transport.removeQueue(thisQueue.name)
			}

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

// Re-declares exchanges during reconnection
func (transport *transportChannel) reconnectDeclareExchanges() error {
	var err error

	redeclareExchanges := func(key, value interface{}) bool {
		thisExchange := value.(*exchangeDeclareArgs)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := transport.Channel.ExchangeDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnect.

		// TODO: add ability to configure whether or not a passive declare should be
		//   used.
		if true {
			method = transport.Channel.ExchangeDeclare
		}

		err = method(
			thisExchange.name,
			thisExchange.kind,
			thisExchange.durable,
			thisExchange.autoDelete,
			thisExchange.internal,
			// we are going to wait so this is done synchronously.
			false,
			thisExchange.args,
		)
		if err != nil {
			var streadwayErr *Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == NotFound {
				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				transport.removeExchange(thisExchange.name)
			}
			err = fmt.Errorf(
				"error re-declaring exchange '%v': %w", thisExchange.name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	transport.declareExchanges.Range(redeclareExchanges)

	return err
}

// Re-declares queue bindings during reconnection
func (transport *transportChannel) reconnectBindQueues() error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	transport.bindQueuesLock.Lock()
	defer transport.bindQueuesLock.Unlock()

	for _, thisBinding := range transport.bindQueues {
		err := transport.Channel.QueueBind(
			thisBinding.name,
			thisBinding.key,
			thisBinding.exchange,
			false,
			thisBinding.args,
		)

		if err != nil {
			return fmt.Errorf(
				"error re-binding queue '%v' to exchange '%v' with routing key"+
					" '%v': %w",
				thisBinding.name,
				thisBinding.exchange,
				thisBinding.key,
				err,
			)
		}
	}

	return nil
}

// Re-declares exchange bindings during reconnection
func (transport *transportChannel) reconnectBindExchanges() error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	transport.bindExchangesLock.Lock()
	defer transport.bindExchangesLock.Unlock()

	for _, thisBinding := range transport.bindExchanges {
		err := transport.Channel.ExchangeBind(
			thisBinding.destination,
			thisBinding.key,
			thisBinding.source,
			false,
			thisBinding.args,
		)

		if err != nil {
			return fmt.Errorf(
				"error re-binding source exchange '%v' to destination exchange"+
					" '%v' with routing key '%v': %w",
				thisBinding.source,
				thisBinding.destination,
				thisBinding.key,
				err,
			)
		}
	}

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

	// Recreate queue and exchange topology
	if debugEnabled {
		logger.Debug().Msg("re-declaring queues")
	}
	err = transport.reconnectDeclareQueues()
	if err != nil {
		return err
	}

	if debugEnabled {
		logger.Debug().Msg("re-declaring exchanges")
	}
	err = transport.reconnectDeclareExchanges()
	if err != nil {
		return err
	}

	if debugEnabled {
		logger.Debug().Msg("re-binding queues")
	}
	err = transport.reconnectBindQueues()
	if err != nil {
		return err
	}

	if debugEnabled {
		logger.Debug().Msg("re-binding exchanges")
	}
	err = transport.reconnectBindExchanges()
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

// Start helper routines.
func (channel *Channel) start() {
	go channel.runAcknowledgementRoutine()
}

// We are going to handle sending acknowledgements in a single goroutine so we can
// avoid using locks to handle tracking the latest acknowledged tag.
func (channel *Channel) runAcknowledgementRoutine() {
	var logger zerolog.Logger
	// Each loop will put the ack request into this variable
	var ackReq ackInfo
	// Each loop will put the tag with offset into this variable
	var tagWithOffset uint64

	transportChannel := channel.transportChannel
	// This needs to be a reference so we get the latest values
	settings := &transportChannel.settings

	// Wrap the underlying methods in closures with identical signatures so we can
	// call them generically based on what kind of method we are using.
	ackMethod := func() error {
		logger = channel.logger.With().Str("METHOD", "ACK").Logger()
		return transportChannel.Ack(
			tagWithOffset,
			ackReq.multiple,
		)
	}

	nackMethod := func() error {
		logger = channel.logger.With().Str("METHOD", "NACK").Logger()
		return transportChannel.Nack(
			tagWithOffset,
			ackReq.multiple,
			ackReq.requeue,
		)
	}

	rejectMethod := func() error {
		logger = channel.logger.With().Str("METHOD", "REJECT").Logger()
		return transportChannel.Reject(
			tagWithOffset,
			ackReq.requeue,
		)
	}

	var method func() error

	// Only one of these operations will be occurring simultaneously.
	operation := func() error {
		var opErr error

		// If there was no error, set the latest delivery tag to this tag on exit.
		defer func() {
			if opErr == nil {
				settings.tagLatestDeliveryAck = ackReq.deliveryTag
			}
		}()

		// If this delivery tag is less or equal to our current offset, then all
		// requested tags are orphans and we can return an error
		if ackReq.deliveryTag <= settings.tagConsumeOffset {
			return newErrCantAcknowledgeOrphans(
				settings.tagLatestDeliveryAck,
				ackReq.deliveryTag,
				settings.tagConsumeOffset,
				ackReq.multiple,
			)
		}

		tagWithOffset = ackReq.deliveryTag - settings.tagConsumeOffset

		// Invoke the method and return any errors
		opErr = method()
		if opErr != nil {
			return opErr
		}

		if logger.Debug().Enabled() {
			logger.Debug().
				Uint64("DELIVERY TAG", ackReq.deliveryTag).
				Bool("MULTIPLE", ackReq.multiple).
				Bool("REQUEUE", ackReq.requeue).
				Msg("delivery acknowledgement(s) sent")
		}

		// If we are acknowledging multiple requests and some of them span into a
		// previous connection, we need to report an orphaned tag error.
		if ackReq.method != reject &&
			ackReq.multiple &&
			settings.tagLatestDeliveryAck < settings.tagConsumeOffset {
			// then:
			return newErrCantAcknowledgeOrphans(
				settings.tagLatestDeliveryAck,
				ackReq.deliveryTag,
				settings.tagConsumeOffset,
				ackReq.multiple,
			)
		}

		return nil
	}

	// Range over all the acknowledgement requests we receive.
	for ackReq = range transportChannel.ackChan {
		// pick the correct underlying channel method to use
		switch ackReq.method {
		case ack:
			method = ackMethod
		case nack:
			method = nackMethod
		case reject:
			method = rejectMethod
		default:
			panic(errors.New("got bad ack request type"))
		}

		// Do the operation
		err := channel.retryOperationOnClosed(channel.ctx, operation, true)
		// Send the result back to the caller
		ackReq.resultChan <- err
	}
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

		// remember this setting so we can put re-established channels into confirmation
		// mode.
		channel.transportChannel.settings.publisherConfirms = true

		return nil
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
	op := func() error {
		opErr := channel.transportChannel.Qos(prefetchCount, prefetchSize, global)
		if opErr != nil {
			return opErr
		}

		// remember these settings so we can configure re-established channels the same
		// way.
		channel.transportChannel.settings.qos = &qosSettings{
			prefetchCount: prefetchCount,
			prefetchSize:  prefetchSize,
			global:        global,
		}

		return nil
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
	op := func() error {
		channel.transportChannel.flowActiveLock.Lock()
		defer channel.transportChannel.flowActiveLock.Unlock()

		opErr := channel.transportChannel.Flow(active)
		if opErr != nil {
			return opErr
		}

		// Update the setting if there was no error.
		channel.transportChannel.settings.flowActive = active

		return nil
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
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		queue, opErr = channel.transportChannel.QueueDeclare(
			name, durable, autoDelete, exclusive, noWait, args,
		)

		if opErr != nil {
			return opErr
		}

		// We need to remember to re-declare this queue on reconnect
		queueArgs := &queueDeclareArgs{
			name:       name,
			durable:    durable,
			autoDelete: autoDelete,
			exclusive:  exclusive,
			// We are always going to wait on re-declares, but we should save the
			// noWait value for posterity.
			noWait: noWait,
			// Copy the args so if the user mutates them for a future call we have
			// an un-changed version of the originals.
			args: copyTable(args),
		}

		channel.transportChannel.declareQueues.Store(name, queueArgs)

		return nil
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
QueueInspect passively declares a queue by name to inspect the current message
publishCount and consumer publishCount.

Use this method to check how many messages ready for delivery reside in the queue,
how many consumers are receiving deliveries, and whether a queue by this
name already exists.

If the queue by this name exists, use Channel.QueueDeclare check if it is
declared with specific parameters.

If a queue by this name does not exist, an error will be returned and the
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
	// Run an an operation to get automatic retries on channel disconnections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.QueueBind(name, key, exchange, noWait, args)

		if opErr != nil {
			return opErr
		}

		bindArgs := &queueBindArgs{
			name:     name,
			key:      key,
			exchange: exchange,
			noWait:   noWait,
			// Copy the arg table so if the caller re-uses it we preserve the original
			// values.
			args: copyTable(args),
		}

		channel.transportChannel.bindQueuesLock.Lock()
		defer channel.transportChannel.bindQueuesLock.Unlock()

		channel.transportChannel.bindQueues = append(
			channel.transportChannel.bindQueues, bindArgs,
		)
		return nil
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
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		opErr := channel.transportChannel.QueueUnbind(name, key, exchange, args)
		if opErr != nil {
			return opErr
		}

		// Remove this binding from the list of bindings to re-create on reconnect.
		channel.transportChannel.removeQueueBindings(
			name,
			exchange,
			key,
		)

		return nil
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
from the server.  The purged message publishCount will not be meaningful. If the queue
could not be deleted, a channel exception will be raised and the channel will
be closed.

*/
func (channel *Channel) QueueDelete(
	name string, ifUnused, ifEmpty, noWait bool,
) (count int, err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		count, opErr = channel.transportChannel.QueueDelete(
			name, ifUnused, ifEmpty, noWait,
		)
		if opErr != nil {
			return opErr
		}

		// Remove the queue from our list of queue to redeclare.
		channel.transportChannel.removeQueue(name)

		return nil
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
binding durable queues to auto-deleted exchanges.

Note: RabbitMQ declares the default exchange types like 'amq.fanout' as
durable, so queues that bind to these pre-declared exchanges must also be
durable.

Exchanges declared as `internal` do not accept accept publishings. Internal
exchanges are useful when you wish to implement inter-exchange topologies
that should not be exposed to users of the broker.

When noWait is true, declare without waiting for a confirmation from the server.
The channel may be closed as a result of an error.  Add a NotifyClose listener
to respond to any exceptions.

Optional amqp.Table of arguments that are specific to the server's implementation of
the exchange can be sent for exchange types that require extra parameters.
*/
func (channel *Channel) ExchangeDeclare(
	name, kind string, durable, autoDelete, internal, noWait bool, args Table,
) (err error) {

	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.ExchangeDeclare(
			name, kind, durable, autoDelete, internal, noWait, args,
		)
		if opErr != nil {
			return opErr
		}

		// Store the args so this exchange can be re-declared on channel
		// re-establishment.
		exchangeArgs := &exchangeDeclareArgs{
			name:       name,
			kind:       kind,
			durable:    durable,
			autoDelete: autoDelete,
			internal:   internal,
			// We will always use wait on re-establishments, but preserve the original
			// setting here for posterity.
			noWait: noWait,
			// Copy the table so if the caller re-uses it we dont have it mutated
			// between re-declarations.
			args: copyTable(args),
		}

		channel.transportChannel.declareExchanges.Store(name, exchangeArgs)

		return nil
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

When noWait is true, do not wait for a server confirmation that the exchange has
been deleted.  Failing to delete the channel could close the channel.  Add a
NotifyClose listener to respond to these channel exceptions.
*/
func (channel *Channel) ExchangeDelete(
	name string, ifUnused, noWait bool,
) (err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.ExchangeDelete(name, ifUnused, noWait)
		if opErr != nil {
			return opErr
		}

		// Remove the exchange from our re-declare on reconnect lists.
		channel.transportChannel.removeExchange(name)

		return nil
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
to the destination exchange when the routing key is matched.

  ExchangeBind("sell", "MSFT", "trade", false, nil)
  ExchangeBind("buy", "AAPL", "trade", false, nil)

  Delivery       Source      Key      Destination
  example        exchange             exchange
  -----------------------------------------------
  key: AAPL  --> trade ----> MSFT     sell
                       \---> AAPL --> buy

When noWait is true, do not wait for the server to confirm the binding.  If any
error occurs the channel will be closed.  Add a listener to NotifyClose to
handle these errors.

Optional arguments specific to the exchanges bound can also be specified.
*/
func (channel *Channel) ExchangeBind(
	destination, key, source string, noWait bool, args Table,
) (err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.ExchangeBind(
			destination, key, source, noWait, args,
		)
		if opErr != nil {
			return opErr
		}

		// Add this binding to the list of exchange bindings we must re-declare on
		// re-connection establishment.
		bindArgs := &exchangeBindArgs{
			destination: destination,
			key:         key,
			source:      source,
			noWait:      noWait,
			args:        args,
		}

		// Store this binding so we can re-bind it if we lose and regain the connection.
		channel.transportChannel.bindExchangesLock.Lock()
		defer channel.transportChannel.bindExchangesLock.Unlock()

		channel.transportChannel.bindExchanges = append(
			channel.transportChannel.bindExchanges, bindArgs,
		)

		return nil
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
}

/*
ExchangeUnbind unbinds the destination exchange from the source exchange on the
server by removing the routing key between them.  This is the inverse of
ExchangeBind.  If the binding does not currently exist, an error will be
returned.

When noWait is true, do not wait for the server to confirm the deletion of the
binding.  If any error occurs the channel will be closed.  Add a listener to
NotifyClose to handle these errors.

Optional arguments that are specific to the type of exchanges bound can also be
provided.  These must match the same arguments specified in ExchangeBind to
identify the binding.
*/
func (channel *Channel) ExchangeUnbind(
	destination, key, source string, noWait bool, args Table,
) (err error) {
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		opErr = channel.transportChannel.ExchangeUnbind(
			destination, key, source, noWait, args,
		)
		if opErr != nil {
			return opErr
		}

		channel.transportChannel.removeExchangeBindings(
			destination, key, source, "",
		)

		return nil
	}

	err = channel.retryOperationOnClosed(channel.ctx, operation, true)
	return err
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
	// Run an an operation to get automatic retries on channel dis-connections.
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
		// under-publishCount our tags.
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
	// Run an an operation to get automatic retries on channel dis-connections.
	operation := func() error {
		var opErr error
		var msgStreadway streadway.Delivery
		msgStreadway, ok, opErr = channel.transportChannel.Get(
			queue,
			autoAck,
		)
		if opErr != nil {
			return opErr
		}

		// If there was no error, atomically increment the delivery tag.
		atomic.AddUint64(channel.transportChannel.settings.tagConsumeCount, 1)
		msg = channel.newDelivery(msgStreadway)
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
	consumerRelay := &consumeRelay{
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
		NewDelivery:      channel.newDelivery,
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
	if channel.ctx.Err() != nil {
		return streadway.ErrClosed
	}

	// Send this ack request to the acknowledgement routine
	ackInfo := newAckInfo(tag, ack, multiple, false)
	channel.transportChannel.ackChan <- ackInfo

	// Pull the result
	return <-ackInfo.resultChan
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
	if channel.ctx.Err() != nil {
		return streadway.ErrClosed
	}

	// Send this ack request to the acknowledgement routine
	ackInfo := newAckInfo(tag, nack, multiple, requeue)
	channel.transportChannel.ackChan <- ackInfo

	// Pull the result
	return <-ackInfo.resultChan
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
	if channel.ctx.Err() != nil {
		return streadway.ErrClosed
	}

	// Send this ack request to the acknowledgement routine
	ackInfo := newAckInfo(tag, reject, false, requeue)
	channel.transportChannel.ackChan <- ackInfo

	// Pull the result
	return <-ackInfo.resultChan
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
Orphaned tags separately, use the new method NotifyConfirmOrOrphaned.

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
func (channel *Channel) NotifyConfirmOrOrphaned(
	ack, nack, orphaned chan uint64,
) (chan uint64, chan uint64, chan uint64) {
	confirmsEvents := channel.NotifyPublish(
		make(chan Confirmation, cap(ack)+cap(nack)+cap(orphaned)),
	)
	logger := channel.logger.With().
		Str("EVENT_TYPE", "NOTIFY_CONFIRM_OR_ORPHAN").
		Logger()

	go func() {
		// Close channels on exit
		defer notifyConfirmCloseConfirmChannels(ack, nack, orphaned)

		// range over confirmation events and place them in the ack and nack channels.
		for confirmation := range confirmsEvents {
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
	}()

	return ack, nack, orphaned
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
	relay := &cancelRelay{
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
	relay := &flowRelay{
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
