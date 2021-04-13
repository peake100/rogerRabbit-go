package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
)

type ChannelMiddlewares struct {
	// TRANSPORT METHOD HANDLERS
	// -------------------------

	// transportClose is the handler invoked when transportManager.Close is called.
	transportClose []amqpmiddleware.Close
	// notifyClose is the handler invoked when transportManager.NotifyClose is called.
	notifyClose []amqpmiddleware.NotifyClose
	// notifyDial is the handler invoked when transportManager.NotifyDial is called.
	notifyDial []amqpmiddleware.NotifyDial
	// notifyDisconnect is the handler invoked when transportManager.NotifyDisconnect
	// is called.
	notifyDisconnect []amqpmiddleware.NotifyDisconnect

	// TRANSPORT EVENT MIDDLEWARE
	// --------------------------
	notifyCloseEvents      []amqpmiddleware.NotifyCloseEvents
	notifyDialEvents       []amqpmiddleware.NotifyDialEvents
	notifyDisconnectEvents []amqpmiddleware.NotifyDisconnectEvents

	// CHANNEL MIDDLEWARE
	// ------------------

	// LIFETIME MIDDLEWARE
	// -------------------

	// channelReconnect is the handler invoked on a reconnection event.
	channelReconnect []amqpmiddleware.ChannelReconnect

	// MODE MIDDLEWARE
	// ---------------

	// qos is the handler for Channel.Qos
	qos []amqpmiddleware.QoS
	// flow is the handler for Channel.Flow
	flow []amqpmiddleware.Flow
	// confirm is the handler for Channel.Confirm
	confirm []amqpmiddleware.Confirm

	// QUEUE MIDDLEWARE
	// ----------------

	// queueDeclare is the handler for Channel.QueueDeclare
	queueDeclare []amqpmiddleware.QueueDeclare
	// queueDeclarePassive is the handler for Channel.QueueDeclare
	queueDeclarePassive []amqpmiddleware.QueueDeclare
	// queueInspect is the handler for Channel.QueueInspect
	queueInspect []amqpmiddleware.QueueInspect
	// queueDelete is the handler for Channel.QueueDelete
	queueDelete []amqpmiddleware.QueueDelete
	// queueBind is the handler for Channel.QueueBind
	queueBind []amqpmiddleware.QueueBind
	// queueUnbind is the handler for Channel.QueueUnbind
	queueUnbind []amqpmiddleware.QueueUnbind
	// queueInspect is the handler for Channel.QueueInspect
	queuePurge []amqpmiddleware.QueuePurge

	// EXCHANGE MIDDLEWARE
	// -------------------

	// exchangeDeclare is the handler for Channel.ExchangeDeclare
	exchangeDeclare []amqpmiddleware.ExchangeDeclare
	// exchangeDeclarePassive is the handler for Channel.ExchangeDeclare
	exchangeDeclarePassive []amqpmiddleware.ExchangeDeclare
	// exchangeDelete is the handler for Channel.ExchangeDelete
	exchangeDelete []amqpmiddleware.ExchangeDelete
	// exchangeBind is the handler for Channel.ExchangeBind
	exchangeBind []amqpmiddleware.ExchangeBind
	// exchangeUnbind is the handler for Channel.ExchangeUnbind
	exchangeUnbind []amqpmiddleware.ExchangeUnbind

	// NOTIFY MIDDLEWARE
	// -----------------

	// notifyPublish is the handler for Channel.NotifyPublish
	notifyPublish []amqpmiddleware.NotifyPublish
	// consume is the handler for Channel.Consume
	consume []amqpmiddleware.Consume
	// notifyConfirm is the handler for Channel.NotifyConfirm
	notifyConfirm []amqpmiddleware.NotifyConfirm
	// notifyConfirmOrOrphaned is the handler for Channel.NotifyConfirmOrOrphaned
	notifyConfirmOrOrphaned []amqpmiddleware.NotifyConfirmOrOrphaned
	// notifyReturn is the handler for Channel.NotifyReturn
	notifyReturn []amqpmiddleware.NotifyReturn
	// notifyCancel is the handler for Channel.NotifyCancel
	notifyCancel []amqpmiddleware.NotifyCancel
	// notifyFlow is the handler for Channel.NotifyFlow
	notifyFlow []amqpmiddleware.NotifyFlow

	// MESSAGING MIDDLEWARE
	// --------------------

	// publish is the handler for Channel.Publish
	publish []amqpmiddleware.Publish
	// get is the handler for Channel.Get
	get []amqpmiddleware.Get

	// ACK MIDDLEWARE
	// --------------

	// ack is the handler for Channel.Ack
	ack []amqpmiddleware.Ack
	// nack is the handler for Channel.Nack
	nack []amqpmiddleware.Nack
	// reject is the handler for Channel.Reject
	reject []amqpmiddleware.Reject

	// EVENT MIDDLEWARE
	// ----------------

	// notifyPublishEvents is middleware to be registered on a
	// notifyPublishRelay.
	notifyPublishEvents []amqpmiddleware.NotifyPublishEvents
	// consumeEvents is middleware to be registered on a
	// consumeRelay.
	consumeEvents []amqpmiddleware.ConsumeEvents
	// notifyConfirmEvents is a middleware to be registered on events for
	// Channel.NotifyConfirm.
	notifyConfirmEvents []amqpmiddleware.NotifyConfirmEvents
	// notifyConfirmOrOrphanedEvents is a middleware to be registered on events for
	// Channel.NotifyConfirmOrOrphaned.
	notifyConfirmOrOrphanedEvents []amqpmiddleware.NotifyConfirmOrOrphanedEvents
	// notifyReturnEvents is a middleware to be registered on events for
	// Channel.NotifyReturn.
	notifyReturnEvents []amqpmiddleware.NotifyReturnEvents
	// notifyCancelEvents is a middleware to be registered on events for
	// Channel.NotifyCancel.
	notifyCancelEvents []amqpmiddleware.NotifyCancelEvents
	// notifyFlowEvents is a middleware to be registered on events for
	// Channel.NotifyFlow.
	notifyFlowEvents []amqpmiddleware.NotifyFlowEvents

	// providerFactory
	providerFactory []struct {
		Factory func() interface{}
	}
}

// AddProviderFactory adds a factory function which creates a new middleware provider
// value which must implement one of the Middleware Provider interfaces from the
// amqpmiddleware package, like amqpmiddleware.ProvidesQueueDeclare.
//
// When middleware is registered on a new Channel, the provider factory will be called
// and all provider methods will be registered as middleware.
//
// If you wish the same provider value's methods to be used as middleware for every
// *Channel created by a *Connection, consider using AddProviderMethods instead.
func (config *ChannelMiddlewares) AddProviderFactory(
	factory func() interface{},
) {
	config.providerFactory = append(config.providerFactory, struct {
		Factory func() interface{}
	}{Factory: factory})
}

// AddClose adds a new middleware to be invoked on Channel.Close method calls.
func (config *ChannelMiddlewares) AddClose(middleware amqpmiddleware.Close) {
	config.transportClose = append(config.transportClose, middleware)
}

// AddNotifyClose adds a new middleware to be invoked on Channel.NotifyClose method
// calls.
func (config *ChannelMiddlewares) AddNotifyClose(middleware amqpmiddleware.NotifyClose) {
	config.notifyClose = append(config.notifyClose, middleware)
}

// AddNotifyDial adds a new middleware to be invoked on Channel.NotifyDial method calls.
func (config *ChannelMiddlewares) AddNotifyDial(middleware amqpmiddleware.NotifyDial) {
	config.notifyDial = append(config.notifyDial, middleware)
}

// AddNotifyDisconnect adds a new middleware to be invoked on Channel.NotifyDisconnect
// method calls.
func (config *ChannelMiddlewares) AddNotifyDisconnect(middleware amqpmiddleware.NotifyDisconnect) {
	config.notifyDisconnect = append(config.notifyDisconnect, middleware)
}

// AddNotifyCloseEvents adds a new middleware to be invoked on all events sent to
// callers of Channel.NotifyClose.
func (config *ChannelMiddlewares) AddNotifyCloseEvents(middleware amqpmiddleware.NotifyCloseEvents) {
	config.notifyCloseEvents = append(config.notifyCloseEvents, middleware)
}

// AddNotifyDialEvents adds a new middleware to be invoked on all events sent to
// callers of Channel.NotifyDial.
func (config *ChannelMiddlewares) AddNotifyDialEvents(middleware amqpmiddleware.NotifyDialEvents) {
	config.notifyDialEvents = append(config.notifyDialEvents, middleware)
}

// AddNotifyDisconnectEvents adds a new middleware to be invoked on all events sent to
// callers of Channel.NotifyDial.
func (config *ChannelMiddlewares) AddNotifyDisconnectEvents(middleware amqpmiddleware.NotifyDisconnectEvents) {
	config.notifyDisconnectEvents = append(config.notifyDisconnectEvents, middleware)
}

// AddChannelReconnect adds a new middleware to be invoked on a Channel reconnection
// event.
func (config *ChannelMiddlewares) AddChannelReconnect(middleware amqpmiddleware.ChannelReconnect) {
	config.channelReconnect = append(config.channelReconnect, middleware)
}

// AddQoS adds a new middleware to be invoked on Channel.Qos method calls.
func (config *ChannelMiddlewares) AddQoS(middleware amqpmiddleware.QoS) {
	config.qos = append(config.qos, middleware)
}

// AddFlow adds a new middleware to be invoked on Channel.Flow method calls.
func (config *ChannelMiddlewares) AddFlow(middleware amqpmiddleware.Flow) {
	config.flow = append(config.flow, middleware)
}

// AddConfirm adds a new middleware to be invoked on Channel.Confirm method calls.
func (config *ChannelMiddlewares) AddConfirm(middleware amqpmiddleware.Confirm) {
	config.confirm = append(config.confirm, middleware)
}

// AddQueueDeclare adds a new middleware to be invoked on Channel.QueueDeclare method
// calls.
func (config *ChannelMiddlewares) AddQueueDeclare(
	middleware amqpmiddleware.QueueDeclare,
) {
	config.queueDeclare = append(config.queueDeclare, middleware)
}

// AddQueueDeclarePassive adds a new middleware to be invoked on
// Channel.QueueDeclarePassive method calls.
func (config *ChannelMiddlewares) AddQueueDeclarePassive(
	middleware amqpmiddleware.QueueDeclare,
) {
	config.queueDeclarePassive = append(config.queueDeclarePassive, middleware)
}

// AddQueueInspect adds a new middleware to be invoked on Channel.QueueInspect method
// calls.
func (config *ChannelMiddlewares) AddQueueInspect(
	middleware amqpmiddleware.QueueInspect,
) {
	config.queueInspect = append(config.queueInspect, middleware)
}

// AddQueueDelete adds a new middleware to be invoked on Channel.QueueDelete method
// calls.
func (config *ChannelMiddlewares) AddQueueDelete(middleware amqpmiddleware.QueueDelete) {
	config.queueDelete = append(config.queueDelete, middleware)
}

// AddQueueBind adds a new middleware to be invoked on Channel.QueueBind method
// calls.
func (config *ChannelMiddlewares) AddQueueBind(middleware amqpmiddleware.QueueBind) {
	config.queueBind = append(config.queueBind, middleware)
}

// AddQueueUnbind adds a new middleware to be invoked on Channel.QueueUnbind method
// calls.
func (config *ChannelMiddlewares) AddQueueUnbind(middleware amqpmiddleware.QueueUnbind) {
	config.queueUnbind = append(config.queueUnbind, middleware)
}

// AddQueuePurge adds a new middleware to be invoked on Channel.QueuePurge method
// calls.
func (config *ChannelMiddlewares) AddQueuePurge(
	middleware amqpmiddleware.QueuePurge,
) {
	config.queuePurge = append(config.queuePurge, middleware)
}

// AddExchangeDeclare adds a new middleware to be invoked on Channel.ExchangeDeclare
// method calls.
func (config *ChannelMiddlewares) AddExchangeDeclare(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	config.exchangeDeclare = append(config.exchangeDeclare, middleware)
}

// AddExchangeDeclarePassive adds a new middleware to be invoked on
// Channel.ExchangeDeclarePassive method calls.
func (config *ChannelMiddlewares) AddExchangeDeclarePassive(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	config.exchangeDeclarePassive = append(config.exchangeDeclarePassive, middleware)
}

// AddExchangeDelete adds a new middleware to be invoked on Channel.ExchangeDelete
// method calls.
func (config *ChannelMiddlewares) AddExchangeDelete(
	middleware amqpmiddleware.ExchangeDelete,
) {
	config.exchangeDelete = append(config.exchangeDelete, middleware)
}

// AddExchangeBind adds a new middleware to be invoked on Channel.ExchangeBind
// method calls.
func (config *ChannelMiddlewares) AddExchangeBind(
	middleware amqpmiddleware.ExchangeBind,
) {
	config.exchangeBind = append(config.exchangeBind, middleware)
}

// AddExchangeUnbind adds a new middleware to be invoked on Channel.ExchangeUnbind
// method calls.
func (config *ChannelMiddlewares) AddExchangeUnbind(
	middleware amqpmiddleware.ExchangeUnbind,
) {
	config.exchangeUnbind = append(config.exchangeUnbind, middleware)
}

// AddPublish adds a new middleware to be invoked on Channel.Publish method calls.
func (config *ChannelMiddlewares) AddPublish(
	middleware amqpmiddleware.Publish,
) {
	config.publish = append(config.publish, middleware)
}

// AddGet adds a new middleware to be invoked on Channel.Get method calls.
func (config *ChannelMiddlewares) AddGet(
	middleware amqpmiddleware.Get,
) {
	config.get = append(config.get, middleware)
}

// AddConsume adds a new middleware to be invoked on Channel.Consume method calls.
//
// NOTE: this is a distinct middleware from AddConsumeEvents, which fires on every
// delivery sent from the broker. This event only fires once when the  Channel.Consume
// method is first called.
func (config *ChannelMiddlewares) AddConsume(
	middleware amqpmiddleware.Consume,
) {
	config.consume = append(config.consume, middleware)
}

// AddAck adds a new middleware to be invoked on Channel.Ack method calls.
func (config *ChannelMiddlewares) AddAck(
	middleware amqpmiddleware.Ack,
) {
	config.ack = append(config.ack, middleware)
}

// AddNack adds a new middleware to be invoked on Channel.Nack method calls.
func (config *ChannelMiddlewares) AddNack(
	middleware amqpmiddleware.Nack,
) {
	config.nack = append(config.nack, middleware)
}

// AddReject adds a new middleware to be invoked on Channel.Reject method calls.
func (config *ChannelMiddlewares) AddReject(
	middleware amqpmiddleware.Reject,
) {
	config.reject = append(config.reject, middleware)
}

// AddNotifyPublish adds a new middleware to be invoked on Channel.NotifyPublish method
// calls.
func (config *ChannelMiddlewares) AddNotifyPublish(
	middleware amqpmiddleware.NotifyPublish,
) {
	config.notifyPublish = append(config.notifyPublish, middleware)
}

// AddNotifyConfirm adds a new middleware to be invoked on Channel.NotifyConfirm method
// calls.
func (config *ChannelMiddlewares) AddNotifyConfirm(
	middleware amqpmiddleware.NotifyConfirm,
) {
	config.notifyConfirm = append(config.notifyConfirm, middleware)
}

// AddNotifyConfirmOrOrphaned adds a new middleware to be invoked on
// Channel.NotifyConfirmOrOrphaned method calls.
func (config *ChannelMiddlewares) AddNotifyConfirmOrOrphaned(
	middleware amqpmiddleware.NotifyConfirmOrOrphaned,
) {
	config.notifyConfirmOrOrphaned = append(config.notifyConfirmOrOrphaned, middleware)
}

// AddNotifyReturn adds a new middleware to be invoked on Channel.NotifyReturn method
// calls.
func (config *ChannelMiddlewares) AddNotifyReturn(
	middleware amqpmiddleware.NotifyReturn,
) {
	config.notifyReturn = append(config.notifyReturn, middleware)
}

// AddNotifyCancel adds a new middleware to be invoked on Channel.NotifyCancel method
// calls.
func (config *ChannelMiddlewares) AddNotifyCancel(
	middleware amqpmiddleware.NotifyCancel,
) {
	config.notifyCancel = append(config.notifyCancel, middleware)
}

// AddNotifyFlow adds a new middleware to be invoked on Channel.NotifyFlow method
// calls.
func (config *ChannelMiddlewares) AddNotifyFlow(
	middleware amqpmiddleware.NotifyFlow,
) {
	config.notifyFlow = append(config.notifyFlow, middleware)
}

// AddNotifyPublishEvents adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyPublish.
func (config *ChannelMiddlewares) AddNotifyPublishEvents(
	middleware amqpmiddleware.NotifyPublishEvents,
) {
	config.notifyPublishEvents = append(config.notifyPublishEvents, middleware)
}

// AddConsumeEvents adds a new middleware to be invoked on events sent to callers
// of Channel.Consume.
func (config *ChannelMiddlewares) AddConsumeEvents(
	middleware amqpmiddleware.ConsumeEvents,
) {
	config.consumeEvents = append(config.consumeEvents, middleware)
}

// AddNotifyConfirmEvents adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyConfirm.
func (config *ChannelMiddlewares) AddNotifyConfirmEvents(
	middleware amqpmiddleware.NotifyConfirmEvents,
) {
	config.notifyConfirmEvents = append(
		config.notifyConfirmEvents, middleware,
	)
}

// AddNotifyConfirmOrOrphanedEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyConfirmOrOrphaned.
func (config *ChannelMiddlewares) AddNotifyConfirmOrOrphanedEvents(
	middleware amqpmiddleware.NotifyConfirmOrOrphanedEvents,
) {
	config.notifyConfirmOrOrphanedEvents = append(
		config.notifyConfirmOrOrphanedEvents, middleware,
	)
}

// AddNotifyReturnEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyReturn.
func (config *ChannelMiddlewares) AddNotifyReturnEvents(
	middleware amqpmiddleware.NotifyReturnEvents,
) {
	config.notifyReturnEvents = append(
		config.notifyReturnEvents, middleware,
	)
}

// AddNotifyCancelEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (config *ChannelMiddlewares) AddNotifyCancelEvents(
	middleware amqpmiddleware.NotifyCancelEvents,
) {
	config.notifyCancelEvents = append(
		config.notifyCancelEvents, middleware,
	)
}

// AddNotifyFlowEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (config *ChannelMiddlewares) AddNotifyFlowEvents(
	middleware amqpmiddleware.NotifyFlowEvents,
) {
	config.notifyFlowEvents = append(
		config.notifyFlowEvents, middleware,
	)
}

// buildAndAddProviderMethods builds invokes all provider factories passed to
// AddProviderFactory and adds their methods as middleware. A copy of the config should
// be made before this call is made, and then discarded once channel handlers are
// created.
func (config *ChannelMiddlewares) buildAndAddProviderMethods() {
	for _, registered := range config.providerFactory {
		config.AddProviderMethods(registered.Factory())
	}
}

// The sheer number of type asserts in the method below breaks the cyclomatic and
// cognitive complexity checks implemented by revive, so we will disable them for this
// method.

//revive:disable:cognitive-complexity
//revive:disable:cyclomatic

// AddProviderMethods adds a Middleware Provider's methods as Middleware. If this method
// is invoked directly by the user, the same type value's method will be added to all
// *Channel values created by a *Connection.
//
// If a new provider value should be made for each *Channel, consider using
// AddProviderFactory instead.
func (config *ChannelMiddlewares) AddProviderMethods(provider interface{}) {
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

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesChannelReconnect); ok {
		config.AddChannelReconnect(hasMethods.ChannelReconnect)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueDeclare); ok {
		config.AddQueueDeclare(hasMethods.QueueDeclare)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueDeclarePassive); ok {
		config.AddQueueDeclarePassive(hasMethods.QueueDeclarePassive)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueInspect); ok {
		config.AddQueueInspect(hasMethods.QueueInspect)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueDelete); ok {
		config.AddQueueDelete(hasMethods.QueueDelete)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueBind); ok {
		config.AddQueueBind(hasMethods.QueueBind)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueueUnbind); ok {
		config.AddQueueUnbind(hasMethods.QueueUnbind)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQueuePurge); ok {
		config.AddQueuePurge(hasMethods.QueuePurge)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesExchangeDeclare); ok {
		config.AddExchangeDeclare(hasMethods.ExchangeDeclare)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesExchangeDeclarePassive); ok {
		config.AddExchangeDeclarePassive(hasMethods.ExchangeDeclarePassive)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesExchangeDelete); ok {
		config.AddExchangeDelete(hasMethods.ExchangeDelete)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesExchangeBind); ok {
		config.AddExchangeBind(hasMethods.ExchangeBind)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesExchangeUnbind); ok {
		config.AddExchangeUnbind(hasMethods.ExchangeUnbind)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesQoS); ok {
		config.AddQoS(hasMethods.QoS)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesFlow); ok {
		config.AddFlow(hasMethods.Flow)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesConfirm); ok {
		config.AddConfirm(hasMethods.Confirm)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesPublish); ok {
		config.AddPublish(hasMethods.Publish)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesGet); ok {
		config.AddGet(hasMethods.Get)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesConsume); ok {
		config.AddConsume(hasMethods.Consume)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesAck); ok {
		config.AddAck(hasMethods.Ack)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNack); ok {
		config.AddNack(hasMethods.Nack)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesReject); ok {
		config.AddReject(hasMethods.Reject)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyPublish); ok {
		config.AddNotifyPublish(hasMethods.NotifyPublish)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyConfirm); ok {
		config.AddNotifyConfirm(hasMethods.NotifyConfirm)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyConfirmOrOrphaned); ok {
		config.AddNotifyConfirmOrOrphaned(hasMethods.NotifyConfirmOrOrphaned)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyReturn); ok {
		config.AddNotifyReturn(hasMethods.NotifyReturn)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyCancel); ok {
		config.AddNotifyCancel(hasMethods.NotifyCancel)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyFlow); ok {
		config.AddNotifyFlow(hasMethods.NotifyFlow)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyPublishEvents); ok {
		config.AddNotifyPublishEvents(hasMethods.NotifyPublishEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesConsumeEvents); ok {
		config.AddConsumeEvents(hasMethods.ConsumeEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyConfirmEvents); ok {
		config.AddNotifyConfirmEvents(hasMethods.NotifyConfirmEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyConfirmOrOrphanedEvents); ok {
		config.AddNotifyConfirmOrOrphanedEvents(hasMethods.NotifyConfirmOrOrphanedEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyReturnEvents); ok {
		config.AddNotifyReturnEvents(hasMethods.NotifyReturnEvents)
	}

	if hasMethods, ok := provider.(amqpmiddleware.ProvidesNotifyFlowEvents); ok {
		config.AddNotifyFlowEvents(hasMethods.NotifyFlowEvents)
	}
}

//revive:enable:cognitive-complexity
//revive:enable:cyclomatic
