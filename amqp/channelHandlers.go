package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"sync"
)

// channelHandlers hols a Channel's method handlers with applied middleware.
type channelHandlers struct {
	// LIFETIME HANDLERS
	// ----------------

	// reconnect is the handler invoked on a reconnection event.
	reconnect amqpmiddleware.HandlerReconnect

	// MODE HANDLERS
	// -------------

	// qos is the handler for Channel.Qos
	qos amqpmiddleware.HandlerQoS
	// flow is the handler for Channel.Flow
	flow amqpmiddleware.HandlerFlow
	// confirm is the handler for Channel.Confirm
	confirm amqpmiddleware.HandlerConfirm

	// QUEUE HANDLERS
	// --------------

	// queueDeclare is the handler for Channel.QueueDeclare
	queueDeclare amqpmiddleware.HandlerQueueDeclare
	// queueDeclarePassive is the handler for Channel.QueueDeclare
	queueDeclarePassive amqpmiddleware.HandlerQueueDeclare
	// queueInspect is the handler for Channel.QueueInspect
	queueInspect amqpmiddleware.HandlerQueueInspect
	// queueDelete is the handler for Channel.QueueDelete
	queueDelete amqpmiddleware.HandlerQueueDelete
	// queueBind is the handler for Channel.QueueBind
	queueBind amqpmiddleware.HandlerQueueBind
	// queueUnbind is the handler for Channel.QueueUnbind
	queueUnbind amqpmiddleware.HandlerQueueUnbind
	// queueInspect is the handler for Channel.QueueInspect
	queuePurge amqpmiddleware.HandlerQueuePurge

	// EXCHANGE HANDLERS
	// -----------------

	// exchangeDeclare is the handler for Channel.ExchangeDeclare
	exchangeDeclare amqpmiddleware.HandlerExchangeDeclare
	// exchangeDeclarePassive is the handler for Channel.ExchangeDeclare
	exchangeDeclarePassive amqpmiddleware.HandlerExchangeDeclare
	// exchangeDelete is the handler for Channel.ExchangeDelete
	exchangeDelete amqpmiddleware.HandlerExchangeDelete
	// exchangeBind is the handler for Channel.ExchangeBind
	exchangeBind amqpmiddleware.HandlerExchangeBind
	// exchangeUnbind is the handler for Channel.ExchangeUnbind
	exchangeUnbind amqpmiddleware.HandlerExchangeUnbind

	// NOTIFY HANDLERS
	// ---------------

	// notifyPublish is the handler for Channel.NotifyPublish
	notifyPublish amqpmiddleware.HandlerNotifyPublish
	// consume is the handler for Channel.Consume
	consume amqpmiddleware.HandlerConsume
	// notifyConfirm is the handler for Channel.NotifyConfirm
	notifyConfirm amqpmiddleware.HandlerNotifyConfirm
	// notifyConfirmOrOrphaned is the handler for Channel.NotifyConfirmOrOrphaned
	notifyConfirmOrOrphaned amqpmiddleware.HandlerNotifyConfirmOrOrphaned
	// notifyReturn is the handler for Channel.NotifyReturn
	notifyReturn amqpmiddleware.HandlerNotifyReturn
	// notifyCancel is the handler for Channel.NotifyCancel
	notifyCancel amqpmiddleware.HandlerNotifyCancel
	// notifyFlow is the handler for Channel.NotifyFlow
	notifyFlow amqpmiddleware.HandlerNotifyFlow

	// MESSAGING HANDLERS
	// ------------------

	// publish is the handler for Channel.Publish
	publish amqpmiddleware.HandlerPublish
	// get is the handler for Channel.Get
	get amqpmiddleware.HandlerGet

	// ACK HANDLERS
	// ------------

	// ack is the handler for Channel.Ack
	ack amqpmiddleware.HandlerAck
	// nack is the handler for Channel.Nack
	nack amqpmiddleware.HandlerNack
	// reject is the handler for Channel.Reject
	reject amqpmiddleware.HandlerReject

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

	// lock should be acquired when adding a middleware to a handler to avoid race
	// conditions.
	lock *sync.RWMutex
}

// AddReconnect adds a new middleware to be invoked on a Channel reconnection event.
func (handlers *channelHandlers) AddReconnect(middleware amqpmiddleware.Reconnect) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reconnect = middleware(handlers.reconnect)
}

// AddQoS adds a new middleware to be invoked on Channel.Qos method calls.
func (handlers *channelHandlers) AddQoS(
	middleware amqpmiddleware.QoS,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.qos = middleware(handlers.qos)
}

// AddFlow adds a new middleware to be invoked on Channel.Flow method calls.
func (handlers *channelHandlers) AddFlow(
	middleware amqpmiddleware.Flow,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.flow = middleware(handlers.flow)
}

// AddConfirm adds a new middleware to be invoked on Channel.Confirm method calls.
func (handlers *channelHandlers) AddConfirm(
	middleware amqpmiddleware.Confirm,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.confirm = middleware(handlers.confirm)
}

// AddQueueDeclare adds a new middleware to be invoked on Channel.QueueDeclare method
// calls.
func (handlers *channelHandlers) AddQueueDeclare(
	middleware amqpmiddleware.QueueDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDeclare = middleware(handlers.queueDeclare)
}

// AddQueueDeclarePassive adds a new middleware to be invoked on
// Channel.QueueDeclarePassive method calls.
func (handlers *channelHandlers) AddQueueDeclarePassive(
	middleware amqpmiddleware.QueueDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDeclarePassive = middleware(handlers.queueDeclarePassive)
}

// AddQueueInspect adds a new middleware to be invoked on Channel.QueueInspect method
// calls.
func (handlers *channelHandlers) AddQueueInspect(
	middleware amqpmiddleware.QueueInspect,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueInspect = middleware(handlers.queueInspect)
}

// AddQueueDelete adds a new middleware to be invoked on Channel.QueueDelete method
// calls.
func (handlers *channelHandlers) AddQueueDelete(middleware amqpmiddleware.QueueDelete) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDelete = middleware(handlers.queueDelete)
}

// AddQueueBind adds a new middleware to be invoked on Channel.QueueBind method
// calls.
func (handlers *channelHandlers) AddQueueBind(middleware amqpmiddleware.QueueBind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueBind = middleware(handlers.queueBind)
}

// AddQueueUnbind adds a new middleware to be invoked on Channel.QueueUnbind method
// calls.
func (handlers *channelHandlers) AddQueueUnbind(middleware amqpmiddleware.QueueUnbind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueUnbind = middleware(handlers.queueUnbind)
}

// AddQueuePurge adds a new middleware to be invoked on Channel.QueuePurge method
// calls.
func (handlers *channelHandlers) AddQueuePurge(
	middleware amqpmiddleware.QueuePurge,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queuePurge = middleware(handlers.queuePurge)
}

// AddExchangeDeclare adds a new middleware to be invoked on Channel.ExchangeDeclare
// method calls.
func (handlers *channelHandlers) AddExchangeDeclare(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclare = middleware(handlers.exchangeDeclare)
}

// AddExchangeDeclarePassive adds a new middleware to be invoked on
// Channel.ExchangeDeclarePassive method calls.
func (handlers *channelHandlers) AddExchangeDeclarePassive(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclarePassive = middleware(handlers.exchangeDeclarePassive)
}

// AddExchangeDelete adds a new middleware to be invoked on Channel.ExchangeDelete
// method calls.
func (handlers *channelHandlers) AddExchangeDelete(
	middleware amqpmiddleware.ExchangeDelete,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDelete = middleware(handlers.exchangeDelete)
}

// AddExchangeBind adds a new middleware to be invoked on Channel.ExchangeBind
// method calls.
func (handlers *channelHandlers) AddExchangeBind(
	middleware amqpmiddleware.ExchangeBind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeBind = middleware(handlers.exchangeBind)
}

// AddExchangeUnbind adds a new middleware to be invoked on Channel.ExchangeUnbind
// method calls.
func (handlers *channelHandlers) AddExchangeUnbind(
	middleware amqpmiddleware.ExchangeUnbind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeUnbind = middleware(handlers.exchangeUnbind)
}

// AddPublish adds a new middleware to be invoked on Channel.Publish method calls.
func (handlers *channelHandlers) AddPublish(
	middleware amqpmiddleware.Publish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.publish = middleware(handlers.publish)
}

// AddGet adds a new middleware to be invoked on Channel.Get method calls.
func (handlers *channelHandlers) AddGet(
	middleware amqpmiddleware.Get,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.get = middleware(handlers.get)
}

// AddConsume adds a new middleware to be invoked on Channel.Consume method calls.
//
// NOTE: this is a distinct middleware from AddConsumeEvent, which fires on every
// delivery sent from the broker. This event only fires once when the  Channel.Consume
// method is first called.
func (handlers *channelHandlers) AddConsume(
	middleware amqpmiddleware.Consume,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.consume = middleware(handlers.consume)
}

// AddAck adds a new middleware to be invoked on Channel.Ack method calls.
func (handlers *channelHandlers) AddAck(
	middleware amqpmiddleware.Ack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.ack = middleware(handlers.ack)
}

// AddNack adds a new middleware to be invoked on Channel.Nack method calls.
func (handlers *channelHandlers) AddNack(
	middleware amqpmiddleware.Nack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.nack = middleware(handlers.nack)
}

// AddReject adds a new middleware to be invoked on Channel.Reject method calls.
func (handlers *channelHandlers) AddReject(
	middleware amqpmiddleware.Reject,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reject = middleware(handlers.reject)
}

// AddNotifyPublish adds a new middleware to be invoked on Channel.NotifyPublish method
// calls.
func (handlers *channelHandlers) AddNotifyPublish(
	middleware amqpmiddleware.NotifyPublish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublish = middleware(handlers.notifyPublish)
}

// AddNotifyConfirm adds a new middleware to be invoked on Channel.NotifyConfirm method
// calls.
func (handlers *channelHandlers) AddNotifyConfirm(
	middleware amqpmiddleware.NotifyConfirm,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyConfirm = middleware(handlers.notifyConfirm)
}

// AddNotifyConfirmOrOrphaned adds a new middleware to be invoked on
// Channel.NotifyConfirmOrOrphaned method calls.
func (handlers *channelHandlers) AddNotifyConfirmOrOrphaned(
	middleware amqpmiddleware.NotifyConfirmOrOrphaned,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyConfirmOrOrphaned = middleware(handlers.notifyConfirmOrOrphaned)
}

// AddNotifyReturn adds a new middleware to be invoked on Channel.NotifyReturn method
// calls.
func (handlers *channelHandlers) AddNotifyReturn(
	middleware amqpmiddleware.NotifyReturn,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyReturn = middleware(handlers.notifyReturn)
}

// AddNotifyCancel adds a new middleware to be invoked on Channel.NotifyCancel method
// calls.
func (handlers *channelHandlers) AddNotifyCancel(
	middleware amqpmiddleware.NotifyCancel,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyCancel = middleware(handlers.notifyCancel)
}

// AddNotifyFlow adds a new middleware to be invoked on Channel.NotifyFlow method
// calls.
func (handlers *channelHandlers) AddNotifyFlow(
	middleware amqpmiddleware.NotifyFlow,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyFlow = middleware(handlers.notifyFlow)
}

// AddNotifyPublishEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyPublish.
func (handlers *channelHandlers) AddNotifyPublishEvent(
	middleware amqpmiddleware.NotifyPublishEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublishEvent = append(
		handlers.notifyPublishEvent, middleware,
	)
}

// AddConsumeEvent adds a new middleware to be invoked on events sent to callers
// of Channel.Consume.
func (handlers *channelHandlers) AddConsumeEvent(
	middleware amqpmiddleware.ConsumeEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.consumeEvent = append(
		handlers.consumeEvent, middleware,
	)
}

// AddNotifyConfirmEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyConfirm.
func (handlers *channelHandlers) AddNotifyConfirmEvent(
	middleware amqpmiddleware.NotifyConfirmEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyConfirmEvent = append(
		handlers.notifyConfirmEvent, middleware,
	)
}

// AddNotifyConfirmOrOrphanedEvent adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyConfirmOrOrphaned.
func (handlers *channelHandlers) AddNotifyConfirmOrOrphanedEvent(
	middleware amqpmiddleware.NotifyConfirmOrOrphanedEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyConfirmOrOrphanedEvent = append(
		handlers.notifyConfirmOrOrphanedEvent, middleware,
	)
}

// AddNotifyReturnEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyReturn.
func (handlers *channelHandlers) AddNotifyReturnEvents(
	middleware amqpmiddleware.NotifyReturnEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyReturnEvents = append(
		handlers.notifyReturnEvents, middleware,
	)
}

// AddNotifyCancelEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (handlers *channelHandlers) AddNotifyCancelEvents(
	middleware amqpmiddleware.NotifyCancelEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyCancelEvents = append(
		handlers.notifyCancelEvents, middleware,
	)
}

// AddNotifyFlowEvents adds a new middleware to be invoked on events sent to
// callers of Channel.NotifyCancel.
func (handlers *channelHandlers) AddNotifyFlowEvents(
	middleware amqpmiddleware.NotifyFlowEvents,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyFlowEvents = append(
		handlers.notifyFlowEvents, middleware,
	)
}

// newChannelHandlers created a new channelHandlers with all base handlers added.
func newChannelHandlers(conn *Connection, channel *Channel) *channelHandlers {
	baseBuilder := &middlewareBaseBuilder{
		connection: conn,
		channel:    channel,
	}

	return &channelHandlers{
		reconnect:               baseBuilder.createBaseHandlerReconnect(),
		queueDeclare:            baseBuilder.createBaseHandlerQueueDeclare(),
		queueDeclarePassive:     baseBuilder.createBaseHandlerQueueDeclarePassive(),
		queueInspect:            baseBuilder.createBaseHandlerQueueInspect(),
		queueDelete:             baseBuilder.createBaseHandlerQueueDelete(),
		queueBind:               baseBuilder.createBaseHandlerQueueBind(),
		queueUnbind:             baseBuilder.createBaseHandlerQueueUnbind(),
		queuePurge:              baseBuilder.createBaseHandlerQueuePurge(),
		exchangeDeclare:         baseBuilder.createBaseHandlerExchangeDeclare(),
		exchangeDeclarePassive:  baseBuilder.createBaseHandlerExchangeDeclarePassive(),
		exchangeDelete:          baseBuilder.createBaseHandlerExchangeDelete(),
		exchangeBind:            baseBuilder.createBaseHandlerExchangeBind(),
		exchangeUnbind:          baseBuilder.createBaseHandlerExchangeUnbind(),
		qos:                     baseBuilder.createBaseHandlerQoS(),
		flow:                    baseBuilder.createBaseHandlerFlow(),
		confirm:                 baseBuilder.createBaseHandlerConfirm(),
		publish:                 baseBuilder.createBaseHandlerPublish(),
		get:                     baseBuilder.createBaseHandlerGet(),
		consume:                 baseBuilder.createBaseHandlerConsume(),
		ack:                     baseBuilder.createBaseHandlerAck(),
		nack:                    baseBuilder.createBaseHandlerNack(),
		reject:                  baseBuilder.createBaseHandlerReject(),
		notifyPublish:           baseBuilder.createBaseHandlerNotifyPublish(),
		notifyConfirm:           baseBuilder.createBaseHandlerNotifyConfirm(),
		notifyConfirmOrOrphaned: baseBuilder.createBaseHandlerNotifyConfirmOrOrphaned(),
		notifyReturn:            baseBuilder.createBaseHandlerNotifyReturn(),
		notifyCancel:            baseBuilder.createBaseHandlerNotifyCancel(),
		notifyFlow:              baseBuilder.createBaseHandlerNotifyFlow(),

		notifyPublishEvent: nil,
		consumeEvent:       nil,
		notifyConfirmEvent: nil,
		notifyReturnEvents: nil,
		notifyCancelEvents: nil,
		notifyFlowEvents:   nil,

		lock: new(sync.RWMutex),
	}
}
