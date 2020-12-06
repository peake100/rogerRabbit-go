package defaultMiddlewares

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
)

// This object implements handlers for re-declaring queues, exchanges, and bindings upon
// reconnectMiddleware.
type RouteDeclarationMiddleware struct {
	// List of queues that must be declared upon re-establishing the channel. We use a
	// map so we can remove queues from this list on queue delete.
	declareQueues *sync.Map
	// List of exchanges that must be declared upon re-establishing the channel. We use
	// a map so we can remove exchanges from this list on exchange delete.
	declareExchanges *sync.Map
	// List of bindings to re-build on channel re-establishment.
	bindQueues []*amqpMiddleware.ArgsQueueBind
	// Lock that must be acquired to alter bindQueues.
	bindQueuesLock *sync.Mutex
	// List of bindings to re-build on channel re-establishment.
	bindExchanges []*amqpMiddleware.ArgsExchangeBind
	// Lock that must be acquired to alter bindQueues.
	bindExchangesLock *sync.Mutex
}

// Removed a queue from the list of queues to be redeclared on reconnectMiddleware
func (middle *RouteDeclarationMiddleware) removeQueue(queueName string) {
	// Remove the queue.
	middle.declareQueues.Delete(queueName)
	// Remove all binding commands associated with this queue from the re-bind on
	// reconnectMiddleware list.
	middle.removeQueueBindings(
		queueName,
		"",
		"",
	)
}

func (middle *RouteDeclarationMiddleware) removeQueueBindingsFromSlice(
	queueName, exchangeName, routingKey string,
	removeQueueMatch, removeExchangeMatch, removeRouteMatch bool,
) {
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

// Remove a re-connection queue binding when a queue, exchange, or binding is removed.
func (middle *RouteDeclarationMiddleware) removeQueueBindings(
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

	middle.removeQueueBindingsFromSlice(
		queueName,
		exchangeName,
		routingKey,
		removeQueueMatch,
		removeExchangeMatch,
		removeRouteMatch,
	)
}

// Remove an exchange from the re-declaration list, as well as all queue and
// inter-exchange bindings it was a part of.
func (middle *RouteDeclarationMiddleware) removeExchange(exchangeName string) {
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
func (middle *RouteDeclarationMiddleware) removeExchangeBindings(
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
		// Key between a different isSet of exchanges).
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

func (middle *RouteDeclarationMiddleware) reconnectDeclareQueues(
	channel *streadway.Channel,
) error {
	var err error

	redeclareQueues := func(key, value interface{}) bool {
		thisQueue := value.(*amqpMiddleware.ArgsQueueDeclare)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := channel.QueueDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnectMiddleware.

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
			var streadwayErr *streadway.Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == streadway.NotFound {
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
func (middle *RouteDeclarationMiddleware) reconnectDeclareExchanges(
	channel *streadway.Channel,
) error {
	var err error

	redeclareExchanges := func(key, value interface{}) bool {
		thisExchange := value.(*amqpMiddleware.ArgsExchangeDeclare)

		// By default, we will passively declare a queue. This allows us to respect
		// queue deletion by other producers or consumers.
		method := channel.ExchangeDeclarePassive
		// UNLESS it is an auto-delete queue. Such a queue may have been cleaned up
		// by the broker and should be fully re-declared on reconnectMiddleware.

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
			var streadwayErr *streadway.Error
			if errors.As(err, &streadwayErr) && streadwayErr.Code == streadway.NotFound {
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
func (middle *RouteDeclarationMiddleware) reconnectBindQueues(
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
func (middle *RouteDeclarationMiddleware) reconnectBindExchanges(
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

func (middle *RouteDeclarationMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	handler = func(
		ctx context.Context, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := next(ctx, logger)

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

	return handler
}

func (middle *RouteDeclarationMiddleware) QueueDeclare(
	next amqpMiddleware.HandlerQueueDeclare,
) (handler amqpMiddleware.HandlerQueueDeclare) {
	handler = func(args *amqpMiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
		// If there is any sort of error, pass it on.
		queue, err := next(args)
		if err != nil {
			return queue, err
		}

		// Store the queue name so we can re-declare it
		middle.declareQueues.Store(args.Name, args)
		return queue, err
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) QueueDelete(
	next amqpMiddleware.HandlerQueueDelete,
) (handler amqpMiddleware.HandlerQueueDelete) {
	// If there is any sort of error, pass it on.
	handler = func(args *amqpMiddleware.ArgsQueueDelete) (count int, err error) {
		count, err = next(args)
		if err != nil {
			return count, err
		}

		// Remove the queue from our list of queue to redeclare.
		middle.removeQueue(args.Name)

		return count, err
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) QueueBind(
	next amqpMiddleware.HandlerQueueBind,
) (handler amqpMiddleware.HandlerQueueBind) {
	// If there is any sort of error, pass it on.
	handler = func(args *amqpMiddleware.ArgsQueueBind) error {
		err := next(args)
		if err != nil {
			return err
		}

		middle.bindQueuesLock.Lock()
		defer middle.bindQueuesLock.Unlock()

		middle.bindQueues = append(middle.bindQueues, args)
		return nil
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) QueueUnbind(
	next amqpMiddleware.HandlerQueueUnbind,
) (handler amqpMiddleware.HandlerQueueUnbind) {
	handler = func(args *amqpMiddleware.ArgsQueueUnbind) error {
		// If there is any sort of error, pass it on.
		err := next(args)
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

	return handler
}

func (middle *RouteDeclarationMiddleware) ExchangeDeclare(
	next amqpMiddleware.HandlerExchangeDeclare,
) (handler amqpMiddleware.HandlerExchangeDeclare) {
	handler = func(args *amqpMiddleware.ArgsExchangeDeclare) error {
		// If there is any sort of error, pass it on.
		err := next(args)
		if err != nil {
			return err
		}

		middle.declareExchanges.Store(args.Name, args)
		return nil
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) ExchangeDelete(
	next amqpMiddleware.HandlerExchangeDelete,
) (handler amqpMiddleware.HandlerExchangeDelete) {
	handler = func(args *amqpMiddleware.ArgsExchangeDelete) error {
		// If there is any sort of error, pass it on.
		err := next(args)
		if err != nil {
			return err
		}

		// Remove the exchange from our re-declare on reconnect lists.
		middle.removeExchange(args.Name)

		return nil
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) ExchangeBind(
	next amqpMiddleware.HandlerExchangeBind,
) (handler amqpMiddleware.HandlerExchangeBind) {
	// If there is any sort of error, pass it on.
	handler = func(args *amqpMiddleware.ArgsExchangeBind) error {
		err := next(args)
		if err != nil {
			return err
		}

		// Store this binding so we can re-bind it if we lose and regain the connection.
		middle.bindExchangesLock.Lock()
		defer middle.bindExchangesLock.Unlock()

		middle.bindExchanges = append(middle.bindExchanges, args)

		return nil
	}

	return handler
}

func (middle *RouteDeclarationMiddleware) ExchangeUnbind(
	next amqpMiddleware.HandlerExchangeUnbind,
) (handler amqpMiddleware.HandlerExchangeUnbind) {
	// If there is any sort of error, pass it on.
	handler = func(args *amqpMiddleware.ArgsExchangeUnbind) error {
		err := next(args)
		if err != nil {
			return err
		}

		middle.removeExchangeBindings(
			args.Destination, args.Key, args.Source, "",
		)

		return nil
	}

	return handler
}

func NewRouteDeclarationMiddleware() *RouteDeclarationMiddleware {
	middleware := &RouteDeclarationMiddleware{
		declareQueues:     new(sync.Map),
		declareExchanges:  new(sync.Map),
		bindQueues:        nil,
		bindQueuesLock:    new(sync.Mutex),
		bindExchanges:     nil,
		bindExchangesLock: new(sync.Mutex),
	}

	return middleware
}
