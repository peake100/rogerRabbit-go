package amqp

import (
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

// transportHandlersBuilder builds the base handlers for transportManager methods.
type transportHandlersBuilder struct {
	manager    *transportManager
	middleware transportManagerMiddleware
}

// createBaseNotifyClose creates the base handler for transportManager.NotifyClose.
func (
	builder transportHandlersBuilder,
) createBaseNotifyClose() amqpmiddleware.HandlerNotifyClose {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyCloseEvents

	return func(args amqpmiddleware.ArgsNotifyClose) chan *streadway.Error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the livesOnce manager has been cancelled, close the
		// receiver and exit.
		if manager.ctx.Err() != nil {
			close(args.Receiver)
			return args.Receiver
		}

		manager.notificationSubscriberClose = append(
			manager.notificationSubscriberClose, args.Receiver,
		)

		var eventHandler amqpmiddleware.HandlerNotifyCloseEvents = func(
			event amqpmiddleware.EventNotifyClose,
		) {
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

		return args.Receiver
	}
}

// createBaseNotifyDial creates the base handler for transportManager.NotifyDial.
func (
	builder transportHandlersBuilder,
) createBaseNotifyDial() amqpmiddleware.HandlerNotifyDial {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyDialEvents

	return func(args amqpmiddleware.ArgsNotifyDial) error {
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
		var eventHandler amqpmiddleware.HandlerNotifyDialEvents = func(
			event amqpmiddleware.EventNotifyDial,
		) {
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
}

// createBaseNotifyDisconnect creates the base handler for
// transportManager.NotifyDisconnect.
func (
	builder transportHandlersBuilder,
) createBaseNotifyDisconnect() amqpmiddleware.HandlerNotifyDisconnect {
	manager := builder.manager
	eventMiddlewares := builder.middleware.notifyDisconnectEvents

	return func(args amqpmiddleware.ArgsNotifyDisconnect) error {
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
		var eventHandler amqpmiddleware.HandlerNotifyDisconnectEvents = func(
			event amqpmiddleware.EventNotifyDisconnect,
		) {
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
}

// createBaseClose creates the base handler for transportManager.Close.
func (builder transportHandlersBuilder) createBaseClose() amqpmiddleware.HandlerClose {
	manager := builder.manager
	return func(args amqpmiddleware.ArgsClose) error {
		// If the context has already been cancelled, we can exit.
		if manager.ctx.Err() != nil {
			return streadway.ErrClosed
		}

		manager.cancelCtxCloseTransport()

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

		// Send closure notifications to all subscribers.
		manager.sendCloseNotifications(nil)

		// Execute any cleanup on behalf of the livesOnce implementation.
		return manager.transport.cleanup()
	}
}
