
package roger

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"sync"
)

// Error returned when trying to publish through the broker channel results in an error.
type ErrProducerPublish struct {
	AmqpErr error
}

func (err ErrProducerPublish) Unwrap() error {
	return err.AmqpErr
}

func (err ErrProducerPublish) Error() string {
	return fmt.Sprintf(
		"error publishing message through broker channel: %v", err.AmqpErr,
	)
}

// Error returned when trying to publish through the broker channel results in an error.
type ErrProducerNack struct {
	// Whether this nack was a result of being an orphaned message. Orphaned messages
	// MAY have been successfully published, but there is no way to know for sure that
	// the broker received it.
	Orphan bool
}

func (err ErrProducerNack) Error() string {
	return fmt.Sprintf(
		"message was nacked by server. orphan status: %v", err.Orphan,
	)
}

// The args we are going to call channel.Publish with. See that methods documentation
// for details on each args meaning.
type PublishArgs struct {
	Exchange string
	Key string
	Mandatory bool
	Immediate bool
	Msg amqp.Publishing
}

// The order for a publications.
type PublishOrder struct {
	// Embed the args.
	PublishArgs

	// Set by the publishing routine after a successful publication.
	DeliveryTag uint64

	// A channel we will send publishing results back to the original caller with.
	result chan error
}

// Options for amqp Producer
type ProducerOpts struct {
	// Whether to confirm publications with the broker.
	confirmPublish bool
	// The buffer size of our internal publication queues.
	internalQueueSize int
}

// Producer is a wrapper for an amqp channel that handles the boilerplate of confirming
// publishings, retrying nacked messages, and other useful quality of life features.
type Producer struct {
	// Context for the producer.
	ctx context.Context
	ctxCancel context.CancelFunc
	publishLock *sync.RWMutex

	// The channel we will be running publishing on. This producer is expected to have
	// full ownership of this channel. Any additional operations that occur on this
	// channel after it is passed to the producer may result in unexpected behavior.
	channel *amqp.Channel
	// Channel receiving events from channel.NotifyPublish.
	publishEvents chan amqp.Confirmation

	// Internal queue for messages waiting to be published.
	publishQueue chan *PublishOrder
	// Internal queue of messages that have been sent to the broker, but are awaiting
	// confirmation.
	confirmQueue chan *PublishOrder

	// Caller options for producer behavior
	opts ProducerOpts
}

func (producer *Producer) runPublisher() {
	// close the confirmation queue on the way out, letting the confirmation wrap up
	// work and exit.
	defer close(producer.confirmQueue)

	// Keep track of the current delivery tag.
	deliveryTag := uint64(0)

	// Range over our publish queue.
	for thisOrder := range producer.publishQueue {
		err := producer.channel.Publish(
			thisOrder.Exchange,
			thisOrder.Key,
			thisOrder.Mandatory,
			thisOrder.Immediate,
			thisOrder.Msg,
		)
		if err != nil {
			// Return the error
			thisOrder.result <- ErrProducerPublish{AmqpErr: err}
			continue
		}

		// Increment the delivery tag
		deliveryTag++
		// If we are not confirming publications, continue
		if !producer.opts.confirmPublish {
			continue
		}

		// If we are confirming orders,  send this to the confirmation routine
		thisOrder.DeliveryTag = deliveryTag
		producer.confirmQueue <- thisOrder
	}
}

func (producer *Producer) runConfirmations() {
	// start shutdown if something goes horribly wrong here.
	defer producer.StartShutdown()

	// Iterate over our internal confirmation queue
	for thisOrder := range producer.confirmQueue {
		// Get the next confirmation. Because confirmations are returned by the client
		// in order, we don't have to worry bout lining them up. We are getting
		// publications in order from both confirmQueue and
		confirmation, ok := <- producer.publishEvents
		if !ok {
			// If this channel is closed, then this is an orphan and we should send
			// a closed err. This channel will not be opened again.
			thisOrder.result <- amqp.ErrClosed
			return
		}

		if confirmation.DeliveryTag != thisOrder.DeliveryTag {
			panic(fmt.Errorf(
				"confirmation delivery tag %v does not line up with order" +
					" delivery tag %v",
					confirmation.DeliveryTag,
					thisOrder.DeliveryTag,
			))
		}

		// If this was an ack, return, closing the result channel.
		if confirmation.Ack {
			thisOrder.result <- nil
			return
		}

		// Otherwise return a nack error with orphan information
		thisOrder.result <- ErrProducerNack{Orphan: confirmation.DisconnectOrphan}
	}
}

// Publish a message.
func (producer *Producer) Publish(
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	msg amqp.Publishing,
) (err error) {
	// Create the order.
	order := &PublishOrder{
		// Save the args
		PublishArgs: PublishArgs{
			Exchange:  exchange,
			Key:       key,
			Mandatory: mandatory,
			Immediate: immediate,
			Msg:       msg,
		},
		// Set the delivery tag to 0, a proper tag will be set once publication occurs.
		DeliveryTag: 0,
		// Buffer the result channel by 1 so we never block the publication routine.
		result:    make(chan error, 1),
	}

	// We are going to use a closure to grab and release the lock so we don't hold onto
	// it longer than we have to.
	err = func() error {
		// Put a read hold on closing the order channel so it isn't closed out from
		// under us
		producer.publishLock.RLock()
		defer producer.publishLock.RUnlock()

		// Exit if the context has been cancelled.
		if producer.ctx.Err() != nil {
			return fmt.Errorf("publisher cancelled: %w", producer.ctx.Err())
		}

		// Push the order into our internal queue, or return an error if our context is
		// cancelled before there is room.
		select {
		case producer.publishQueue <- order:
		case <- producer.ctx.Done():
			return fmt.Errorf("publisher cancelled: %w", producer.ctx.Err())
		}

		return nil
	}()
	// If we got an error from a cancelled context, return it.
	if err != nil {
		return err
	}

	// Block until we have a final result, then return it.
	return <- order.result
}

// Run the producer, this method blocks until the producer has been shut down.
func (producer *Producer) Run() error {
	// Make sure we are not re-running an closed producer.
	if producer.ctx.Err() != nil {
		return fmt.Errorf(
			"cannot run cancelled producer: %w", producer.ctx.Err(),
		)
	}

	// Close the amqp channel when all work is done.
	defer producer.channel.Close()

	// If we are confirming publications, place the channel into confirm mode.
	if producer.opts.confirmPublish {
		err := producer.channel.Confirm(false)
		if err != nil {
			return fmt.Errorf("error placing channel into confirm mode: %w", err)
		}

		// Register our confirmation channel with the amqp channel.
		producer.channel.NotifyPublish(producer.publishEvents)
	}

	// Launch a monitor that will listen for an unexpected channel closure, and shutdown
	// the producer if it occurs.
	go func() {
		defer producer.StartShutdown()
		notifyClose := make(chan *amqp.Error, 1)
		producer.channel.NotifyClose(notifyClose)
		<-notifyClose
	}()

	// All our workers will release this group when the producer is done.
	complete := new(sync.WaitGroup)

	// Launch a monitor routine to close the internal order queue when the producer
	// context is cancelled. We run this in it's own routine rather than making it
	// part of the shutdown function so publishQueue is only closed once.
	complete.Add(1)
	go func() {
		defer complete.Done()
		// Unlock the lock, allowing new publishers to encounter the closed context and
		// error without panicking on a closed channel.
		defer producer.publishLock.Unlock()
		// Close the publish queue, allowing the publish routine to wrap up and spin
		// down.
		defer close(producer.publishQueue)
		// Grab the publish lock for write, this will block the publish function from
		// placing any new messages into the internal queue, and will allow any in the
		// process of doing so to finish before we close that queue, avoiding panics
		// from sending to a closed channel.
		defer producer.publishLock.Lock()

		// Wait for the producer context to be cancelled.
		<-producer.ctx.Done()
	}()

	// Launch the confirmation routine.
	complete.Add(1)
	go func() {
		defer complete.Done()
		producer.runConfirmations()
	}()

	// Launch the publisher routine.
	complete.Add(1)
	go func() {
		defer complete.Done()
		producer.runPublisher()
	}()

	// Wait for all worker routines to wrap up and exit.
	complete.Wait()
	return nil
}

// Begin the shutdown of the producer. This method may exit before remaining work is
// completed.
func (producer *Producer) StartShutdown() {
	// Cancel the main context
	defer producer.ctxCancel()
}

// Returns a new ProducerOpts with default options.
func NewProducerOpts() *ProducerOpts {
	return &ProducerOpts{
		confirmPublish: true,
		internalQueueSize: 128,
	}
}

// Create a new producer using the given amqpChannel and opts.
//
// amqpChannel should be a fresh channel that has been configured with the correct
// queues for the producer. It should have no messages published through it before it
// is passed to this method.
//
// Once the channel is passed to the producer, the producer should be considered to
// own it. No additional operations should be done through amqpChannel once it is
// passed to this method. The channel will be closed upon producer shutdown. A single
// channel should NOT be passed to multiple producers.
//
// If opts.confirmPublish is set to true, the channel will be put into confirm mode
// by the producer. There is no need to do so before passing it to this method.
//
// If opts is nil, default options will be used.
func NewProducer(amqpChannel *amqp.Channel, opts *ProducerOpts) *Producer {
	ctx, cancel := context.WithCancel(context.Background())

	if opts == nil {
		opts = NewProducerOpts()
	}

	return &Producer{
		ctx:           ctx,
		ctxCancel:     cancel,
		publishLock:   new(sync.RWMutex),
		channel:       amqpChannel,
		publishEvents: make(chan amqp.Confirmation, opts.internalQueueSize),
		publishQueue:  make(chan *PublishOrder, opts.internalQueueSize),
		confirmQueue:  make(chan *PublishOrder, opts.internalQueueSize),
		opts:          *opts,
	}
}
