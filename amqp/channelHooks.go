package amqp

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
)

// HOOK DEFINITIONS
type HookReconnect = func(
	next func() (*streadway.Channel, error), logger zerolog.Logger,
) (*streadway.Channel, error)

type HookQueueDeclare = func(
	next func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error

type HookQueueDelete = func(
	next func() error, args *QueueDeleteArgs, logger zerolog.Logger,
) error

type HookQueueBind = func(
	next func() error, args *QueueBindArgs, logger zerolog.Logger,
) error

type HookQueueUnbind = func(
	next func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error

type HookExchangeDeclare func(
	next func() error, args *ExchangeDeclareArgs, logger zerolog.Logger,
) error

type HookExchangeDelete func(
	next func() error, args *ExchangeDeleteArgs, logger zerolog.Logger,
) error

type HookExchangeBind func(
	next func() error, args *ExchangeBindArgs, logger zerolog.Logger,
) error

type HookExchangeUnbind func(
	next func() error, args *ExchangeUnbindArgs, logger zerolog.Logger,
) error

type channelHooks struct {
	reconnect       []HookReconnect
	queueDeclare    []HookQueueDeclare
	queueDelete     []HookQueueDelete
	queueBind       []HookQueueBind
	queueUnbind     []HookQueueUnbind
	exchangeDeclare []HookExchangeDeclare
	exchangeDelete  []HookExchangeDelete
	exchangeBind    []HookExchangeBind
	exchangeUnbind  []HookExchangeUnbind

	lock *sync.RWMutex
}

func (hooks *channelHooks) RegisterReconnect(hook HookReconnect) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.reconnect = append(hooks.reconnect, hook)
}

func (hooks *channelHooks) runHooksReconnect(
	reconnect func() (*streadway.Channel, error),
	logger zerolog.Logger,
) (*streadway.Channel, error) {
	topMethod := reconnect

	for _, thisHook := range hooks.reconnect {
		topMethod = func() (*streadway.Channel, error) {
			return thisHook(topMethod, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterQueueDeclare(hook HookQueueDeclare) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.queueDeclare = append(hooks.queueDeclare, hook)
}

func (hooks *channelHooks) runHooksQueueDeclare(
	runMethodOnce func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.queueDeclare {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterQueueDelete(hook HookQueueDelete) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.queueDelete = append(hooks.queueDelete, hook)
}

func (hooks *channelHooks) runHooksQueueDelete(
	runMethodOnce func() error, args *QueueDeleteArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.queueDelete {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterQueueBind(hook HookQueueBind) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.queueBind = append(hooks.queueBind, hook)
}

func (hooks *channelHooks) runHooksQueueBind(
	runMethodOnce func() error, args *QueueBindArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.queueBind {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterQueueUnbind(hook HookQueueUnbind) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.queueUnbind = append(hooks.queueUnbind, hook)
}

func (hooks *channelHooks) runHooksQueueUnbind(
	runMethodOnce func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.queueUnbind {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterExchangeDeclare(hook HookExchangeDeclare) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.exchangeDeclare = append(hooks.exchangeDeclare, hook)
}

func (hooks *channelHooks) runHooksExchangeDeclare(
	runMethodOnce func() error, args *ExchangeDeclareArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.exchangeDeclare {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterExchangeDelete(hook HookExchangeDelete) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.exchangeDelete = append(hooks.exchangeDelete, hook)
}

func (hooks *channelHooks) runHooksExchangeDelete(
	runMethodOnce func() error, args *ExchangeDeleteArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.exchangeDelete {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterExchangeBind(hook HookExchangeBind) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.exchangeBind = append(hooks.exchangeBind, hook)
}

func (hooks *channelHooks) runHooksExchangeBind(
	runMethodOnce func() error, args *ExchangeBindArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.exchangeBind {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

func (hooks *channelHooks) RegisterExchangeUnbind(hook HookExchangeUnbind) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()

	hooks.exchangeUnbind = append(hooks.exchangeUnbind, hook)
}

func (hooks *channelHooks) runHooksExchangeUnbind(
	runMethodOnce func() error, args *ExchangeUnbindArgs, logger zerolog.Logger,
) error {
	topMethod := runMethodOnce

	for _, thisHook := range hooks.exchangeUnbind {
		topMethod = func() error {
			return thisHook(topMethod, args, logger)
		}
	}

	return topMethod()
}

// This object implements hooks for re-declaring queues, exchanges, and bindings upon
// reconnect.
type routeDeclarationHooks struct {
	// List of queues that must be declared upon re-establishing the channel. We use a
	// map so we can remove queues from this list on queue delete.
	declareQueues *sync.Map
	// List of exchanges that must be declared upon re-establishing the channel. We use
	// a map so we can remove exchanges from this list on exchange delete.
	declareExchanges *sync.Map
	// List of bindings to re-build on channel re-establishment.
	bindQueues []*QueueBindArgs
	// Lock that must be acquired to alter bindQueues.
	bindQueuesLock *sync.Mutex
	// List of bindings to re-build on channel re-establishment.
	bindExchanges []*ExchangeBindArgs
	// Lock that must be acquired to alter bindQueues.
	bindExchangesLock *sync.Mutex
}

// Removed a queue from the list of queues to be redeclared on reconnect
func (defaults *routeDeclarationHooks) removeQueue(queueName string) {
	// Remove the queue.
	defaults.declareQueues.Delete(queueName)
	// Remove all binding commands associated with this queue from the re-bind on
	// reconnect list.
	defaults.removeQueueBindings(
		queueName,
		"",
		"",
	)
}

// Remove a re-connection queue binding when a queue, exchange, or binding is removed.
func (defaults *routeDeclarationHooks) removeQueueBindings(
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

	defaults.bindQueuesLock.Lock()
	defer defaults.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range defaults.bindQueues {
		// If there is a routing Key to match, then the queue and exchange must match
		// too (so we don't end up removing a binding with the same routing Key between
		// a different queue-exchange pair).
		if removeRouteMatch &&
			thisBinding.Key == routingKey &&
			thisBinding.Name == queueName &&
			thisBinding.Exchange == exchangeName {
			// then:
			continue
		} else if removeQueueMatch && thisBinding.Name == queueName {
			continue
		} else if removeExchangeMatch && thisBinding.Exchange == exchangeName {
			continue
		}

		defaults.bindQueues[i] = thisBinding
		i++
	}

	defaults.bindQueues = defaults.bindQueues[0:i]
}

// Remove an exchange from the re-declaration list, as well as all queue and
// inter-exchange bindings it was a part of.
func (defaults *routeDeclarationHooks) removeExchange(exchangeName string) {
	// Remove the exchange from the list of exchanges we need to re-declare
	defaults.declareExchanges.Delete(exchangeName)
	// Remove all bindings associated with this exchange from the list of bindings
	// to re-declare on re-connections.
	defaults.removeQueueBindings("", exchangeName, "")
	defaults.removeExchangeBindings(
		"", "", "", exchangeName,
	)
}

// Remove a re-connection binding when a binding or exchange is removed.
func (defaults *routeDeclarationHooks) removeExchangeBindings(
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

	defaults.bindQueuesLock.Lock()
	defer defaults.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range defaults.bindExchanges {
		// If there is a routing Key to match, then the source and destination exchanges
		// must match too (so we don't end up removing a binding with the same routing
		// Key between a different set of exchanges).
		if removeKeyMatch &&
			thisBinding.Key == key &&
			thisBinding.Source == source &&
			thisBinding.Destination == destination {
			// then:
			continue
		} else if removeDestinationMatch && thisBinding.Destination == destination {
			continue
		} else if removeSourceMatch && thisBinding.Source == source {
			continue
		}

		defaults.bindExchanges[i] = thisBinding
		i++
	}

	defaults.bindQueues = defaults.bindQueues[0:i]
}

func (defaults *routeDeclarationHooks) reconnectDeclareQueues(channel *streadway.Channel) error {
	var err error

	redeclareQueues := func(key, value interface{}) bool {
		thisQueue := value.(*QueueDeclareArgs)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := channel.QueueDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnect.

		// TODO: add ability to configure whether or not a passive declare should be
		//   used.
		if true {
			method = channel.QueueDeclare
		}

		_, err = method(
			thisQueue.Name,
			thisQueue.Durable,
			thisQueue.AutoDelete,
			thisQueue.Exclusive,
			// We need to wait and confirm this gets received before moving on
			false,
			thisQueue.Args,
		)
		if err != nil {
			var streadwayErr *Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == NotFound {
				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				defaults.removeQueue(thisQueue.Name)
			}

			err = fmt.Errorf(
				"error re-declaring queue '%v': %w", thisQueue.Name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	defaults.declareQueues.Range(redeclareQueues)

	return err
}

// Re-declares exchanges during reconnection
func (defaults *routeDeclarationHooks) reconnectDeclareExchanges(
	channel *streadway.Channel,
) error {
	var err error

	redeclareExchanges := func(key, value interface{}) bool {
		thisExchange := value.(*ExchangeDeclareArgs)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := channel.ExchangeDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnect.

		// TODO: add ability to configure whether or not a passive declare should be
		//   used.
		if true {
			method = channel.ExchangeDeclare
		}

		err = method(
			thisExchange.Name,
			thisExchange.Kind,
			thisExchange.Durable,
			thisExchange.AutoDelete,
			thisExchange.Internal,
			// we are going to wait so this is done synchronously.
			false,
			thisExchange.Args,
		)
		if err != nil {
			var streadwayErr *Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == NotFound {
				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				defaults.removeExchange(thisExchange.Name)
			}
			err = fmt.Errorf(
				"error re-declaring exchange '%v': %w", thisExchange.Name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	defaults.declareExchanges.Range(redeclareExchanges)

	return err
}

// Re-declares queue bindings during reconnection
func (defaults *routeDeclarationHooks) reconnectBindQueues(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	defaults.bindQueuesLock.Lock()
	defer defaults.bindQueuesLock.Unlock()

	for _, thisBinding := range defaults.bindQueues {
		err := channel.QueueBind(
			thisBinding.Name,
			thisBinding.Key,
			thisBinding.Exchange,
			false,
			thisBinding.Args,
		)

		if err != nil {
			return fmt.Errorf(
				"error re-binding queue '%v' to exchange '%v' with routing Key"+
					" '%v': %w",
				thisBinding.Name,
				thisBinding.Exchange,
				thisBinding.Key,
				err,
			)
		}
	}

	return nil
}

// Re-declares exchange bindings during reconnection
func (defaults *routeDeclarationHooks) reconnectBindExchanges(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	defaults.bindExchangesLock.Lock()
	defer defaults.bindExchangesLock.Unlock()

	for _, thisBinding := range defaults.bindExchanges {
		err := channel.ExchangeBind(
			thisBinding.Destination,
			thisBinding.Key,
			thisBinding.Source,
			false,
			thisBinding.Args,
		)

		if err != nil {
			return fmt.Errorf(
				"error re-binding source exchange '%v' to destination exchange"+
					" '%v' with routing Key '%v': %w",
				thisBinding.Source,
				thisBinding.Destination,
				thisBinding.Key,
				err,
			)
		}
	}

	return nil
}

func (defaults *routeDeclarationHooks) HookReconnect(
	next func() (*streadway.Channel, error), logger zerolog.Logger,
) (*streadway.Channel, error) {
	channel, err := next()
	// If there was an error, pass it up the chain.
	if err != nil {
		return channel, err
	}

	// Recreate queue and exchange topology
	debugEnabled := logger.Debug().Enabled()

	// Declare the queues first.
	if debugEnabled {
		logger.Debug().Msg("re-declaring queues")
	}
	err = defaults.reconnectDeclareQueues(channel)
	if err != nil {
		return channel, err
	}

	// Next declare our exchanges.
	if debugEnabled {
		logger.Debug().Msg("re-declaring exchanges")
	}
	err = defaults.reconnectDeclareExchanges(channel)
	if err != nil {
		return channel, err
	}

	// Now, re-bind our exchanges.
	if debugEnabled {
		logger.Debug().Msg("re-binding exchanges")
	}
	err = defaults.reconnectBindExchanges(channel)
	if err != nil {
		return channel, err
	}

	// Finally, re-bind our queues.
	if debugEnabled {
		logger.Debug().Msg("re-binding queues")
	}
	err = defaults.reconnectBindQueues(channel)
	if err != nil {
		return channel, err
	}

	return channel, nil
}

func (defaults *routeDeclarationHooks) HookQueueDeclare(
	next func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Store the queue name so we can re-declare it
	defaults.declareQueues.Store(args.Name, args)
	return nil
}

func (defaults *routeDeclarationHooks) HookQueueDelete(
	next func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Remove the queue from our list of queue to redeclare.
	defaults.removeQueue(args.Name)
	return nil
}

func (defaults *routeDeclarationHooks) HookQueueBind(
	next func() error, args *QueueBindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	defaults.bindQueuesLock.Lock()
	defer defaults.bindQueuesLock.Unlock()

	defaults.bindQueues = append(defaults.bindQueues, args)
	return nil
}

func (defaults *routeDeclarationHooks) HookQueueUnbind(
	next func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Remove this binding from the list of bindings to re-create on reconnect.
	defaults.removeQueueBindings(
		args.Name,
		args.Exchange,
		args.Key,
	)
	return nil
}
