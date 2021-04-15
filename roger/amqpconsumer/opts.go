package amqpconsumer

import (
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/internal"
	"github.com/peake100/rogerRabbit-go/roger/amqpconsumer/middleware"
	"github.com/rs/zerolog"
)

// Opts holds options for running a consumer.
type Opts struct {
	// The maximum number of workers that can be active at one time.
	maxWorkers int

	// The middleware we should add to each AmqpDeliveryProcessor.
	middleware Middleware

	// noLoggingMiddleware is true if we should not add logging middleware.
	noLoggingMiddleware bool
	// logger to use in the default logging middleware.
	logger zerolog.Logger
	// logSuccessLevel is the level at which the default logger should log a
	// successfully processed delivery.
	logSuccessLevel zerolog.Level
	// logSuccessLevel is the level at which the default logger should log the full
	// delivery object.
	logDeliveryLevel zerolog.Level
}

// it's correct that we are modifying value-receivers here. Disable the revive check
// for this.

// revive:disable:modifies-value-receiver

// WithMaxWorkers sets the maximum number of workers that can be running at the same
// time. If 0 or less, no limit will be used.
//
// Default: 0.
func (opts Opts) WithMaxWorkers(max int) Opts {
	opts.maxWorkers = max
	return opts
}

// WithMiddleware sets the Middleware to use. Default: includes all default middleware
// in the consumer/middleware package.
//
// Default: Middleware{}
func (opts Opts) WithMiddleware(processorMiddleware Middleware) Opts {
	opts.middleware = processorMiddleware
	return opts
}

// WithDefaultLogging enables the default zerolog.Logger logging middleware. If false
// all other logging settings have no effect.
//
// Default: true
func (opts Opts) WithDefaultLogging(log bool) Opts {
	opts.noLoggingMiddleware = !log
	return opts
}

// WithLogger sets the zerolog.Logger for the default logging middleware to use
// If WithDefaultLogging is false, this setting has no effect.
//
// Default: lockless, pretty-printed logger set to Info level.
func (opts Opts) WithLogger(logger zerolog.Logger) Opts {
	opts.logger = logger
	return opts.WithLoggingLevel(logger.GetLevel())
}

// WithLoggingLevel sets the level of the logger passed to WithLogger.
func (opts Opts) WithLoggingLevel(level zerolog.Level) Opts {
	opts.logger = opts.logger.Level(level)
	return opts
}

// WithLogSuccessLevel is the minimum logging level to log a successful delivery at.
//
// Default: zerolog.DebugLevel.
func (opts Opts) WithLogSuccessLevel(level zerolog.Level) Opts {
	opts.logSuccessLevel = level
	return opts
}

// WithLogDeliveryLevel is the minimum logging level to log the full delivery object at.
//
// Default: zerolog.DebugLevel.
func (opts Opts) WithLogDeliveryLevel(level zerolog.Level) Opts {
	opts.logDeliveryLevel = level
	return opts
}

// revive:enable:modifies-value-receiver

// DefaultOpts returns new Opts object with default settings.
func DefaultOpts() Opts {
	opts := Opts{
		maxWorkers: 0,
		middleware: Middleware{
			providers: make(map[middleware.ProviderTypeId]struct{}),
		},
		noLoggingMiddleware: false,
		logger:              internal.CreateDefaultLogger(zerolog.InfoLevel),
		logSuccessLevel:     zerolog.DebugLevel,
		logDeliveryLevel:    zerolog.ErrorLevel,
	}

	return opts
}

// Middleware holds the middleware to register on a consumer.
type Middleware struct {
	// setupChannel is all SetupChannel middlewares to register on the processor.
	setupChannel []middleware.SetupChannel
	// delivery is all Delivery middlewares to register on the processor.
	delivery []middleware.Delivery
	// cleanupChannel is all CleanupChannel middlewares to register on the processor.
	cleanupChannel []middleware.CleanupChannel

	// providerFactories are middleware provider constructors to run for each processor
	// passed to a consumer.
	providerFactories []func() middleware.ProvidesMiddleware

	// providers tracks the type IDs of providers passed to this config.
	providers map[middleware.ProviderTypeId]struct{}
}

// AddSetupChannel adds a middleware.SetupChannel to be added to each
// AmqpDeliveryProcessor.SetupChannel passed to a Consumer.
func (config *Middleware) AddSetupChannel(processorMiddleware middleware.SetupChannel) {
	config.setupChannel = append(config.setupChannel, processorMiddleware)
}

// AddDelivery adds a middleware.Delivery to be added to each
// AmqpDeliveryProcessor.HandleDelivery passed to a Consumer.
func (config *Middleware) AddDelivery(processorMiddleware middleware.Delivery) {
	config.delivery = append(config.delivery, processorMiddleware)
}

// AddCleanupChannel adds a middleware.CleanupChannel to be added to each
// AmqpDeliveryProcessor.CleanupChannel passed to a Consumer.
func (config *Middleware) AddCleanupChannel(processorMiddleware middleware.CleanupChannel) {
	config.cleanupChannel = append(config.cleanupChannel, processorMiddleware)
}

// AddProvider adds consume middleware provided by methods of provider.
func (config *Middleware) AddProvider(provider middleware.ProvidesMiddleware) error {
	return config.addProviderMethods(provider)
}

// AddCleanupChannel adds a middleware.CleanupChannel to be added to each
// AmqpDeliveryProcessor.CleanupChannel passed to a Consumer.
func (config *Middleware) addProviderMethods(provider middleware.ProvidesMiddleware) error {
	// Check if this provider has already been registered.
	if _, ok := config.providers[provider.TypeID()]; ok {
		return amqp.ErrDuplicateProvider
	}

	// Register it's methods.
	methodsFound := false
	if hasMethod, ok := provider.(middleware.ProvidesSetupChannel); ok {
		config.AddSetupChannel(hasMethod.SetupChannel)
		methodsFound = true
	}

	if hasMethod, ok := provider.(middleware.ProvidesDelivery); ok {
		config.AddDelivery(hasMethod.Delivery)
		methodsFound = true
	}

	if hasMethod, ok := provider.(middleware.ProvidesCleanupChannel); ok {
		config.AddCleanupChannel(hasMethod.CleanupChannel)
		methodsFound = true
	}

	if !methodsFound {
		return amqp.ErrNoMiddlewareMethods
	}

	// Add the provider to our cache.
	config.providers[provider.TypeID()] = struct{}{}

	return nil
}

// buildProviderFactories creates new providers from all registered factories and
// registers their methods.
func (config *Middleware) buildProviderFactories() error {
	for _, thisFactory := range config.providerFactories {
		provider := thisFactory()
		err := config.addProviderMethods(provider)
		if err != nil {
			return fmt.Errorf(
				"could not reegister middleware provider '%v': %w", provider.TypeID(), err,
			)
		}
	}

	return nil
}
