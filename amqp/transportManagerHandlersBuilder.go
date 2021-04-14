package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// transportManagerMiddleware holds all the middleware configured for a specific
// transportManager.
type transportManagerMiddleware struct {
	// METHOD HANDLERS
	// ---------------

	// middleware for transportManager.NotifyClose
	notifyClose []amqpmiddleware.NotifyClose
	// middleware for transportManager.NotifyDial
	notifyDial []amqpmiddleware.NotifyDial
	// middleware for transportManager.NotifyDisconnect
	notifyDisconnect []amqpmiddleware.NotifyDisconnect
	// middleware for transportManager.Close
	transportClose []amqpmiddleware.Close

	// EVENT HANDLERS
	// ---------------

	// middleware for transportManager.NotifyDial events
	notifyDialEvents []amqpmiddleware.NotifyDialEvents
	// middleware for transportManager.NotifyDisconnect events
	notifyDisconnectEvents []amqpmiddleware.NotifyDisconnectEvents
	// middleware for transportManager.NotifyClose events
	notifyCloseEvents []amqpmiddleware.NotifyCloseEvents
}

//revive:disable:line-length-limit
// we need to disable this here since the builder signatures are so long.

// transportHandlersBuilder builds the base handlers for transportManager methods.
type transportHandlersBuilder struct {
	manager    *transportManager
	middleware transportManagerMiddleware
}

// createNotifyClose creates the base handler for transportManager.NotifyClose.
func (builder transportHandlersBuilder) createNotifyClose() amqpmiddleware.HandlerNotifyClose {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyCloseEvents

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyClose) (results amqpmiddleware.ResultsNotifyClose) {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		results.CallerChan = args.Receiver

		// If the context of the livesOnce manager has been cancelled, close the
		// receiver and exit.
		if manager.ctx.Err() != nil {
			close(args.Receiver)
			return results
		}

		manager.notificationSubscriberClose = append(
			manager.notificationSubscriberClose, args.Receiver,
		)

		eventHandler := func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyClose) {
			// We send the error then close the channel. This handler will only be
			// called once on the final livesOnce close.
			args.Receiver <- event.Err
			close(args.Receiver)
		}

		for _, middleware := range eventMiddlewares {
			eventHandler = middleware(eventHandler)
		}

		manager.notifyCloseEventHandlers = append(
			manager.notifyCloseEventHandlers, eventHandler,
		)

		return results
	}

	for _, middleware := range builder.middleware.notifyClose {
		handler = middleware(handler)
	}

	return handler
}

// createNotifyDial creates the base handler for transportManager.NotifyDial.
func (builder transportHandlersBuilder) createNotifyDial() amqpmiddleware.HandlerNotifyDial {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyDialEvents

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyDial) error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the livesOnce manager has been cancelled, close the
		// receiver and exit.
		if manager.ctx.Err() != nil {
			close(args.Receiver)
			return streadway.ErrClosed
		}

		manager.notificationSubscribersDial = append(
			manager.notificationSubscribersDial, args.Receiver,
		)

		// Add the event handler to the list of event handlers.
		eventHandler := func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyDial) {
			args.Receiver <- event.Err
		}

		for _, thisMiddleware := range eventMiddlewares {
			eventHandler = thisMiddleware(eventHandler)
		}

		manager.notifyDialEventHandlers = append(
			manager.notifyDialEventHandlers, eventHandler,
		)

		return nil
	}

	for _, middleware := range builder.middleware.notifyDial {
		handler = middleware(handler)
	}

	return handler
}

// createNotifyDisconnect creates the base handler for
// transportManager.NotifyDisconnect.
func (builder transportHandlersBuilder) createNotifyDisconnect() amqpmiddleware.HandlerNotifyDisconnect {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyDisconnectEvents

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyDisconnect) error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the livesOnce manager has been cancelled, close the
		// receiver and exit.
		if manager.ctx.Err() != nil {
			close(args.Receiver)
			return streadway.ErrClosed
		}

		manager.notificationSubscriberDisconnect = append(
			manager.notificationSubscriberDisconnect, args.Receiver,
		)

		// Set up the event handler for this receiver.
		eventHandler := func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyDisconnect) {
			args.Receiver <- event.Err
		}

		for _, middleware := range eventMiddlewares {
			eventHandler = middleware(eventHandler)
		}

		manager.notifyDisconnectEventHandlers = append(
			manager.notifyDisconnectEventHandlers, eventHandler,
		)

		return nil
	}

	for _, middleware := range builder.middleware.notifyDisconnect {
		handler = middleware(handler)
	}

	return handler
}

// createClose creates the base handler for transportManager.Close.
func (builder transportHandlersBuilder) createClose() amqpmiddleware.HandlerClose {
	manager := builder.manager

	handler := func(ctx context.Context, args amqpmiddleware.ArgsClose) error {

		// If the context has already been cancelled, we can exit.
		if manager.ctx.Err() != nil {
			return streadway.ErrClosed
		}

		manager.cancelCtxCloseTransport()

		func() {
			// We need to acquire and release the lock here to avoid race conditions.
			manager.notificationSubscriberLock.Lock()
			defer manager.notificationSubscriberLock.Unlock()

			// Close all disconnect and connect subscribers, then clear them. We don't
			// need to grab the lock for this since the cancelled context will keep any new
			// subscribers from being added.
			for _, subscriber := range manager.notificationSubscribersDial {
				close(subscriber)
			}
			// We clear these to avoid sending on a nil channel.
			manager.notificationSubscribersDial = nil
			manager.notifyDialEventHandlers = nil

			for _, subscriber := range manager.notificationSubscriberDisconnect {
				close(subscriber)
			}
			// We clear these to avoid sending on a nil channel.
			manager.notificationSubscriberDisconnect = nil
			manager.notifyDisconnectEventHandlers = nil
		}()

		// Send closure notifications to all subscribers.
		manager.sendCloseNotifications(nil)

		// Execute any cleanup on behalf of the livesOnce implementation.
		return manager.transport.cleanup()
	}

	for _, middleware := range builder.middleware.transportClose {
		handler = middleware(handler)
	}

	return handler
}

//revive:enable:line-length-limit
