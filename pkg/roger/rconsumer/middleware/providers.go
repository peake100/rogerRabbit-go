package middleware

// ProviderTypeID identifies a ProvidesMiddleware type for catching duplicate providers.
type ProviderTypeID string

// ProvidesMiddleware is the base interface that must be implemented by any middleware
// provider.
type ProvidesMiddleware interface {
	// TypeID returns a unique ID for verifying that a provider has not been registered
	// more than once.
	TypeID() ProviderTypeID
}

// ProvidesSetupChannel provides SetupChannel middleware as a method.
type ProvidesSetupChannel interface {
	ProvidesMiddleware
	SetupChannel(next HandlerSetupChannel) HandlerSetupChannel
}

// ProvidesDelivery provides Delivery middleware as a method.
type ProvidesDelivery interface {
	ProvidesMiddleware
	Delivery(next HandlerDelivery) HandlerDelivery
}

// ProvidesCleanupChannel provides CleanupChannel middleware as a method.
type ProvidesCleanupChannel interface {
	ProvidesMiddleware
	CleanupChannel(next HandlerCleanupChannel) HandlerCleanupChannel
}

// ProvidesAll provides all consumer middlewares.
type ProvidesAll interface {
	ProvidesSetupChannel
	ProvidesDelivery
	ProvidesCleanupChannel
}
