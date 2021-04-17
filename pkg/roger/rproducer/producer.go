package rproducer

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"sync"
)

// Producer is a wrapper for an amqp channel that handles the boilerplate of confirming
// publishings, retrying nacked messages, and other useful quality of life features.
type Producer struct {
	// Context for the producer.
	ctx              context.Context
	ctxCancel        context.CancelFunc
	publishQueueLock *sync.RWMutex

	// The channel we will be running publishing on. This producer is expected to have
	// full ownership of this channel. Any additional operations that occur on this
	// channel after it is passed to the producer may result in unexpected behavior.
	channel *amqp.Channel
	// Channel receiving events from channel.NotifyPublish.
	publishEvents chan amqp.Confirmation

	// Internal Queue for messages waiting to be published.
	publishQueue chan *Publication
	// Internal Queue of messages that have been sent to the broker, but are awaiting
	// confirmation.
	confirmQueue chan *Publication

	// Caller options for producer behavior
	opts Opts
}

// runPublisher runs the main loop of the publisher.
func (producer *Producer) runPublisher() {
	// close the confirmation Queue on the way out, letting the confirmation wrap up
	// work and exit.
	defer close(producer.confirmQueue)

	// Keep track of the current delivery tag.
	deliveryTag := uint64(0)

	// Range over our publish Queue.
	for thisOrder := range producer.publishQueue {
		// If the context has expired on this publication, skip it.
		if thisOrder.ctx.Err() != nil {
			continue
		}

		err := producer.channel.Publish(
			thisOrder.args.Exchange,
			thisOrder.args.Key,
			thisOrder.args.Mandatory,
			thisOrder.args.Immediate,
			thisOrder.args.Msg,
		)
		if err != nil {
			// Return the error
			thisOrder.result <- ErrPublish{AmqpErr: err}
			continue
		}

		// Increment the delivery tag
		deliveryTag++
		// If we are not confirming publications, continue
		if !producer.opts.confirmPublish {
			continue
		}

		// If we are confirming orders, send this to the confirmation routine. We don't
		// want to check if the requester context has cancelled, as we already sent
		// the message, and need to tack it to make sure our confirmations line up
		thisOrder.publicationTag = deliveryTag
		producer.confirmQueue <- thisOrder
	}
}

// runConfirmations runs the confirmation listener, which listens to confirmation
// events from the amqp.Channel and routes them to the correct caller.
func (producer *Producer) runConfirmations() {
	// start shutdown if something goes horribly wrong here.
	defer producer.StartShutdown()

	// Iterate over our internal confirmation Queue
	for thisOrder := range producer.confirmQueue {
		// Get the next confirmation. Because confirmations are returned by the client
		// in order, we don't have to worry bout lining them up. We are getting
		// publications in order from both confirmQueue and
		confirmation, ok := <-producer.publishEvents
		if !ok {
			// If this channel is closed, then this is an orphan and we should send
			// a closed err. This channel will not be opened again.
			thisOrder.result <- amqp.ErrClosed
			return
		}

		if confirmation.DeliveryTag != thisOrder.publicationTag {
			panic(fmt.Errorf(
				"confirmation delivery tag %v does not line up with order"+
					" delivery tag %v",
				confirmation.DeliveryTag,
				thisOrder.publicationTag,
			))
		}

		// If this was an ack, send an ack result
		if confirmation.Ack {
			thisOrder.result <- nil
		} else {
			// Otherwise return a nack error with orphan information
			thisOrder.result <- ErrNack{Orphan: confirmation.DisconnectOrphan}
		}
	}
}

// Publish a message. This method is goroutine safe, even when confirming publications
// with the broker. When confirming messages with a broker, this method will block until
// the broker confirms the message publication or ctx is cancelled.
//
// Cancelling the context will cause this method to return, but have no other effect
// if the message has already been sent to thr broker and we are waiting for a response.
func (producer *Producer) Publish(
	ctx context.Context,
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	msg amqp.Publishing,
) (err error) {
	// Create the order.
	order := &Publication{
		// Use the producer context.
		ctx: ctx,
		// Save the args
		args: publishArgs{
			Exchange:  exchange,
			Key:       key,
			Mandatory: mandatory,
			Immediate: immediate,
			Msg:       msg,
		},
		// Set the delivery tag to 0, a proper tag will be set once publication occurs.
		publicationTag: 0,
		// Buffer the result channel by 1 so we never block the publication routine.
		result: make(chan error, 1),
	}

	if err = producer.queueOrder(order); err != nil {
		return fmt.Errorf("error queuing order: %w", err)
	}

	if err = order.WaitOnConfirmation(); err != nil {
		return fmt.Errorf("error waiting for order publication: %w", err)
	}

	return nil
}

// QueueForPublication is as Publish, but only puts the order into an internal queue
// before returning, allowing the user to wait on publication confirmation themselves
// through the returned Publication value.
//
// Useful when ensuring publication order is important.
func (producer *Producer) QueueForPublication(
	ctx context.Context,
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	msg amqp.Publishing,
) (order *Publication, err error) {
	// Create the order.
	order = &Publication{
		// Use the producer context.
		ctx: ctx,
		// Save the args
		args: publishArgs{
			Exchange:  exchange,
			Key:       key,
			Mandatory: mandatory,
			Immediate: immediate,
			Msg:       msg,
		},
		// Set the delivery tag to 0, a proper tag will be set once publication occurs.
		publicationTag: 0,
		// Buffer the result channel by 1 so we never block the publication routine.
		result: make(chan error, 1),
	}

	if err = producer.queueOrder(order); err != nil {
		return nil, fmt.Errorf("error queuing order: %w", err)
	}

	return order, nil
}

func (producer *Producer) queueOrder(order *Publication) error {
	// We are going to use a closure to grab and release the lock so we don't hold onto
	// it longer than we have to.
	err := func() error {
		// Put a read hold on closing the order channel so it isn't closed out from
		// under us.
		producer.publishQueueLock.RLock()
		defer producer.publishQueueLock.RUnlock()

		// Exit if the context has been cancelled.
		if producer.ctx.Err() != nil {
			return fmt.Errorf("publisher cancelled: %w", producer.ctx.Err())
		}

		// Push the order into our internal Queue, or return an error if our context is
		// cancelled before there is room.
		select {
		case producer.publishQueue <- order:
		case <-producer.ctx.Done():
			return fmt.Errorf("publisher cancelled: %w", producer.ctx.Err())
		case <-order.ctx.Done():
			return fmt.Errorf("message cancelled: %w", order.ctx.Err())
		}

		return nil
	}()

	// If we got an error from a cancelled context, return it.
	if err != nil {
		return err
	}

	return nil
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

	// Launch a monitor routine to close the internal order Queue when the producer
	// context is cancelled. We run this in it's own routine rather than making it
	// part of the shutdown function so publishQueue is only closed once.
	complete.Add(1)
	go func() {
		defer complete.Done()
		// Unlock the lock, allowing new publishers to encounter the closed context and
		// error without panicking on a closed channel.
		defer producer.publishQueueLock.Unlock()
		// Close the publish Queue, allowing the publish routine to wrap up and spin
		// down.
		defer close(producer.publishQueue)
		// Grab the publish lock for write, this will block the publish function from
		// placing any new messages into the internal Queue, and will allow any in the
		// process of doing so to finish before we close that Queue, avoiding panics
		// from sending to a closed channel.
		defer producer.publishQueueLock.Lock()

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

// StartShutdown begins the shutdown of the producer. This method may exit before
// remaining work is completed.
func (producer *Producer) StartShutdown() {
	// Cancel the main context
	defer producer.ctxCancel()
}

// New creates a new producer using the given amqp.Channel and opts.
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
func New(amqpChannel *amqp.Channel, opts *Opts) *Producer {
	ctx, cancel := context.WithCancel(context.Background())

	if opts == nil {
		opts = NewOpts()
	}

	return &Producer{
		ctx:              ctx,
		ctxCancel:        cancel,
		publishQueueLock: new(sync.RWMutex),
		channel:          amqpChannel,
		publishEvents:    make(chan amqp.Confirmation, opts.internalQueueCapacity),
		publishQueue:     make(chan *Publication, opts.internalQueueCapacity),
		confirmQueue:     make(chan *Publication, opts.internalQueueCapacity),
		opts:             *opts,
	}
}
