package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
)

// transportManagerHandlers holds the method handlers and event middleware for
// transportManager methods.
type transportManagerHandlers struct {
	// METHOD HANDLERS
	// ---------------

	// notifyClose is the handler invoked when transportManager.NotifyClose is called.
	notifyClose amqpmiddleware.HandlerNotifyClose

	// notifyDial is the handler invoked when transportManager.NotifyDial is called.
	notifyDial amqpmiddleware.HandlerNotifyDial

	// notifyDisconnect is the handler invoked when transportManager.NotifyDisconnect
	// is called.
	notifyDisconnect amqpmiddleware.HandlerNotifyDisconnect

	// transportClose is the handler invoked when transportManager.Close is called.
	transportClose amqpmiddleware.HandlerClose

	// EVENT Middleware
	// ----------------
	notifyDialEvents       []amqpmiddleware.NotifyDialEvents
	notifyDisconnectEvents []amqpmiddleware.NotifyDisconnectEvents
	notifyCloseEvents      []amqpmiddleware.NotifyCloseEvents
}

// AddNotifyClose adds a new middleware to be invoked on a Connection or Channel
// NotifyClose method call.
func (handlers *transportManagerHandlers) AddNotifyClose(
	middleware amqpmiddleware.NotifyClose,
) {
	handlers.notifyClose = middleware(handlers.notifyClose)
}

// AddNotifyDial adds a new middleware to be invoked on a Connection or Channel
// NotifyClose method call.
func (handlers *transportManagerHandlers) AddNotifyDial(
	middleware amqpmiddleware.NotifyDial,
) {
	handlers.notifyDial = middleware(handlers.notifyDial)
}

// AddNotifyDisconnect adds a new middleware to be invoked on a Connection or Channel
// NotifyDisconnect method call.
func (handlers *transportManagerHandlers) AddNotifyDisconnect(
	middleware amqpmiddleware.NotifyDisconnect,
) {
	handlers.notifyDisconnect = middleware(handlers.notifyDisconnect)
}

// AddClose adds a new middleware to be invoked on a Connection or Channel Close method
// call.
func (handlers *transportManagerHandlers) AddClose(
	middleware amqpmiddleware.Close,
) {
	handlers.transportClose = middleware(handlers.transportClose)
}

// AddNotifyDialEvents adds a new middleware to be invoked on each Connection or Channel
// NotifyDial event.
func (handlers *transportManagerHandlers) AddNotifyDialEvents(
	middleware amqpmiddleware.NotifyDialEvents,
) {
	handlers.notifyDialEvents = append(handlers.notifyDialEvents, middleware)
}

// AddNotifyDisconnectEvents adds a new middleware to be invoked on each Connection or
// Channel NotifyDisconnect event.
func (handlers *transportManagerHandlers) AddNotifyDisconnectEvents(
	middleware amqpmiddleware.NotifyDisconnectEvents,
) {
	handlers.notifyDisconnectEvents = append(
		handlers.notifyDisconnectEvents, middleware,
	)
}

// AddNotifyCloseEvents adds a new middleware to be invoked on each Connection or
// Channel NotifyClose event.
func (handlers *transportManagerHandlers) AddNotifyCloseEvents(
	middleware amqpmiddleware.NotifyCloseEvents,
) {
	handlers.notifyCloseEvents = append(handlers.notifyCloseEvents, middleware)
}

// newTransportManagerHandlers creates the base method handlers for a transportManager
// and returns a transportManagerHandlers with them.
func newTransportManagerHandlers(manager *transportManager) transportManagerHandlers {
	builder := transportHandlersBaseBuilder{
		manager: manager,
	}

	return transportManagerHandlers{
		notifyClose:      builder.createBaseNotifyClose(),
		notifyDial:       builder.createBaseNotifyDial(),
		notifyDisconnect: builder.createBaseNotifyDisconnect(),
		transportClose:   builder.createBaseClose(),

		notifyDialEvents:       nil,
		notifyDisconnectEvents: nil,
		notifyCloseEvents:      nil,
	}
}
