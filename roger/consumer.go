package roger

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	streadway "github.com/streadway/amqp"
	"sync"
)

// Interface that only exposes the Queue and exchange methods for a channel. These
// are the only methods we want SetupChannel to have access to.
type AmqpRouteManager interface {
	QueueDeclare(
		name string,
		durable bool,
		autoDelete bool,
		exclusive bool,
		noWait bool,
		args streadway.Table,
	) (queue amqp.Queue, err error)
	QueueDeclarePassive(
		name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table,
	) (queue amqp.Queue, err error)
	QueueInspect(name string) (queue amqp.Queue, err error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (count int, err error)

	ExchangeDeclare(
		name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table,
	) (err error)
	ExchangeDeclarePassive(
		name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table,
	) (err error)
	ExchangeBind(
		destination, key, source string, noWait bool, args amqp.Table,
	) (err error)
	ExchangeDelete(name string, ifUnused, noWait bool) (err error)
}

// Args the consumer will be created with.
type ConsumeArgs struct {
	// The Queue to consume from
	Queue string
	// The consumer name to identify this consumer with the broker.
	ConsumerName string
	// Whether the broker should ack messages automatically as it sends them. Otherwise
	// the consumer will handle acking messages.
	AutoAck bool
	// Whether this consumer should be the exclusive consumer for this Queue.
	Exclusive bool
	// Additional args to pass to the amqp.Channel.Consume() method.
	Args amqp.Table
}

// Interface for handling consuming from a route. Implementors of this interface will
// be registered with a consumer.
type AmqpDeliveryProcessor interface {
	// The amqp route key handler should be made with
	ConsumeArgs() *ConsumeArgs
	// This method is called before the consumer is created, and is designed to let
	// this handler declare any exchanges or queues necessary to handle deliveries.
	SetupChannel(ctx context.Context, amqpChannel AmqpRouteManager) error
	// This method will be called once per delivery. Returning a non-nil err will result
	// in it being logged and the delivery being nacked. If requeue is true, the nacked
	// delivery will be requeued. If err is nil, requeue is ignored.
	HandleDelivery(
		ctx context.Context, delivery amqp.Delivery, logger zerolog.Logger,
	) (err error, requeue bool)

	// Run at shutdown to allow the route handler to clean up any necessary resources.
	Cleanup(amqpChannel AmqpRouteManager) error
}

// Options for running a consumer.
type ConsumerOpts struct {
	// The maximum number of workers that can be active at one time.
	maxWorkers int

	// logger to use
	logger zerolog.Logger
}

// The maximum number of workers that can be running at the same time. If 0 or less,
// no limit will be used. Default: 0.
func (opts *ConsumerOpts) WithMaxWorkers(max int) *ConsumerOpts {
	opts.maxWorkers = max
	return opts
}

// Zerolog logger to use for logging. Defaults to a fresh logger with the global
// settings applied.
func (opts *ConsumerOpts) WithLogger(logger zerolog.Logger) *ConsumerOpts {
	opts.logger = logger
	return opts
}

// Returns new ConsumerOpts object with default settings.
func NewConsumerOpts() *ConsumerOpts {
	return new(ConsumerOpts).WithLogger(log.Logger)
}

// Consumer is a service helper for consuming messages from one or more queues.
type Consumer struct {
	// Context of the consumer. Cancelling this context will begin shutdown of the
	// consumer.
	ctx       context.Context
	cancelCtx context.CancelFunc

	// The channel we will be consuming from
	channel *amqp.Channel
	// Handlers registered to this consumer
	processors []AmqpDeliveryProcessor
	// Lock for processors.
	handlersLock sync.Mutex

	// Whether the consumer has been started.
	started bool
	// This channel will be used to throttle the number of workers we can have active
	// at a single time. Whenever a new delivery is received.
	workerThrottle chan struct{}

	// Caller options.
	opts ConsumerOpts

	// Logger.
	logger zerolog.Logger
}

// Register a consumer handler. Will panic if called after consumer start.
func (consumer *Consumer) RegisterProcessor(
	processor AmqpDeliveryProcessor,
) {
	consumer.handlersLock.Lock()
	defer consumer.handlersLock.Unlock()

	if consumer.started {
		panic(errors.New("tried to register processor for started consumer"))
	}

	consumer.processors = append(consumer.processors, processor)
}

// Handle a single delivery.
func (consumer *Consumer) handleDelivery(
	handler AmqpDeliveryProcessor,
	delivery amqp.Delivery,
	args *ConsumeArgs,
	logger zerolog.Logger,
	complete *sync.WaitGroup,
) {
	defer complete.Done()

	// If we are throttling workers, free a space on the throttle at the end of this
	// work.
	if consumer.opts.maxWorkers > 0 {
		defer func() {
			<-consumer.workerThrottle
		}()
	}

	// Create a logger for this delivery.
	logger = logger.With().
		Uint64("DELIVERY_TAG", delivery.DeliveryTag).
		Str("MESSAGE_ID", delivery.MessageId).
		Logger()

	// Create a context for this delivery derived from the consumer context.
	deliveryCtx, deliveryCancel := context.WithCancel(consumer.ctx)
	defer deliveryCancel()
	err, requeueDelivery := handler.HandleDelivery(deliveryCtx, delivery, logger)
	// Log any errors returned
	if err != nil {
		logger.Error().Err(err).Msg("error handling delivery")
	}

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

	// Log any acknowledgement errors.
	if err != nil {
		logger.Error().
			Bool("ACK", ack).
			Bool("REQUEUE", requeueDelivery).
			Err(err).
			Msg("error acknowledging delivery")
	}
}

// Runs an individual consumer.
func (consumer *Consumer) runProcessor(
	handler AmqpDeliveryProcessor,
	complete *sync.WaitGroup,
) {
	// Release work complete WaitGroup on exit.
	defer complete.Done()

	// Cache the consumer args in a var
	args := handler.ConsumeArgs()

	// Setup a logger for this consumer.
	consumerLogger := consumer.logger.With().
		Str("CONSUMER", args.ConsumerName).
		Logger()

	// Create the consumer channel we will pull deliveries over.
	consumerChan, err := consumer.channel.Consume(
		args.Queue,
		args.ConsumerName,
		args.AutoAck,
		args.Exclusive,
		false,
		false,
		args.Args,
	)
	if err != nil {
		// If there is an error here, log it and bring down the whole consumer. This is
		// a critical setup stage happening in it's own goroutine. It is better to fail
		// fast here and nuke the consumer if something goes wrong.
		defer consumer.StartShutdown()
		consumerLogger.Error().
			Err(err).
			Msg("error creating channel consumer. triggering shutdown")
		return
	}

	// WaitGroup for workers to close when done.
	workersComplete := new(sync.WaitGroup)

	// Pull all deliveries down from the consumer.
	var delivery amqp.Delivery
deliveriesLoop:
	for {
		// Pull a delivery or exit on context close.
		select {
		case <-consumer.ctx.Done():
			break deliveriesLoop
		case delivery = <-consumerChan:
		}

		// If we are throttling simultaneous worker count, we need to push to the
		// throttle channel, if the channel is full (max workers reached) this will
		// block.
		if consumer.opts.maxWorkers > 0 {
			consumer.workerThrottle <- struct{}{}
		}

		// Launch a delivery handler.
		workersComplete.Add(1)
		go consumer.handleDelivery(
			handler, delivery, args, consumerLogger, workersComplete,
		)
	}

	// Wait for all outstanding workers to be complete
	workersComplete.Wait()

	// Shut down the consumer
	err = handler.Cleanup(consumer.channel)
	if err != nil {
		// Log any shutdown error
		consumerLogger.Error().Err(err).Msg("error running cleanup")
	}
}

// runs all the channel setups from our registered processors.
func (consumer *Consumer) runHandlerSetups() error {
	// Let all the processors run their setup script and return an error if.
	for _, thisHandler := range consumer.processors {
		var err error
		func() {
			ctx, cancel := context.WithCancel(consumer.ctx)
			defer cancel()
			err = thisHandler.SetupChannel(ctx, consumer.channel)
		}()
		if err != nil {
			return fmt.Errorf(
				"error setting up channel for consumer %v: %w",
				thisHandler.ConsumeArgs().ConsumerName,
				err,
			)
		}
	}

	return nil
}

// Run the consumer. This method blocks until the consumer has completed shutdown.
func (consumer *Consumer) Run() error {
	// Close the channel on the way out
	defer consumer.channel.Close()
	defer consumer.StartShutdown()

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
	consumersComplete := new(sync.WaitGroup)

	// Wait on this before exiting
	defer consumersComplete.Wait()

	// Launch the individual consumers.
	for _, thisHandler := range consumer.processors {
		consumersComplete.Add(1)
		go consumer.runProcessor(thisHandler, consumersComplete)
	}

	// Return.
	return nil
}

// Start the shutdown of the Consumer. This method will return immediately. It does
// not block until shutdown is complete.
func (consumer *Consumer) StartShutdown() {
	defer consumer.cancelCtx()
}

func NewConsumer(
	channel *amqp.Channel, opts *ConsumerOpts,
) *Consumer {
	if opts == nil {
		opts = NewConsumerOpts()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		ctx:            ctx,
		cancelCtx:      cancel,
		channel:        channel,
		processors:     nil,
		handlersLock:   sync.Mutex{},
		started:        false,
		workerThrottle: make(chan struct{}, opts.maxWorkers),
		opts:           *opts,
		logger:         opts.logger,
	}
}
