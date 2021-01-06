package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"sync"
)

// channelHandlers hols a Channel's method handlers with applied middleware.
type channelHandlers struct {
	// LIFETIME HANDLERS
	// ----------------

	// reconnect is the handler invoked on a reconnection event.
	reconnect amqpMiddleware.HandlerReconnect

	// MODE HANDLERS
	// -------------

	// qos is the handler for Channel.Qos
	qos     amqpMiddleware.HandlerQoS
	// flow is the handler for Channel.Flow
	flow    amqpMiddleware.HandlerFlow
	// confirm is the handler for Channel.Confirm
	confirm amqpMiddleware.HandlerConfirm

	// QUEUE HANDLERS
	// --------------

	// queueDeclare is the handler for Channel.QueueDeclare
	queueDeclare amqpMiddleware.HandlerQueueDeclare
	// queueDelete is the handler for Channel.QueueDelete
	queueDelete  amqpMiddleware.HandlerQueueDelete
	// queueBind is the handler for Channel.QueueBind
	queueBind    amqpMiddleware.HandlerQueueBind
	// queueUnbind is the handler for Channel.QueueUnbind
	queueUnbind  amqpMiddleware.HandlerQueueUnbind

	// EXCHANGE HANDLERS
	// -----------------

	// exchangeDeclare is the handler for Channel.ExchangeDeclare
	exchangeDeclare amqpMiddleware.HandlerExchangeDeclare
	// exchangeDelete is the handler for Channel.ExchangeDelete
	exchangeDelete  amqpMiddleware.HandlerExchangeDelete
	// exchangeBind is the handler for Channel.ExchangeBind
	exchangeBind    amqpMiddleware.HandlerExchangeBind
	// exchangeUnbind is the handler for Channel.ExchangeUnbind
	exchangeUnbind  amqpMiddleware.HandlerExchangeUnbind

	// NOTIFY HANDLERS
	// ---------------

	// notifyPublish is the handler for Channel.NotifyPublish
	notifyPublish amqpMiddleware.HandlerNotifyPublish

	// MESSAGING HANDLERS
	// ------------------

	// publish is the handler for Channel.Publish
	publish amqpMiddleware.HandlerPublish
	// get is the handler for Channel.Get
	get     amqpMiddleware.HandlerGet

	// ACK HANDLERS
	// ------------

	// ack is the handler for Channel.Ack
	ack    amqpMiddleware.HandlerAck
	// nack is the handler for Channel.Nack
	nack   amqpMiddleware.HandlerNack
	// reject is the handler for Channel.Reject
	reject amqpMiddleware.HandlerReject

	// EVENT MIDDLEWARE
	// ----------------

	// notifyPublishEventMiddleware is middleware to be registered on a
	// notifyPublishRelay.
	notifyPublishEventMiddleware []amqpMiddleware.NotifyPublishEvent
	// consumeEventMiddleware is middleware to be registered on a
	// consumeRelay.
	consumeEventMiddleware       []amqpMiddleware.ConsumeEvent

	// lock should be acquired when adding a middleware to a handler to avoid race
	// conditions.
	lock *sync.RWMutex
}

// AddReconnect adds a new middleware to be invoked on a Channel reconnection event.
func (handlers *channelHandlers) AddReconnect(middleware amqpMiddleware.Reconnect) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reconnect = middleware(handlers.reconnect)
}

// AddQoS adds a new middleware to be invoked on Channel.Qos method calls.
func (handlers *channelHandlers) AddQoS(
	middleware amqpMiddleware.QoS,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.qos = middleware(handlers.qos)
}

// AddFlow adds a new middleware to be invoked on Channel.Flow method calls.
func (handlers *channelHandlers) AddFlow(
	middleware amqpMiddleware.Flow,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.flow = middleware(handlers.flow)
}

// AddConfirm adds a new middleware to be invoked on Channel.Confirm method calls.
func (handlers *channelHandlers) AddConfirm(
	middleware amqpMiddleware.Confirm,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.confirm = middleware(handlers.confirm)
}

// AddQueueDeclare adds a new middleware to be invoked on Channel.QueueDeclare method
// calls.
func (handlers *channelHandlers) AddQueueDeclare(
	middleware amqpMiddleware.QueueDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDeclare = middleware(handlers.queueDeclare)
}

// AddQueueDelete adds a new middleware to be invoked on Channel.QueueDelete method
// calls.
func (handlers *channelHandlers) AddQueueDelete(middleware amqpMiddleware.QueueDelete) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDelete = middleware(handlers.queueDelete)
}

// AddQueueBind adds a new middleware to be invoked on Channel.QueueBind method
// calls.
func (handlers *channelHandlers) AddQueueBind(middleware amqpMiddleware.QueueBind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueBind = middleware(handlers.queueBind)
}

// AddQueueUnbind adds a new middleware to be invoked on Channel.QueueUnbind method
// calls.
func (handlers *channelHandlers) AddQueueUnbind(middleware amqpMiddleware.QueueUnbind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueUnbind = middleware(handlers.queueUnbind)
}

// AddExchangeDeclare adds a new middleware to be invoked on Channel.ExchangeDeclare
// method calls.
func (handlers *channelHandlers) AddExchangeDeclare(
	middleware amqpMiddleware.ExchangeDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclare = middleware(handlers.exchangeDeclare)
}

// AddExchangeDelete adds a new middleware to be invoked on Channel.ExchangeDelete
// method calls.
func (handlers *channelHandlers) AddExchangeDelete(
	middleware amqpMiddleware.ExchangeDelete,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDelete = middleware(handlers.exchangeDelete)
}

// AddExchangeBind adds a new middleware to be invoked on Channel.ExchangeBind
// method calls.
func (handlers *channelHandlers) AddExchangeBind(
	middleware amqpMiddleware.ExchangeBind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeBind = middleware(handlers.exchangeBind)
}

// AddExchangeUnbind adds a new middleware to be invoked on Channel.ExchangeUnbind
// method calls.
func (handlers *channelHandlers) AddExchangeUnbind(
	middleware amqpMiddleware.ExchangeUnbind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeUnbind = middleware(handlers.exchangeUnbind)
}

// AddPublish adds a new middleware to be invoked on Channel.Publish method calls.
func (handlers *channelHandlers) AddPublish(
	middleware amqpMiddleware.Publish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.publish = middleware(handlers.publish)
}

// AddGet adds a new middleware to be invoked on Channel.Get method calls.
func (handlers *channelHandlers) AddGet(
	middleware amqpMiddleware.Get,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.get = middleware(handlers.get)
}

// AddAck adds a new middleware to be invoked on Channel.Ack method calls.
func (handlers *channelHandlers) AddAck(
	middleware amqpMiddleware.Ack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.ack = middleware(handlers.ack)
}

// AddNack adds a new middleware to be invoked on Channel.Nack method calls.
func (handlers *channelHandlers) AddNack(
	middleware amqpMiddleware.Nack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.nack = middleware(handlers.nack)
}

// AddReject adds a new middleware to be invoked on Channel.Reject method calls.
func (handlers *channelHandlers) AddReject(
	middleware amqpMiddleware.Reject,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reject = middleware(handlers.reject)
}

// AddNotifyPublish adds a new middleware to be invoked on Channel.NotifyPublish method
// calls.
func (handlers *channelHandlers) AddNotifyPublish(
	middleware amqpMiddleware.NotifyPublish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublish = middleware(handlers.notifyPublish)
}

// AddNotifyPublishEvent adds a new middleware to be invoked on events sent to callers
// of Channel.NotifyPublish.
func (handlers *channelHandlers) AddNotifyPublishEvent(
	middleware amqpMiddleware.NotifyPublishEvent,
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
	middleware amqpMiddleware.ConsumeEvent,
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
