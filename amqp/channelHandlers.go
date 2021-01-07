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
	// queueDelete is the handler for Channel.QueueDelete
	queueDelete amqpmiddleware.HandlerQueueDelete
	// queueBind is the handler for Channel.QueueBind
	queueBind amqpmiddleware.HandlerQueueBind
	// queueUnbind is the handler for Channel.QueueUnbind
	queueUnbind amqpmiddleware.HandlerQueueUnbind

	// EXCHANGE HANDLERS
	// -----------------

	// exchangeDeclare is the handler for Channel.ExchangeDeclare
	exchangeDeclare amqpmiddleware.HandlerExchangeDeclare
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

	// notifyPublishEventMiddleware is middleware to be registered on a
	// notifyPublishRelay.
	notifyPublishEventMiddleware []amqpmiddleware.NotifyPublishEvent
	// consumeEventMiddleware is middleware to be registered on a
	// consumeRelay.
	consumeEventMiddleware []amqpmiddleware.ConsumeEvent

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

// AddExchangeDeclare adds a new middleware to be invoked on Channel.ExchangeDeclare
// method calls.
func (handlers *channelHandlers) AddExchangeDeclare(
	middleware amqpmiddleware.ExchangeDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclare = middleware(handlers.exchangeDeclare)
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

// AddNotifyPublishEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyPublish.
func (handlers *channelHandlers) AddNotifyPublishEvent(
	middleware amqpmiddleware.NotifyPublishEvent,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublishEventMiddleware = append(
		handlers.notifyPublishEventMiddleware, middleware,
	)
}

// AddConsumeEvent adds a new middleware to be invoked on events sent to callers
// of Channel.Consume.
func (handlers *channelHandlers) AddConsumeEvent(
	middleware amqpmiddleware.ConsumeEvent,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.consumeEventMiddleware = append(
		handlers.consumeEventMiddleware, middleware,
	)
}

// newChannelHandlers created a new channelHandlers with all base handlers added.
func newChannelHandlers(conn *Connection, channel *Channel) *channelHandlers {
	baseBuilder := &middlewareBaseBuilder{
		connection: conn,
		channel:    channel,
	}

	return &channelHandlers{
		reconnect:       baseBuilder.createBaseHandlerReconnect(conn),
		queueDeclare:    baseBuilder.createBaseHandlerQueueDeclare(),
		queueDelete:     baseBuilder.createBaseHandlerQueueDelete(),
		queueBind:       baseBuilder.createBaseHandlerQueueBind(),
		queueUnbind:     baseBuilder.createBaseHandlerQueueUnbind(),
		exchangeDeclare: baseBuilder.createBaseHandlerExchangeDeclare(),
		exchangeDelete:  baseBuilder.createBaseHandlerExchangeDelete(),
		exchangeBind:    baseBuilder.createBaseHandlerExchangeBind(),
		exchangeUnbind:  baseBuilder.createBaseHandlerExchangeUnbind(),
		qos:             baseBuilder.createBaseHandlerQoS(),
		flow:            baseBuilder.createBaseHandlerFlow(),
		confirm:         baseBuilder.createBaseHandlerConfirm(),
		publish:         baseBuilder.createBaseHandlerPublish(),
		get:             baseBuilder.createBaseHandlerGet(),
		ack:             baseBuilder.createBaseHandlerAck(),
		nack:            baseBuilder.createBaseHandlerNack(),
		reject:          baseBuilder.createBaseHandlerReject(),
		notifyPublish:   baseBuilder.createBaseHandlerNotifyPublish(),

		notifyPublishEventMiddleware: nil,

		lock: new(sync.RWMutex),
	}
}
