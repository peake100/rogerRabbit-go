package amqp

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
)

// HOOK DEFINITIONS

// Signature for hooks triggered when a channel is being re-established.
type HandlerReconnect = func(
	next func() (*streadway.Channel, error), logger zerolog.Logger,
) (*streadway.Channel, error)

type HandlerQueueDeclare = func(
	next func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error

type MiddlewareQueueDeclare = func(handler HandlerQueueDeclare) HandlerQueueDeclare

type HandlerQueueDelete = func(
	next func() error, args *QueueDeleteArgs, logger zerolog.Logger,
) error

type HandlerQueueBind = func(
	next func() error, args *QueueBindArgs, logger zerolog.Logger,
) error

type HandlerQueueUnbind = func(
	next func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error

type HandlerExchangeDeclare func(
	next func() error, args *ExchangeDeclareArgs, logger zerolog.Logger,
) error

type HandlerExchangeDelete func(
	next func() error, args *ExchangeDeleteArgs, logger zerolog.Logger,
) error

type HandlerExchangeBind func(
	next func() error, args *ExchangeBindArgs, logger zerolog.Logger,
) error

type HandlerExchangeUnbind func(
	next func() error, args *ExchangeUnbindArgs, logger zerolog.Logger,
) error

type channelHandlers struct {
	reconnect       []HandlerReconnect
	queueDeclare    HandlerQueueDeclare
	queueDelete     []HandlerQueueDelete
	queueBind       []HandlerQueueBind
	queueUnbind     []HandlerQueueUnbind
	exchangeDeclare []HandlerExchangeDeclare
	exchangeDelete  []HandlerExchangeDelete
	exchangeBind    []HandlerExchangeBind
	exchangeUnbind  []HandlerExchangeUnbind

	lock *sync.RWMutex
}

func (handlers *channelHandlers) RegisterReconnect(hook HandlerReconnect) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reconnect = append(handlers.reconnect, hook)
}

func (handlers *channelHandlers) runHooksReconnect(
	reconnect func() (*streadway.Channel, error),
	logger zerolog.Logger,
) (*streadway.Channel, error) {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() (*streadway.Channel, error)

	for _, thisHook := range handlers.reconnect {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = reconnect
		}

		reconnect = func() (*streadway.Channel, error) {
			return thisHook(innerMethod, logger)
		}
		lastMethod = reconnect
	}

	return reconnect()
}

func (handlers *channelHandlers) RegisterQueueDeclare(
	middleware MiddlewareQueueDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDeclare = middleware(handlers.queueDeclare)
}

func (handlers *channelHandlers) runHooksQueueDeclare(
	runMethodOnce func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.queueDeclare {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterQueueDelete(hook HandlerQueueDelete) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDelete = append(handlers.queueDelete, hook)
}

func (handlers *channelHandlers) runHooksQueueDelete(
	runMethodOnce func() error, args *QueueDeleteArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.queueDelete {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterQueueBind(hook HandlerQueueBind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueBind = append(handlers.queueBind, hook)
}

func (handlers *channelHandlers) runHooksQueueBind(
	runMethodOnce func() error, args *QueueBindArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.queueBind {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterQueueUnbind(hook HandlerQueueUnbind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueUnbind = append(handlers.queueUnbind, hook)
}

func (handlers *channelHandlers) runHooksQueueUnbind(
	runMethodOnce func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.queueUnbind {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterExchangeDeclare(hook HandlerExchangeDeclare) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclare = append(handlers.exchangeDeclare, hook)
}

func (handlers *channelHandlers) runHooksExchangeDeclare(
	runMethodOnce func() error, args *ExchangeDeclareArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.exchangeDeclare {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterExchangeDelete(hook HandlerExchangeDelete) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDelete = append(handlers.exchangeDelete, hook)
}

func (handlers *channelHandlers) runHooksExchangeDelete(
	runMethodOnce func() error, args *ExchangeDeleteArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.exchangeDelete {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterExchangeBind(hook HandlerExchangeBind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeBind = append(handlers.exchangeBind, hook)
}

func (handlers *channelHandlers) runHooksExchangeBind(
	runMethodOnce func() error, args *ExchangeBindArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.exchangeBind {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

func (handlers *channelHandlers) RegisterExchangeUnbind(hook HandlerExchangeUnbind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeUnbind = append(handlers.exchangeUnbind, hook)
}

func (handlers *channelHandlers) runHooksExchangeUnbind(
	runMethodOnce func() error, args *ExchangeUnbindArgs, logger zerolog.Logger,
) error {
	handlers.lock.RLock()
	defer handlers.lock.RUnlock()

	var lastMethod func() error

	for _, thisHook := range handlers.exchangeUnbind {
		// We have to declare a new variable here, or all inner methods will recursively
		// call whatever the function pointer is set to outside the loop
		innerMethod := lastMethod
		if innerMethod == nil {
			innerMethod = runMethodOnce
		}

		runMethodOnce = func() error {
			return thisHook(innerMethod, args, logger)
		}
		lastMethod = runMethodOnce
	}

	return runMethodOnce()
}

// This object implements hooks for re-declaring queues, exchanges, and bindings upon
// reconnect.
type routeDeclarationMiddleware struct {
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
func (middle *routeDeclarationMiddleware) removeQueue(queueName string) {
	// Remove the queue.
	middle.declareQueues.Delete(queueName)
	// Remove all binding commands associated with this queue from the re-bind on
	// reconnect list.
	middle.removeQueueBindings(
		queueName,
		"",
		"",
	)
}

// Remove a re-connection queue binding when a queue, exchange, or binding is removed.
func (middle *routeDeclarationMiddleware) removeQueueBindings(
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

	middle.bindQueuesLock.Lock()
	defer middle.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range middle.bindQueues {
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

		middle.bindQueues[i] = thisBinding
		i++
	}

	middle.bindQueues = middle.bindQueues[0:i]
}

// Remove an exchange from the re-declaration list, as well as all queue and
// inter-exchange bindings it was a part of.
func (middle *routeDeclarationMiddleware) removeExchange(exchangeName string) {
	// Remove the exchange from the list of exchanges we need to re-declare
	middle.declareExchanges.Delete(exchangeName)
	// Remove all bindings associated with this exchange from the list of bindings
	// to re-declare on re-connections.
	middle.removeQueueBindings("", exchangeName, "")
	middle.removeExchangeBindings(
		"", "", "", exchangeName,
	)
}

// Remove a re-connection binding when a binding or exchange is removed.
func (middle *routeDeclarationMiddleware) removeExchangeBindings(
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

	middle.bindQueuesLock.Lock()
	defer middle.bindQueuesLock.Unlock()

	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range middle.bindExchanges {
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

		middle.bindExchanges[i] = thisBinding
		i++
	}

	middle.bindQueues = middle.bindQueues[0:i]
}

func (middle *routeDeclarationMiddleware) reconnectDeclareQueues(channel *streadway.Channel) error {
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
				middle.removeQueue(thisQueue.Name)
			}

			err = fmt.Errorf(
				"error re-declaring queue '%v': %w", thisQueue.Name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	middle.declareQueues.Range(redeclareQueues)

	return err
}

// Re-declares exchanges during reconnection
func (middle *routeDeclarationMiddleware) reconnectDeclareExchanges(
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
				middle.removeExchange(thisExchange.Name)
			}
			err = fmt.Errorf(
				"error re-declaring exchange '%v': %w", thisExchange.Name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	middle.declareExchanges.Range(redeclareExchanges)

	return err
}

// Re-declares queue bindings during reconnection
func (middle *routeDeclarationMiddleware) reconnectBindQueues(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	middle.bindQueuesLock.Lock()
	defer middle.bindQueuesLock.Unlock()

	for _, thisBinding := range middle.bindQueues {
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
func (middle *routeDeclarationMiddleware) reconnectBindExchanges(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	middle.bindExchangesLock.Lock()
	defer middle.bindExchangesLock.Unlock()

	for _, thisBinding := range middle.bindExchanges {
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

func (middle *routeDeclarationMiddleware) HookReconnect(
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
	err = middle.reconnectDeclareQueues(channel)
	if err != nil {
		return channel, err
	}

	// Next declare our exchanges.
	if debugEnabled {
		logger.Debug().Msg("re-declaring exchanges")
	}
	err = middle.reconnectDeclareExchanges(channel)
	if err != nil {
		return channel, err
	}

	// Now, re-bind our exchanges.
	if debugEnabled {
		logger.Debug().Msg("re-binding exchanges")
	}
	err = middle.reconnectBindExchanges(channel)
	if err != nil {
		return channel, err
	}

	// Finally, re-bind our queues.
	if debugEnabled {
		logger.Debug().Msg("re-binding queues")
	}
	err = middle.reconnectBindQueues(channel)
	if err != nil {
		return channel, err
	}

	return channel, nil
}

func (middle *routeDeclarationMiddleware) HookQueueDeclare(
	next func() error, args *QueueDeclareArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Store the queue name so we can re-declare it
	middle.declareQueues.Store(args.Name, args)
	return nil
}

func (middle *routeDeclarationMiddleware) HookQueueDelete(
	next func() error, args *QueueDeleteArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Remove the queue from our list of queue to redeclare.
	middle.removeQueue(args.Name)
	return nil
}

func (middle *routeDeclarationMiddleware) HookQueueBind(
	next func() error, args *QueueBindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	middle.bindQueuesLock.Lock()
	defer middle.bindQueuesLock.Unlock()

	middle.bindQueues = append(middle.bindQueues, args)
	return nil
}

func (middle *routeDeclarationMiddleware) HookQueueUnbind(
	next func() error, args *QueueUnbindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Remove this binding from the list of bindings to re-create on reconnect.
	middle.removeQueueBindings(
		args.Name,
		args.Exchange,
		args.Key,
	)
	return nil
}

func (middle *routeDeclarationMiddleware) HookExchangeDeclare(
	next func() error, args *ExchangeDeclareArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	middle.declareExchanges.Store(args.Name, args)
	return nil
}

func (middle *routeDeclarationMiddleware) HookExchangeDelete(
	next func() error, args *ExchangeDeleteArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Remove the exchange from our re-declare on reconnect lists.
	middle.removeExchange(args.Name)

	return nil
}

func (middle *routeDeclarationMiddleware) HookExchangeBind(
	next func() error, args *ExchangeBindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	// Store this binding so we can re-bind it if we lose and regain the connection.
	middle.bindExchangesLock.Lock()
	defer middle.bindExchangesLock.Unlock()

	middle.bindExchanges = append(middle.bindExchanges, args)

	return nil
}

func (middle *routeDeclarationMiddleware) HookExchangeUnbind(
	next func() error, args *ExchangeUnbindArgs, logger zerolog.Logger,
) error {
	// If there is any sort of error, pass it on.
	err := next()
	if err != nil {
		return err
	}

	middle.removeExchangeBindings(
		args.Destination, args.Key, args.Source, "",
	)

	return nil
}
