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

	// providerFactory
	providerFactory []struct {
		Factory func() interface{}
	}
}

// AddProviderFactory adds a factory function which creates a new middleware provider
// value which must implement one of the Middleware Provider interfaces from the
// amqpmiddleware package, like amqpmiddleware.ProvidesClose.
//
// When middleware is registered on a new Connection, the provider factory will be
// called and all provider methods will be registered as middleware.
//
// If you wish the same provider value's methods to be used as middleware for every
// *Connection created by a Config, consider using AddProviderMethods instead.
func (config *ConnectionMiddleware) AddProviderFactory(
	factory func() interface{},
) {
	config.providerFactory = append(config.providerFactory, struct {
		Factory func() interface{}
	}{Factory: factory})
}

// AddClose adds a new middleware to be invoked on a Connection.Close call.
func (config *ConnectionMiddleware) AddClose(
	middleware amqpmiddleware.Close,
) {
	config.transportClose = append(config.transportClose, middleware)
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

// AddNotifyClose adds a new middleware to be invoked when a connection attempts to
// re-establish a connection.
func (config *ConnectionMiddleware) AddConnectionReconnect(
	middleware amqpmiddleware.ConnectionReconnect,
) {
	config.connectionReconnect = append(config.connectionReconnect, middleware)
}

// buildAndAddProviderMethods builds invokes all provider factories passed to
// AddProviderFactory and adds their methods as middleware. A copy of the config should
// be made before this call is made, and then discarded once connection handlers are
// created.
func (config *ConnectionMiddleware) buildAndAddProviderMethods() {
	for _, registered := range config.providerFactory {
		config.AddProviderMethods(registered.Factory())
	}
}

// AddProviderMethods adds a Middleware Provider's methods as Middleware. If this method
// is invoked directly by the user, the same type value's method will be added to all
// *Connection values created by a Config
//
// If a new provider value should be made for each Connection, consider using
// AddProviderFactory instead.
func (config *ConnectionMiddleware) AddProviderMethods(provider interface{}) {
	if hasMethods, ok := provider.(amqpmiddleware.ProvidesClose); ok {
		config.AddClose(hasMethods.Close)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyClose); ok {
		config.AddNotifyClose(hasMethods.NotifyClose)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyDial); ok {
		config.AddNotifyDial(hasMethods.NotifyDial)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyDisconnect); ok {
		config.AddNotifyDisconnect(hasMethods.NotifyDisconnect)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyCloseEvents); ok {
		config.AddNotifyCloseEvents(hasMethods.NotifyCloseEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyDialEvents); ok {
		config.AddNotifyDialEvents(hasMethods.NotifyDialEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyDisconnectEvents); ok {
		config.AddNotifyDisconnectEvents(hasMethods.NotifyDisconnectEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesConnectionReconnect); ok {
		config.AddConnectionReconnect(hasMethods.ConnectionReconnect)
	}
}
