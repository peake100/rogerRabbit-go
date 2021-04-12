package amqp

import "github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"

// ConnectionMiddleware holds the middleware to add to Connection methods
// and events.
type ConnectionMiddleware struct {
	// TRANSPORT METHOD HANDLERS
	// -------------------------

	// notifyClose is the handler invoked when transportManager.NotifyClose is called.
	notifyClose []amqpmiddleware.NotifyClose
	// notifyDial is the handler invoked when transportManager.NotifyDial is called.
	notifyDial []amqpmiddleware.NotifyDial
	// notifyDisconnect is the handler invoked when transportManager.NotifyDisconnect
	// is called.
	notifyDisconnect []amqpmiddleware.NotifyDisconnect
	// transportClose is the handler invoked when transportManager.Close is called.
	transportClose []amqpmiddleware.Close

	// TRANSPORT EVENT MIDDLEWARE
	// --------------------------
	notifyDialEvents       []amqpmiddleware.NotifyDialEvents
	notifyDisconnectEvents []amqpmiddleware.NotifyDisconnectEvents
	notifyCloseEvents      []amqpmiddleware.NotifyCloseEvents

	// CONNECTION MIDDLEWARE
	// ---------------------
	connectionReconnect []amqpmiddleware.ConnectionReconnect
}

// AddNotifyClose adds a new middleware to be invoked when a connection attempts to
// re-establish a connection.
func (config *ConnectionMiddleware) AddConnectionReconnect(
	middleware amqpmiddleware.ConnectionReconnect,
) {
	config.connectionReconnect = append(config.connectionReconnect, middleware)
}

// AddNotifyClose adds a new middleware to be invoked on a Connection.NotifyClose call.
func (config *ConnectionMiddleware) AddNotifyClose(
	middleware amqpmiddleware.NotifyClose,
) {
	config.notifyClose = append(config.notifyClose, middleware)
}

// AddNotifyDial adds a new middleware to be invoked on a Connection.NotifyDial call.
func (config *ConnectionMiddleware) AddNotifyDial(
	middleware amqpmiddleware.NotifyDial,
) {
	config.notifyDial = append(config.notifyDial, middleware)
}

// AddNotifyDisconnect adds a new middleware to be invoked on a
// Connection.NotifyDisconnect call.
func (config *ConnectionMiddleware) AddNotifyDisconnect(
	middleware amqpmiddleware.NotifyDisconnect,
) {
	config.notifyDisconnect = append(config.notifyDisconnect, middleware)
}

// AddClose adds a new middleware to be invoked on a Connection.Close call.
func (config *ConnectionMiddleware) AddClose(
	middleware amqpmiddleware.Close,
) {
	config.transportClose = append(config.transportClose, middleware)
}

// AddNotifyDialEvents adds a new middleware to be invoked on each
// Connection.NotifyDial event.
func (config *ConnectionMiddleware) AddNotifyDialEvents(
	middleware amqpmiddleware.NotifyDialEvents,
) {
	config.notifyDialEvents = append(config.notifyDialEvents, middleware)
}

// AddNotifyDialEvents adds a new middleware to be invoked on each
// Connection.NotifyDial event.
func (config *ConnectionMiddleware) AddNotifyDisconnectEvents(
	middleware amqpmiddleware.NotifyDisconnectEvents,
) {
	config.notifyDisconnectEvents = append(config.notifyDisconnectEvents, middleware)
}

// AddNotifyDialEvents adds a new middleware to be invoked on each
// Connection.NotifyDial event.
func (config *ConnectionMiddleware) AddNotifyCloseEvents(
	middleware amqpmiddleware.NotifyCloseEvents,
) {
	config.notifyCloseEvents = append(config.notifyCloseEvents, middleware)
}
