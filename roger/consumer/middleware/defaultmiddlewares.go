package middleware

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	"time"
)

// DefaultLoggerKey is the context value key for fetching the logger provided by the
// DefaultLogging middleware.
const DefaultLoggerKey = "ConsumerDefaultLogger"

const DefaultLoggerTypeID ProviderTypeId = "DefaultLogger"

// ctxWithLogger adds a logger provided by DefaultLogging to a context.
func ctxWithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, DefaultLoggerKey, logger)
}

// GetDefaultLogger fetched the default logger from ctx.
func GetDefaultLogger(ctx context.Context) (logger zerolog.Logger, ok bool) {
	logger, ok = ctx.Value(DefaultLoggerKey).(zerolog.Logger)
	return logger, ok
}

// DefaultLogging provides logging middleware for roger.Consumer.
type DefaultLogging struct {
	Logger           zerolog.Logger
	LogDeliveryLevel zerolog.Level
	SuccessLogLevel  zerolog.Level
}

// TypeID implement ProvidesMiddleware.
func (middleware DefaultLogging) TypeID() ProviderTypeId {
	return DefaultLoggerTypeID
}

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
				Msg("AMQP Channel Cleanup complete")
		}
		return err
	}
}

func (middleware DefaultLogging) Delivery(next HandlerDelivery) HandlerDelivery {
	return func(ctx context.Context, delivery datamodels.Delivery) (err error, requeue bool) {
		logger := middleware.Logger.With().
			Str("HANDLER", "Delivery").
			Uint64("DELIVERY_TAG", delivery.DeliveryTag).
			Logger()
		ctx = ctxWithLogger(ctx, logger)

		start := time.Now().UTC()
		err, requeue = next(ctx, delivery)

		var event *zerolog.Event
		var eventLevel zerolog.Level
		if err != nil {
			event = logger.Err(err)
			eventLevel = zerolog.ErrorLevel
		} else {
			event = logger.WithLevel(middleware.SuccessLogLevel)
			eventLevel = middleware.SuccessLogLevel
		}

		if !event.Enabled() {
			return err, requeue
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

		return err, requeue
	}
}

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
func NewDefaultLogger(
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
