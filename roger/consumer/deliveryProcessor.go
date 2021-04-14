package consumer

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/peake100/rogerRabbit-go/roger/consumer/middleware"
)

// AmqpArgs are the args the consumer will be created with by calling amqp.Channel.Args.
type AmqpArgs struct {
	// Queue is the name of the Queue to consume from
	Queue string
	// ConsumerName identifies this consumer with the broker.
	ConsumerName string
	// AutoAck is whether the broker should ack messages automatically as it sends them.
	// Otherwise the consumer will handle acking messages.
	AutoAck bool
	// Exclusive is whether this consumer should be the exclusive consumer for this
	// Queue.
	Exclusive bool
	// Args are additional args to pass to the amqp.Channel.Consume() method.
	Args amqp.Table
}

// AmqpDeliveryProcessor is an interface for handling consuming from a route.
// Implementors of this interface will be registered with a consumer.
type AmqpDeliveryProcessor interface {
	// AmqpArgs returns the args that amqp.Channel.Consume should be called with.
	AmqpArgs() AmqpArgs

	// SetupChannel is called before the consumer is created, and is designed to let
	// this handler declare any exchanges or queues necessary to handle deliveries.
	SetupChannel(ctx context.Context, amqpChannel middleware.AmqpRouteManager) error

	// HandleDelivery will be called once per delivery. Returning a non-nil err will
	// result in it being logged and the delivery being nacked. If requeue is true, the
	// nacked delivery will be requeued. If err is nil, requeue is ignored.
	HandleDelivery(ctx context.Context, delivery datamodels.Delivery) (err error, requeue bool)

	// CleanupChannel is called at shutdown to allow the route handler to clean up any
	// necessary resources.
	CleanupChannel(ctx context.Context, amqpChannel middleware.AmqpRouteManager) error
}

// deliveryProcessor is a delivery processor with middleware-wrapped handlers. This is
// what we will actually run in the consumer.
type deliveryProcessor struct {
	// AmqpArgs is the result of AmqpDeliveryProcessor.AmqpArgs.
	AmqpArgs AmqpArgs
	// SetupChannel is the user-provided AmqpDeliveryProcessor.SetupChannel wrapped in
	// middleware.
	SetupChannel middleware.HandlerSetupChannel
	// HandleDelivery is the user-provided AmqpDeliveryProcessor.HandleDelivery wrapped
	// in middleware.
	HandleDelivery middleware.HandlerDelivery
	// CleanupChannel is the user-provided AmqpDeliveryProcessor.CleanupChannel wrapped
	// in middleware.
	CleanupChannel middleware.HandlerCleanupChannel

	// consumeChan is created during the setup process so we can handle errors there
	// and stored here.
	consumeChan <-chan amqp.Delivery
}

// newDeliveryProcessor creates a new deliveryProcessor from a AmqpDeliveryProcessor and the middlewares in opts.
func newDeliveryProcessor(coreProcessor AmqpDeliveryProcessor, opts Opts) (deliveryProcessor, error) {
	processor := deliveryProcessor{}
	processor.AmqpArgs = coreProcessor.AmqpArgs()

	if !opts.noLoggingMiddleware {
		logger := opts.logger.With().Str(":CONSUMER", processor.AmqpArgs.ConsumerName).Logger()
		loggingMiddleware := middleware.NewDefaultLogger(logger, opts.logDeliveryLevel, opts.logSuccessLevel)

		if err := opts.middleware.AddProvider(loggingMiddleware); err != nil {
			return processor, fmt.Errorf("error rergistering default logging middleware: %w", err)
		}
	}

	// Setup the core handlers.
	processor.SetupChannel = coreProcessor.SetupChannel
	processor.HandleDelivery = coreProcessor.HandleDelivery
	processor.CleanupChannel = coreProcessor.CleanupChannel

	// Apply the middlewares.
	for _, thisMiddleware := range opts.middleware.setupChannel {
		processor.SetupChannel = thisMiddleware(processor.SetupChannel)
	}

	for _, thisMiddleware := range opts.middleware.delivery {
		processor.HandleDelivery = thisMiddleware(processor.HandleDelivery)
	}

	for _, thisMiddleware := range opts.middleware.cleanupChannel {
		processor.CleanupChannel = thisMiddleware(processor.CleanupChannel)
	}

	return processor, nil
}
