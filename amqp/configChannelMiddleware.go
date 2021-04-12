package amqp

import "github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"

type ChannelMiddleware struct {
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

	// notifyPublishEvent is middleware to be registered on a
	// notifyPublishRelay.
	notifyPublishEvent []amqpmiddleware.NotifyPublishEvents
	// consumeEvent is middleware to be registered on a
	// consumeRelay.
	consumeEvent []amqpmiddleware.ConsumeEvents
	// notifyConfirmEvent is a middleware to be registered on events for
	// Channel.NotifyConfirm.
	notifyConfirmEvent []amqpmiddleware.NotifyConfirmEvents
	// notifyConfirmOrOrphanedEvent is a middleware to be registered on events for
	// Channel.NotifyConfirmOrOrphaned.
	notifyConfirmOrOrphanedEvent []amqpmiddleware.NotifyConfirmOrOrphanedEvents
	// notifyReturnEvents is a middleware to be registered on events for
	// Channel.NotifyReturn.
	notifyReturnEvents []amqpmiddleware.NotifyReturnEvents
	// notifyCancelEvents is a middleware to be registered on events for
	// Channel.NotifyCancel.
	notifyCancelEvents []amqpmiddleware.NotifyCancelEvents
	// notifyFlowEvents is a middleware to be registered on events for
	// Channel.NotifyFlow.
	notifyFlowEvents []amqpmiddleware.NotifyFlowEvents
}

// AddChannelReconnect adds a new middleware to be invoked on a Channel reconnection
// event.
func (config *ChannelMiddleware) AddChannelReconnect(
	middleware amqpmiddleware.ChannelReconnect,
) {
	config.channelReconnect = append(config.channelReconnect, middleware)
}

// AddQoS adds a new middleware to be invoked on Channel.Qos method calls.
func (config *ChannelMiddleware) AddQoS(middleware amqpmiddleware.QoS) {
	config.qos = append(config.qos, middleware)
}

// AddFlow adds a new middleware to be invoked on Channel.Flow method calls.
func (config *ChannelMiddleware) AddFlow(middleware amqpmiddleware.Flow) {
	config.flow = append(config.flow, middleware)
}

// AddConfirm adds a new middleware to be invoked on Channel.Confirm method calls.
func (config *ChannelMiddleware) AddConfirm(middleware amqpmiddleware.Confirm) {
	config.confirm = append(config.confirm, middleware)
}

// AddQueueDeclare adds a new middleware to be invoked on Channel.QueueDeclare method
// calls.
func (config *ChannelMiddleware) AddQueueDeclare(
	middleware amqpmiddleware.QueueDeclare,
) {
	config.queueDeclare = append(config.queueDeclare, middleware)
}

// AddQueueDeclarePassive adds a new middleware to be invoked on
// Channel.QueueDeclarePassive method calls.
func (config *ChannelMiddleware) AddQueueDeclarePassive(
	middleware amqpmiddleware.QueueDeclare,
) {
	config.queueDeclarePassive = append(config.queueDeclarePassive, middleware)
}

// AddQueueInspect adds a new middleware to be invoked on Channel.QueueInspect method
// calls.
func (config *ChannelMiddleware) AddQueueInspect(
	middleware amqpmiddleware.QueueInspect,
) {
	config.queueInspect = append(config.queueInspect, middleware)
}

// AddQueueDelete adds a new middleware to be invoked on Channel.QueueDelete method
// calls.
func (config *ChannelMiddleware) AddQueueDelete(middleware amqpmiddleware.QueueDelete) {
	config.queueDelete = append(config.queueDelete, middleware)
}

// AddQueueBind adds a new middleware to be invoked on Channel.QueueBind method
// calls.
func (config *ChannelMiddleware) AddQueueBind(middleware amqpmiddleware.QueueBind) {
	config.queueBind = append(config.queueBind, middleware)
}

// AddQueueUnbind adds a new middleware to be invoked on Channel.QueueUnbind method
// calls.
func (config *ChannelMiddleware) AddQueueUnbind(middleware amqpmiddleware.QueueUnbind) {
	config.queueUnbind = append(config.queueUnbind, middleware)
}

// AddQueuePurge adds a new middleware to be invoked on Channel.QueuePurge method
// calls.
func (config *ChannelMiddleware) AddQueuePurge(
	middleware amqpmiddleware.QueuePurge,
) {
	config.queuePurge = append(config.queuePurge, middleware)
}

// AddExchangeDeclare adds a new middleware to be invoked on Channel.ExchangeDeclare
// method calls.
func (config *ChannelMiddleware) AddExchangeDeclare(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	config.exchangeDeclare = append(config.exchangeDeclare, middleware)
}

// AddExchangeDeclarePassive adds a new middleware to be invoked on
// Channel.ExchangeDeclarePassive method calls.
func (config *ChannelMiddleware) AddExchangeDeclarePassive(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	config.exchangeDeclarePassive = append(config.exchangeDeclarePassive, middleware)
}

// AddExchangeDelete adds a new middleware to be invoked on Channel.ExchangeDelete
// method calls.
func (config *ChannelMiddleware) AddExchangeDelete(
	middleware amqpmiddleware.ExchangeDelete,
) {
	config.exchangeDelete = append(config.exchangeDelete, middleware)
}

// AddExchangeBind adds a new middleware to be invoked on Channel.ExchangeBind
// method calls.
func (config *ChannelMiddleware) AddExchangeBind(
	middleware amqpmiddleware.ExchangeBind,
) {
	config.exchangeBind = append(config.exchangeBind, middleware)
}

// AddExchangeUnbind adds a new middleware to be invoked on Channel.ExchangeUnbind
// method calls.
func (config *ChannelMiddleware) AddExchangeUnbind(
	middleware amqpmiddleware.ExchangeUnbind,
) {
	config.exchangeUnbind = append(config.exchangeUnbind, middleware)
}

// AddPublish adds a new middleware to be invoked on Channel.Publish method calls.
func (config *ChannelMiddleware) AddPublish(
	middleware amqpmiddleware.Publish,
) {
	config.publish = append(config.publish, middleware)
}

// AddGet adds a new middleware to be invoked on Channel.Get method calls.
func (config *ChannelMiddleware) AddGet(
	middleware amqpmiddleware.Get,
) {
	config.get = append(config.get, middleware)
}

// AddConsume adds a new middleware to be invoked on Channel.Consume method calls.
//
// NOTE: this is a distinct middleware from AddConsumeEvent, which fires on every
// delivery sent from the broker. This event only fires once when the  Channel.Consume
// method is first called.
func (config *ChannelMiddleware) AddConsume(
	middleware amqpmiddleware.Consume,
) {
	config.consume = append(config.consume, middleware)
}

// AddAck adds a new middleware to be invoked on Channel.Ack method calls.
func (config *ChannelMiddleware) AddAck(
	middleware amqpmiddleware.Ack,
) {
	config.ack = append(config.ack, middleware)
}

// AddNack adds a new middleware to be invoked on Channel.Nack method calls.
func (config *ChannelMiddleware) AddNack(
	middleware amqpmiddleware.Nack,
) {
	config.nack = append(config.nack, middleware)
}

// AddReject adds a new middleware to be invoked on Channel.Reject method calls.
func (config *ChannelMiddleware) AddReject(
	middleware amqpmiddleware.Reject,
) {
	config.reject = append(config.reject, middleware)
}

// AddNotifyPublish adds a new middleware to be invoked on Channel.NotifyPublish method
// calls.
func (config *ChannelMiddleware) AddNotifyPublish(
	middleware amqpmiddleware.NotifyPublish,
) {
	config.notifyPublish = append(config.notifyPublish, middleware)
}

// AddNotifyConfirm adds a new middleware to be invoked on Channel.NotifyConfirm method
// calls.
func (config *ChannelMiddleware) AddNotifyConfirm(
	middleware amqpmiddleware.NotifyConfirm,
) {
	config.notifyConfirm = append(config.notifyConfirm, middleware)
}

// AddNotifyConfirmOrOrphaned adds a new middleware to be invoked on
// Channel.NotifyConfirmOrOrphaned method calls.
func (config *ChannelMiddleware) AddNotifyConfirmOrOrphaned(
	middleware amqpmiddleware.NotifyConfirmOrOrphaned,
) {
	config.notifyConfirmOrOrphaned = append(config.notifyConfirmOrOrphaned, middleware)
}

// AddNotifyReturn adds a new middleware to be invoked on Channel.NotifyReturn method
// calls.
func (config *ChannelMiddleware) AddNotifyReturn(
	middleware amqpmiddleware.NotifyReturn,
) {
	config.notifyReturn = append(config.notifyReturn, middleware)
}

// AddNotifyCancel adds a new middleware to be invoked on Channel.NotifyCancel method
// calls.
func (config *ChannelMiddleware) AddNotifyCancel(
	middleware amqpmiddleware.NotifyCancel,
) {
	config.notifyCancel = append(config.notifyCancel, middleware)
}

// AddNotifyFlow adds a new middleware to be invoked on Channel.NotifyFlow method
// calls.
func (config *ChannelMiddleware) AddNotifyFlow(
	middleware amqpmiddleware.NotifyFlow,
) {
	config.notifyFlow = append(config.notifyFlow, middleware)
}

// AddNotifyPublishEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyPublish.
func (config *ChannelMiddleware) AddNotifyPublishEvent(
	middleware amqpmiddleware.NotifyPublishEvents,
) {
	config.notifyPublishEvent = append(config.notifyPublishEvent, middleware)
}

// AddConsumeEvent adds a new middleware to be invoked on events sent to callers
// of Channel.Consume.
func (config *ChannelMiddleware) AddConsumeEvent(
	middleware amqpmiddleware.ConsumeEvents,
) {
	config.consumeEvent = append(config.consumeEvent, middleware)
}

// AddNotifyConfirmEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyConfirm.
func (config *ChannelMiddleware) AddNotifyConfirmEvent(
	middleware amqpmiddleware.NotifyConfirmEvents,
) {
	config.notifyConfirmEvent = append(
		config.notifyConfirmEvent, middleware,
	)
}

// AddNotifyConfirmOrOrphanedEvent adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyConfirmOrOrphaned.
func (config *ChannelMiddleware) AddNotifyConfirmOrOrphanedEvent(
	middleware amqpmiddleware.NotifyConfirmOrOrphanedEvents,
) {
	config.notifyConfirmOrOrphanedEvent = append(
		config.notifyConfirmOrOrphanedEvent, middleware,
	)
}

// AddNotifyReturnEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyReturn.
func (config *ChannelMiddleware) AddNotifyReturnEvents(
	middleware amqpmiddleware.NotifyReturnEvents,
) {
	config.notifyReturnEvents = append(
		config.notifyReturnEvents, middleware,
	)
}

// AddNotifyCancelEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (config *ChannelMiddleware) AddNotifyCancelEvents(
	middleware amqpmiddleware.NotifyCancelEvents,
) {
	config.notifyCancelEvents = append(
		config.notifyCancelEvents, middleware,
	)
}

// AddNotifyFlowEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (config *ChannelMiddleware) AddNotifyFlowEvents(
	middleware amqpmiddleware.NotifyFlowEvents,
) {
	config.notifyFlowEvents = append(
		config.notifyFlowEvents, middleware,
	)
}
