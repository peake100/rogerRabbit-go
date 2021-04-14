package middleware

// SetupChannel is a middleware signature for wrapping roger.Consumer.SetupChannel
type SetupChannel = func(next HandlerSetupChannel) HandlerSetupChannel

// Delivery is a middleware signature for wrapping roger.Consumer.HandleDelivery.
type Delivery = func(next HandlerDelivery) HandlerDelivery

// CleanupChannel is a middleware signature for wrapping roger.Consumer.CleanupChannel.
type CleanupChannel = func(next HandlerCleanupChannel) HandlerCleanupChannel
