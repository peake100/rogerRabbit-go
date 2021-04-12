package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// transportHandlersBaseBuilder builds the base handlers for transportManager methods.
type transportHandlersBaseBuilder struct {
	manager *transportManager
}

// createBaseNotifyClose creates the base handler for transportManager.NotifyClose.
func (
	builder transportHandlersBaseBuilder,
) createBaseNotifyClose() amqpmiddleware.HandlerNotifyClose {
	manager := builder.manager
	return func(args amqpmiddleware.ArgsNotifyClose) chan *streadway.Error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the transport manager has been cancelled, close the
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
			// called once on the final transport close.
			args.Receiver <- event.Err
			close(args.Receiver)
		}

		for _, middleware := range manager.handlers.notifyCloseEvents {
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
	builder transportHandlersBaseBuilder,
) createBaseNotifyDial() amqpmiddleware.HandlerNotifyDial {
	manager := builder.manager
	return func(args amqpmiddleware.ArgsNotifyDial) error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the transport manager has been cancelled, close the
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

		for _, thisMiddleware := range manager.handlers.notifyDialEvents {
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
	builder transportHandlersBaseBuilder,
) createBaseNotifyDisconnect() amqpmiddleware.HandlerNotifyDisconnect {
	manager := builder.manager
	return func(args amqpmiddleware.ArgsNotifyDisconnect) error {
		manager.notificationSubscriberLock.Lock()
		defer manager.notificationSubscriberLock.Unlock()

		// If the context of the transport manager has been cancelled, close the
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

		for _, middleware := range builder.manager.handlers.notifyDisconnectEvents {
			eventHandler = middleware(eventHandler)
		}

		manager.notifyDisconnectEventHandlers = append(
			manager.notifyDisconnectEventHandlers, eventHandler,
		)

		return nil
	}
}

// createBaseClose creates the base handler for transportManager.Close.
func (
	builder transportHandlersBaseBuilder,
) createBaseClose() amqpmiddleware.HandlerClose {
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

		// Execute any cleanup on behalf of the transport implementation.
		return manager.transport.cleanup()
	}
}
