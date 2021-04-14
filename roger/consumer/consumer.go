package consumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"sync"
	"time"
)

// Consumer is a service helper for consuming messages from one or more queues.
type Consumer struct {
	// Context of the consumer. Cancelling this context will begin shutdown of the
	// consumer.
	ctx       context.Context
	cancelCtx context.CancelFunc

	// The channel we will be consuming from
	channel *amqp.Channel
	// Handlers registered to this consumer
	processors []deliveryProcessor
	// Lock for processors.
	handlersLock sync.Mutex

	// Whether the consumer has been started.
	started bool
	// This channel will be used to throttle the number of workers we can have active
	// at a single time. Whenever a new delivery is received.
	workerThrottle chan struct{}

	// errChan is the channel we should unrecoverable process errors on.
	processorErrs chan error

	// Caller options.
	opts Opts
}

// RegisterProcessor registers a AmqpDeliveryProcessor implementation value. Will panic
// if called after consumer start.
func (consumer *Consumer) RegisterProcessor(
	processor AmqpDeliveryProcessor,
) error {
	consumer.handlersLock.Lock()
	defer consumer.handlersLock.Unlock()

	if consumer.started {
		panic(errors.New("tried to register processor for started consumer"))
	}

	wrappedProcessor, err := newDeliveryProcessor(processor, consumer.opts)
	if err != nil {
		return err
	}

	consumer.processors = append(consumer.processors, wrappedProcessor)
	return nil
}

// handleDelivery handles a single delivery.
func (consumer *Consumer) handleDelivery(
	handler deliveryProcessor,
	delivery datamodels.Delivery,
	args AmqpArgs,
	done *sync.WaitGroup,
) {
	defer done.Done()
	defer func() {
		// If we are throttling workers, free a space on the throttle at the end
		// of this work.
		if consumer.opts.maxWorkers > 0 {
			defer func() {
				<-consumer.workerThrottle
			}()
		}
	}()

	// Create a context for this delivery derived from the consumer context.
	deliveryCtx, deliveryCancel := context.WithCancel(consumer.ctx)
	defer deliveryCancel()
	err, requeueDelivery := handler.HandleDelivery(deliveryCtx, delivery)

	// If we are auto-acking, continue.
	if args.AutoAck {
		return
	}

	// Otherwise ack non-error returns and nack error returns.
	ack := err == nil
	if ack {
		err = delivery.Ack(false)
	} else {
		err = delivery.Nack(false, requeueDelivery)
	}
}

// runProcessor runs an individual consumer pulling from an amqp.Channel.Consume
// channel.
func (consumer *Consumer) runProcessor(
	processor deliveryProcessor,
	complete *sync.WaitGroup,
) {
	// Release work complete WaitGroup on exit.
	defer complete.Done()
	var err error

	// We need to communicate any unexpected errors to the main thread. Only th first
	// such error will be fetched, so we can move on if there is not a listener waiting
	// for it.
	defer func() {
		if err == nil {
			return
		}
		select {
		case consumer.processorErrs <- err:
		default:
		}
	}()

	consumer.processDeliveries(processor)
	// If the processor exited before the main context cancellation, something went
	// wrong.
	if consumer.ctx.Err() == nil {
		err = fmt.Errorf(
			"processor '%v' exited before shutdown",
			processor.AmqpArgs.ConsumerName,
		)
	}

	// Shut down the consumer with a 30 second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = processor.CleanupChannel(ctx, consumer.channel)
	if err != nil {
		err = fmt.Errorf(
			"error cleaning up channels for '%v': %w", processor.AmqpArgs.Args, err,
		)
	}
}

func (consumer *Consumer) processDeliveries(processor deliveryProcessor) {
	// WaitGroup for workers to close when done.
	workersComplete := new(sync.WaitGroup)
	// Wait for all outstanding workers to be complete before we exit
	defer workersComplete.Wait()

	// Pull all deliveries down from the consumer.
	var delivery datamodels.Delivery
	useThrottle := consumer.opts.maxWorkers > 0

	for {
		// Pull a delivery or exit on context close.
		select {
		case <-consumer.ctx.Done():
			return
		case delivery = <-processor.consumeChan:
		}

		// If we are throttling simultaneous worker count, we need to push to the
		// throttle channel, if the channel is full (max workers reached) this will
		// block.
		if useThrottle {
			select {
			case consumer.workerThrottle <- struct{}{}:
			case <-consumer.ctx.Done():
				// Nack this in it's own routine and exit.
				go delivery.Nack(false, true)
				return
			}
		}

		workersComplete.Add(1)
		go consumer.handleDelivery(processor, delivery, processor.AmqpArgs, workersComplete)
	}
}

// runHandlerSetups runs all the amqp.Channel setups from our registered processors.
func (consumer *Consumer) runHandlerSetups() error {
	// Let all the processors run their setup script and return an error if.
	for i, thisProcessor := range consumer.processors {
		err := thisProcessor.SetupChannel(consumer.ctx, consumer.channel)
		if err != nil {
			return fmt.Errorf(
				"error setting up channel for consumer %v: %w",
				thisProcessor.AmqpArgs.ConsumerName,
				err,
			)
		}

		// Create the consumer channel we will pull deliveries over.
		thisProcessor.consumeChan, err = consumer.channel.Consume(
			thisProcessor.AmqpArgs.Queue,
			thisProcessor.AmqpArgs.ConsumerName,
			thisProcessor.AmqpArgs.AutoAck,
			thisProcessor.AmqpArgs.Exclusive,
			false,
			false,
			thisProcessor.AmqpArgs.Args,
		)
		if err != nil {
			return fmt.Errorf(
				"error consuming from queue '%v': %w", thisProcessor.AmqpArgs.ConsumerName, err,
			)
		}
		consumer.processors[i] = thisProcessor
	}

	return nil
}

// Run the consumer. This method blocks until the consumer has completed shutdown.
func (consumer *Consumer) Run() error {
	// Close the channel on the way out
	defer consumer.channel.Close()
	defer consumer.StartShutdown()
	var processorErrs error

	// Launch a routine to listen for unexpected processor errors.
	errsCollected := make(chan struct{})
	go func() {
		defer close(errsCollected)
		defer consumer.StartShutdown()
		processorErrs = <-consumer.processorErrs
	}()

	// Mark the consumer as started so we can't register any more processors.
	func() {
		consumer.handlersLock.Lock()
		defer consumer.handlersLock.Unlock()
		consumer.started = true
	}()

	// Create a monitor routine to signal shutdown of the consumer in case the
	// rabbitMQ channel unexpectedly closes
	go func() {
		defer consumer.StartShutdown()
		notifyClose := make(chan *amqp.Error, 1)
		consumer.channel.NotifyClose(notifyClose)
		<-notifyClose
	}()

	// Run all the channel setups from our processors.
	err := consumer.runHandlerSetups()
	if err != nil {
		return err
	}

	// Create a WaitGroup to wait for all work to be complete.
	processorsDone := new(sync.WaitGroup)

	// Launch the individual consumers.
	for _, thisProcessor := range consumer.processors {
		processorsDone.Add(1)
		go consumer.runProcessor(thisProcessor, processorsDone)
	}

	// Wait for the processors to all exit.
	processorsDone.Wait()

	// Wait for error collection to be done.
	close(consumer.processorErrs)
	<-errsCollected

	// Return any unexpected errors.
	return processorErrs
}

// StartShutdown beings shutdown of the Consumer. This method will return immediately,
// it does not block until shutdown is complete.
func (consumer *Consumer) StartShutdown() {
	defer consumer.cancelCtx()
}

// New returns a new consumer which will pull deliveries from the passed amqp.Channel.
func New(channel *amqp.Channel, opts Opts) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		ctx:            ctx,
		cancelCtx:      cancel,
		channel:        channel,
		processors:     nil,
		handlersLock:   sync.Mutex{},
		started:        false,
		workerThrottle: make(chan struct{}, opts.maxWorkers),
		processorErrs:  make(chan error),
		opts:           opts,
	}
}
