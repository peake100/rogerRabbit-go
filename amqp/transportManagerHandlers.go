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
}

// newTransportManagerHandlers creates the base method handlers for a transportManager
// and returns a transportManagerHandlers with them.
func newTransportManagerHandlers(
	manager *transportManager,
	middleware transportManagerMiddleware,
) transportManagerHandlers {
	builder := transportHandlersBuilder{
		manager:    manager,
		middleware: middleware,
	}

	return transportManagerHandlers{
		notifyClose:      builder.createBaseNotifyClose(),
		notifyDial:       builder.createBaseNotifyDial(),
		notifyDisconnect: builder.createBaseNotifyDisconnect(),
		transportClose:   builder.createBaseClose(),
	}
}
