package defaultmiddlewares

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
	"sync"
)

// RouteDeclarationMiddlewareID can be used to retrieve the running instance of
// RouteDeclarationMiddleware during testing.
const RouteDeclarationMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultRouteDeclaration"

// RouteDeclarationMiddleware implements handlers for re-declaring queues, exchanges,
// and bindings upon reconnectMiddleware.
type RouteDeclarationMiddleware struct {
	// declareQueues is a map queues that must be declared upon re-establishing the
	// channel. We use a map so we can remove queues from this list on queue delete.
	declareQueues *sync.Map
	// declareExchanges us a map of exchanges that must be declared upon re-establishing
	// the channel. We use a map so we can remove exchanges from this list on exchange
	// delete.
	declareExchanges *sync.Map
	// bindQueues is a list of bindings to re-build on channel re-establishment.
	bindQueues []amqpmiddleware.ArgsQueueBind
	// bindQueuesLock must be acquired to alter bindQueues.
	bindQueuesLock *sync.Mutex
	// bindExchanges is a list of bindings to re-build on channel re-establishment.
	bindExchanges []amqpmiddleware.ArgsExchangeBind
	// bindExchangesLock must be acquired to alter bindQueues.
	bindExchangesLock *sync.Mutex
}

// TypeID implements amqpmiddleware.ProvidesMiddleware and returns a static type ID for
// retrieving the active middleware value during testing.
func (middleware *RouteDeclarationMiddleware) TypeID() amqpmiddleware.ProviderTypeID {
	return RouteDeclarationMiddlewareID
}

// removeQueue Removes a queue from the list of queues to be redeclared on
// reconnectMiddleware.
func (middleware *RouteDeclarationMiddleware) removeQueue(queueName string) {
	// Remove the queue.
	middleware.declareQueues.Delete(queueName)
	// Remove all binding commands associated with this queue from the re-bind on
	// reconnectMiddleware list.
	middleware.removeQueueBindings(
		queueName,
		"",
		"",
	)
}

// removeQueueBindingOpts holds information about a queue to be removed.
type removeQueueBindingOpts struct {
	// queueName is the name to remove bindings for.
	queueName string
	// exchangeName is an exchange name to remove bindings for
	exchangeName string
	// routingKey is the key on the exchange to remove bindings for
	routingKey string

	// removeQueueMatch: when true, allows the removal of a  binding if queue name
	// matches
	removeQueueMatch bool
	// removeExchangeMatch: when true, allows the removal of a  binding if exchange
	// name matches
	removeExchangeMatch bool
	// removeExchangeMatch: when true, allows the removal of a binding if route name
	// matches
	removeRouteMatch bool
}

// removeQueueBindingOk compares the original args a queue bind was made with and
// a removeQueueBindingOpts to see if a queue should be removed.
func removeQueueBindingOk(
	binding amqpmiddleware.ArgsQueueBind, opts removeQueueBindingOpts,
) bool {
	// If there is a routing Key to match, then the queue and exchange must match
	// too (so we don't end up removing a binding with the same routing Key between
	// a different queue-exchange pair).
	if opts.removeRouteMatch &&
		binding.Key == opts.routingKey &&
		binding.Name == opts.queueName &&
		binding.Exchange == opts.exchangeName {
		// then:
		return true
	}

	if opts.removeQueueMatch && binding.Name == opts.queueName {
		return true
	}

	if opts.removeExchangeMatch && binding.Exchange == opts.exchangeName {
		return true
	}

	return false
}

// removeQueueBindingsFromSlice iterates over all queue bindings, and removes any
// relevant to the detailed removeQueueBindingOpts.
func (middleware *RouteDeclarationMiddleware) removeQueueBindingsFromSlice(
	opts removeQueueBindingOpts,
) {
	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range middleware.bindQueues {
		if removeQueueBindingOk(thisBinding, opts) {
			continue
		}

		middleware.bindQueues[i] = thisBinding
		i++
	}

	middleware.bindQueues = middleware.bindQueues[0:i]
}

// removeQueueBindings removes a re-connection queue binding when a queue, exchange, or
// binding is removed.
func (middleware *RouteDeclarationMiddleware) removeQueueBindings(
	queueName string,
	exchangeName string,
	routingKey string,
) {
	middleware.bindQueuesLock.Lock()
	defer middleware.bindQueuesLock.Unlock()

	middleware.removeQueueBindingsFromSlice(
		removeQueueBindingOpts{
			queueName:    queueName,
			exchangeName: exchangeName,
			routingKey:   routingKey,

			// If the queue name is non-emtpy, remove on queue match (will remove all
			// bindings for the queue).
			removeQueueMatch: queueName != "",
			// If the exchange name is non-empty, remove on exchange match (will remove
			//all bindings for the exchange).
			removeExchangeMatch: exchangeName != "",
			// If the routeingKey is non-nil, only remove bindings where the routing
			// key, queue, and exchange all match (that exact binding).
			removeRouteMatch: routingKey != "",
		},
	)
}

// removeExchange removes an exchange from the re-declaration list, as well as all queue
// and inter-exchange bindings it was a part of.
func (middleware *RouteDeclarationMiddleware) removeExchange(exchangeName string) {
	// Remove the exchange from the list of exchanges we need to re-declare
	middleware.declareExchanges.Delete(exchangeName)
	// Remove all bindings associated with this exchange from the list of bindings
	// to re-declare on re-connections.
	middleware.removeQueueBindings("", exchangeName, "")
	middleware.removeExchangeBindings(
		exchangeName, "", exchangeName, true,
	)
}

// removeExchangeBindingOpts details the information for removing exchange bindings.
type removeExchangeBindingOpts struct {
	// destination exchange requested for removal.
	destination string
	// key: routing key requested for removal.
	key string
	// source exchange requested for removal.
	source string
	// When a queue is deleted we need to remove any binding where the source or
	// destination matches
	destinationOrSource bool

	// removeDestinationMatch: when true, remove any bindings were destination matches
	// the binding.
	removeDestinationMatch bool
	// removeDestinationMatch: when true, remove any bindings were key matches
	// the binding.
	removeKeyMatch bool
	// removeDestinationMatch: when true, remove any bindings were source matches
	// the binding.
	removeSourceMatch bool
}

// removeExchangeBindingOk returns true is binding made with
// amqpmiddleware.ArgsExchangeBind should be removed based on removeExchangeBindingOpts.
func removeExchangeBindingOk(
	binding amqpmiddleware.ArgsExchangeBind, opts removeExchangeBindingOpts,
) bool {
	// If there is a routing Key to match, then the source and destination exchanges
	// must match too (so we don't end up removing a binding with the same routing
	// Key between a different isSet of exchanges).
	if opts.removeKeyMatch &&
		binding.Key == opts.key &&
		binding.Source == opts.source &&
		binding.Destination == opts.destination {
		// then:
		return true
	}

	if opts.removeDestinationMatch && binding.Destination == opts.destination {
		return true
	}

	if opts.removeSourceMatch && binding.Source == opts.source {
		return true
	}

	return false
}

// removeExchangeBindingsFromSlice removes all exchange bindings relevant to
// removeExchangeBindingOpts.
func (middleware *RouteDeclarationMiddleware) removeExchangeBindingsFromSlice(
	opts removeExchangeBindingOpts,
) {
	// Rather than creating a new slice, we are going to filter out any matching
	// bind declarations we find, then constrain the slice to the number if items
	// we have left.
	i := 0
	for _, thisBinding := range middleware.bindExchanges {
		// If we are to remove the binding, continue.
		if removeExchangeBindingOk(thisBinding, opts) {
			continue
		}

		middleware.bindExchanges[i] = thisBinding
		i++
	}

	middleware.bindQueues = middleware.bindQueues[0:i]
}

// removeExchangeBindings removes a re-connection binding when a binding or exchange is
// removed.
func (middleware *RouteDeclarationMiddleware) removeExchangeBindings(
	destination string,
	key string,
	source string,
	// When a queue is deleted we need to remove any binding where the source or
	// destination matches
	destinationOrSource bool,
) {
	middleware.bindQueuesLock.Lock()
	defer middleware.bindQueuesLock.Unlock()

	middleware.removeExchangeBindingsFromSlice(
		removeExchangeBindingOpts{
			destination:            destination,
			key:                    key,
			source:                 source,
			destinationOrSource:    destinationOrSource,
			removeDestinationMatch: destination != "" || destinationOrSource,
			removeKeyMatch:         key != "",
			removeSourceMatch:      source != "" || destinationOrSource,
		},
	)
}

// reconnectDeclareQueues re-declares all previously declared queues on a amqp.Channel
// reconnection.
func (middleware *RouteDeclarationMiddleware) reconnectDeclareQueues(
	channel *streadway.Channel,
) error {
	var err error

	redeclareQueues := func(key, value interface{}) bool {
		thisQueue := value.(amqpmiddleware.ArgsQueueDeclare)

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
			if errors.As(err, &streadwayErr) &&
				streadwayErr.Code == streadway.NotFound {
				// THEN:

				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				middleware.removeQueue(thisQueue.Name)
			}

			err = fmt.Errorf(
				"error re-declaring queue '%v': %w", thisQueue.Name, err,
			)
			return false
		}

		return true
	}
	// Redeclare all queues in the map.
	middleware.declareQueues.Range(redeclareQueues)

	return err
}

// reconnectDeclareExchanges re-declares all exchanges previously declared on am
// amqp.Channel during reconnection.
func (middleware *RouteDeclarationMiddleware) reconnectDeclareExchanges(
	channel *streadway.Channel,
) error {
	var err error

	redeclareExchanges := func(key, value interface{}) bool {
		thisExchange := value.(amqpmiddleware.ArgsExchangeDeclare)

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
			if errors.As(err, &streadwayErr) &&
				streadwayErr.Code == streadway.NotFound {
				// If this is a passive declare, we can get a 404 NOT_FOUND error. If we
				// do, then we should remove this queue from the list of queues that is
				// to be re-declared, so that we don't get caught in an endless loop
				// of reconnects.
				middleware.removeExchange(thisExchange.Name)
			}
			err = fmt.Errorf(
				"error re-declaring exchange '%v': %w", thisExchange.Name, err,
			)
			return false
		}

		return true
	}

	// Redeclare all queues in the map.
	middleware.declareExchanges.Range(redeclareExchanges)

	return err
}

// reconnectBindQueues re-declares queue bindings previously made on an amqp.Channel
// during reconnection.
func (middleware *RouteDeclarationMiddleware) reconnectBindQueues(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	middleware.bindQueuesLock.Lock()
	defer middleware.bindQueuesLock.Unlock()

	for _, thisBinding := range middleware.bindQueues {
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

// reconnectBindExchanges re-declares all exchange bindings previously made on an
// amqp.Channel during reconnection.
func (middleware *RouteDeclarationMiddleware) reconnectBindExchanges(
	channel *streadway.Channel,
) error {
	// We shouldn't meed to lock this resource here, since this method will only be
	// used when we have a write lock on the transport, and all methods that modify the
	// binding list must first acquire the same lock for read, but we will put this here
	// in case that changes in the future.
	middleware.bindExchangesLock.Lock()
	defer middleware.bindExchangesLock.Unlock()

	for _, thisBinding := range middleware.bindExchanges {
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

// reconnectHandler re-establishes queue and exchange topologies on a channel
// reconnection event.
func (middleware *RouteDeclarationMiddleware) reconnectHandler(
	ctx context.Context,
	args amqpmiddleware.ArgsChannelReconnect,
	next amqpmiddleware.HandlerChannelReconnect,
) (*streadway.Channel, error) {
	channel, err := next(ctx, args)
	// If there was an error, pass it up the chain.
	if err != nil {
		return channel, err
	}

	err = middleware.reconnectDeclareQueues(channel)
	if err != nil {
		return channel, err
	}

	err = middleware.reconnectDeclareExchanges(channel)
	if err != nil {
		return channel, err
	}

	err = middleware.reconnectBindExchanges(channel)
	if err != nil {
		return channel, err
	}

	err = middleware.reconnectBindQueues(channel)
	if err != nil {
		return channel, err
	}

	return channel, nil
}

// Reconnect is invoked on reconnection of the underlying amqp Channel, and makes sure
// our queue and exchange topology is re-configured to present a seamless experience
// to the caller.
func (middleware *RouteDeclarationMiddleware) ChannelReconnect(
	next amqpmiddleware.HandlerChannelReconnect,
) (handler amqpmiddleware.HandlerChannelReconnect) {
	handler = func(ctx context.Context, args amqpmiddleware.ArgsChannelReconnect) (*streadway.Channel, error) {
		return middleware.reconnectHandler(ctx, args, next)
	}

	return handler
}

// QueueDeclare captures amqp.Channel.QueueDeclare() calls and stores their arguments
// for re-declaring channels on disconnect.
func (middleware *RouteDeclarationMiddleware) QueueDeclare(
	next amqpmiddleware.HandlerQueueDeclare,
) (handler amqpmiddleware.HandlerQueueDeclare) {
	handler = func(ctx context.Context, args amqpmiddleware.ArgsQueueDeclare) (streadway.Queue, error) {
		// If there is any sort of error, pass it on.
		queue, err := next(ctx, args)
		if err != nil {
			return queue, err
		}

		// Store the queue name so we can re-declare it
		middleware.declareQueues.Store(args.Name, args)
		return queue, err
	}

	return handler
}

// QueueDelete captures amqp.Channel.QueueDelete() calls and removes all relevant
// saved queues and bindings so they are not re-declared on a channel reconnect.
func (middleware *RouteDeclarationMiddleware) QueueDelete(
	next amqpmiddleware.HandlerQueueDelete,
) (handler amqpmiddleware.HandlerQueueDelete) {
	// If there is any sort of error, pass it on.
	handler = func(ctx context.Context, args amqpmiddleware.ArgsQueueDelete) (count int, err error) {
		count, err = next(ctx, args)
		if err != nil {
			return count, err
		}

		// Remove the queue from our list of queue to redeclare.
		middleware.removeQueue(args.Name)

		return count, err
	}

	return handler
}

// QueueBind captures amqp.Channel.QueueBind() saves queue bind arguments, so they can
// be re-declared on channel reconnection.
func (middleware *RouteDeclarationMiddleware) QueueBind(
	next amqpmiddleware.HandlerQueueBind,
) (handler amqpmiddleware.HandlerQueueBind) {
	// If there is any sort of error, pass it on.
	handler = func(ctx context.Context, args amqpmiddleware.ArgsQueueBind) error {
		err := next(ctx, args)
		if err != nil {
			return err
		}

		middleware.bindQueuesLock.Lock()
		defer middleware.bindQueuesLock.Unlock()

		middleware.bindQueues = append(middleware.bindQueues, args)
		return nil
	}

	return handler
}

// QueueUnbind captures amqp.Channel.QueueUnbind() calls and removes all relevant saved
// bindings so they are not re-declared on a channel reconnect.
func (middleware *RouteDeclarationMiddleware) QueueUnbind(
	next amqpmiddleware.HandlerQueueUnbind,
) (handler amqpmiddleware.HandlerQueueUnbind) {
	handler = func(ctx context.Context, args amqpmiddleware.ArgsQueueUnbind) error {
		// If there is any sort of error, pass it on.
		err := next(ctx, args)
		if err != nil {
			return err
		}

		// Remove this binding from the list of bindings to re-create on reconnect.
		middleware.removeQueueBindings(
			args.Name,
			args.Exchange,
			args.Key,
		)

		return nil
	}

	return handler
}

// ExchangeDeclare captures amqp.Channel.ExchangeDeclare() saves passed arguments so
// exchanges can be re-declared on channel reconnection.
func (middleware *RouteDeclarationMiddleware) ExchangeDeclare(
	next amqpmiddleware.HandlerExchangeDeclare,
) (handler amqpmiddleware.HandlerExchangeDeclare) {
	handler = func(ctx context.Context, args amqpmiddleware.ArgsExchangeDeclare) error {
		// If there is any sort of error, pass it on.
		err := next(ctx, args)
		if err != nil {
			return err
		}

		middleware.declareExchanges.Store(args.Name, args)
		return nil
	}

	return handler
}

// ExchangeDelete captures amqp.Channel.ExchangeDelete() calls and removes all relevant
// saved exchanges so they are not re-declared on a channel reconnect.
func (middleware *RouteDeclarationMiddleware) ExchangeDelete(
	next amqpmiddleware.HandlerExchangeDelete,
) (handler amqpmiddleware.HandlerExchangeDelete) {
	handler = func(ctx context.Context, args amqpmiddleware.ArgsExchangeDelete) error {
		// If there is any sort of error, pass it on.
		err := next(ctx, args)
		if err != nil {
			return err
		}

		// Remove the exchange from our re-declare on reconnect lists.
		middleware.removeExchange(args.Name)

		return nil
	}

	return handler
}

// ExchangeBind captures amqp.Channel.ExchangeBind() saves passed arguments, so exchange
// bindings can be re-declared on channel reconnection.
func (middleware *RouteDeclarationMiddleware) ExchangeBind(
	next amqpmiddleware.HandlerExchangeBind,
) (handler amqpmiddleware.HandlerExchangeBind) {
	// If there is any sort of error, pass it on.
	handler = func(ctx context.Context, args amqpmiddleware.ArgsExchangeBind) error {
		err := next(ctx, args)
		if err != nil {
			return err
		}

		// Store this binding so we can re-bind it if we lose and regain the connection.
		middleware.bindExchangesLock.Lock()
		defer middleware.bindExchangesLock.Unlock()

		middleware.bindExchanges = append(middleware.bindExchanges, args)

		return nil
	}

	return handler
}

// ExchangeUnbind captures amqp.Channel.ExchangeUnbind() calls and removes all relevant
// saved bindings so they are not re-declared on a channel reconnect.
func (middleware *RouteDeclarationMiddleware) ExchangeUnbind(
	next amqpmiddleware.HandlerExchangeUnbind,
) (handler amqpmiddleware.HandlerExchangeUnbind) {
	// If there is any sort of error, pass it on.
	handler = func(ctx context.Context, args amqpmiddleware.ArgsExchangeUnbind) error {
		err := next(ctx, args)
		if err != nil {
			return err
		}

		middleware.removeExchangeBindings(
			args.Destination, args.Key, args.Source, false,
		)

		return nil
	}

	return handler
}

// NewRouteDeclarationMiddleware creates a new RouteDeclarationMiddleware.
func NewRouteDeclarationMiddleware() amqpmiddleware.ProvidesMiddleware {
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
