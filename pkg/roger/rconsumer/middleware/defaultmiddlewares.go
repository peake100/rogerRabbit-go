package middleware

import (
	"context"
	"errors"
	"github.com/peake100/pears-go/pkg/pears"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
	"github.com/rs/zerolog"
	"time"
)

// DefaultLoggerKey is the context value key for fetching the logger provided by the
// DefaultLogging middleware.
const DefaultLoggerKey = amqpmiddleware.MetadataKey("ConsumerDefaultLogger")

// DefaultLoggerTypeID is the ProviderTypeID for DefaultLogging.
const DefaultLoggerTypeID ProviderTypeID = "DefaultLogger"

// ctxWithLogger adds a logger provided by DefaultLogging to a context.
func ctxWithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, DefaultLoggerKey, logger)
}

// DefaultLogging provides logging middleware for roger.Consumer.
type DefaultLogging struct {
	Logger           zerolog.Logger
	LogDeliveryLevel zerolog.Level
	SuccessLogLevel  zerolog.Level
}

// TypeID implement ProvidesMiddleware.
func (middleware DefaultLogging) TypeID() ProviderTypeID {
	return DefaultLoggerTypeID
}

// SetupChannel implements ProvidesSetupChannel for logging.
func (middleware DefaultLogging) SetupChannel(next HandlerSetupChannel) HandlerSetupChannel {
	return func(ctx context.Context, amqpChannel AmqpRouteManager) error {
		logger := middleware.Logger.With().Str("HANDLER", "SetupChannel").Logger()
		ctx = ctxWithLogger(ctx, logger)
		logger.Debug().Msg("Setting Up AMQP Channel")

		start := time.Now().UTC()
		err := next(ctx, amqpChannel)
		if err != nil {
			logger.Err(err).
				TimeDiff("DURATION", time.Now().UTC(), start)
		} else {
			logger.Debug().
				TimeDiff("DURATION", time.Now().UTC(), start).
				Msg("AMQP Channel Setup complete")
		}
		return err
	}
}

// Delivery implements ProvidesDelivery for logging.
func (middleware DefaultLogging) Delivery(next HandlerDelivery) HandlerDelivery {
	return func(ctx context.Context, delivery amqp.Delivery) (requeue bool, err error) {
		logger := middleware.Logger.With().
			Str("HANDLER", "Delivery").
			Uint64("DELIVERY_TAG", delivery.DeliveryTag).
			Logger()
		ctx = ctxWithLogger(ctx, logger)

		start := time.Now().UTC()
		requeue, err = next(ctx, delivery)
		middleware.logDeliveryResult(delivery, requeue, err, start, logger)

		return requeue, err
	}
}

// logDeliveryResult logs the result of a delivery handler to logger.
func (middleware DefaultLogging) logDeliveryResult(
	delivery amqp.Delivery,
	requeue bool,
	err error,
	start time.Time,
	logger zerolog.Logger,
) {
	var event *zerolog.Event
	var eventLevel zerolog.Level
	if err != nil {
		event = logger.Err(err)
		var errPanic pears.PanicError
		if errors.As(err, &errPanic) {
			event.Str("STACKTRACE", errPanic.StackTrace)
		}
		eventLevel = zerolog.ErrorLevel
	} else {
		event = logger.WithLevel(middleware.SuccessLogLevel)
		eventLevel = middleware.SuccessLogLevel
	}

	if !event.Enabled() {
		return
	}

	event.TimeDiff("DURATION", time.Now().UTC(), start).Bool("REQUEUE", requeue)
	if middleware.LogDeliveryLevel <= eventLevel {
		event.Interface("DELIVERY", delivery)
	}

	if err != nil {
		event.Msg("delivery processed")
	} else {
		event.Send()
	}
}

// CleanupChannel implements ProvidesCleanupChannel for logging.
func (middleware DefaultLogging) CleanupChannel(next HandlerCleanupChannel) HandlerCleanupChannel {
	return func(ctx context.Context, amqpChannel AmqpRouteManager) error {
		logger := middleware.Logger.With().Str("HANDLER", "CleanupChannel").Logger()
		ctx = ctxWithLogger(ctx, logger)
		logger.Debug().Msg("Cleaning up AMQP Channel")

		start := time.Now().UTC()
		err := next(ctx, amqpChannel)
		if err != nil {
			logger.Err(err).
				TimeDiff("DURATION", time.Now().UTC(), start)
		} else {
			logger.Debug().
				TimeDiff("DURATION", time.Now().UTC(), start).
				Msg("AMQP Channel Cleanup complete")
		}

		return err
	}
}

// NewDefaultLogging returns a new DefaultLogging as a ProvidesMiddleware interface.
func NewDefaultLogging(
	logger zerolog.Logger,
	logDeliveryLevel zerolog.Level,
	successLogLevel zerolog.Level,
) ProvidesMiddleware {
	return DefaultLogging{
		Logger:           logger,
		LogDeliveryLevel: logDeliveryLevel,
		SuccessLogLevel:  successLogLevel,
	}
}
