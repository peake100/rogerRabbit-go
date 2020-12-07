package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"sync"
)

type channelHandlers struct {
	// METHODS MIDDLEWARE

	reconnect amqpMiddleware.HandlerReconnect

	qos     amqpMiddleware.HandlerQoS
	flow    amqpMiddleware.HandlerFlow
	confirm amqpMiddleware.HandlerConfirm

	queueDeclare amqpMiddleware.HandlerQueueDeclare
	queueDelete  amqpMiddleware.HandlerQueueDelete
	queueBind    amqpMiddleware.HandlerQueueBind
	queueUnbind  amqpMiddleware.HandlerQueueUnbind

	exchangeDeclare amqpMiddleware.HandlerExchangeDeclare
	exchangeDelete  amqpMiddleware.HandlerExchangeDelete
	exchangeBind    amqpMiddleware.HandlerExchangeBind
	exchangeUnbind  amqpMiddleware.HandlerExchangeUnbind

	notifyPublish amqpMiddleware.HandlerNotifyPublish

	publish amqpMiddleware.HandlerPublish
	get     amqpMiddleware.HandlerGet

	ack    amqpMiddleware.HandlerAck
	nack   amqpMiddleware.HandlerNack
	reject amqpMiddleware.HandlerReject

	// EVENTS MIDDLEWARE

	notifyPublishEventMiddleware []amqpMiddleware.NotifyPublishEvent
	consumeEventMiddleware       []amqpMiddleware.ConsumeEvent

	lock *sync.RWMutex
}

func (handlers *channelHandlers) AddReconnect(middleware amqpMiddleware.Reconnect) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reconnect = middleware(handlers.reconnect)
}

func (handlers *channelHandlers) AddQoS(
	middleware amqpMiddleware.QoS,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.qos = middleware(handlers.qos)
}

func (handlers *channelHandlers) AddFlow(
	middleware amqpMiddleware.Flow,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.flow = middleware(handlers.flow)
}

func (handlers *channelHandlers) AddConfirm(
	middleware amqpMiddleware.Confirm,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.confirm = middleware(handlers.confirm)
}

func (handlers *channelHandlers) AddQueueDeclare(
	middleware amqpMiddleware.QueueDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDeclare = middleware(handlers.queueDeclare)
}

func (handlers *channelHandlers) AddQueueDelete(middleware amqpMiddleware.QueueDelete) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueDelete = middleware(handlers.queueDelete)
}

func (handlers *channelHandlers) AddQueueBind(middleware amqpMiddleware.QueueBind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueBind = middleware(handlers.queueBind)
}

func (handlers *channelHandlers) AddQueueUnbind(middleware amqpMiddleware.QueueUnbind) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.queueUnbind = middleware(handlers.queueUnbind)
}

func (handlers *channelHandlers) AddExchangeDeclare(
	middleware amqpMiddleware.ExchangeDeclare,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDeclare = middleware(handlers.exchangeDeclare)
}

func (handlers *channelHandlers) AddExchangeDelete(
	middleware amqpMiddleware.ExchangeDelete,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeDelete = middleware(handlers.exchangeDelete)
}

func (handlers *channelHandlers) AddExchangeBind(
	middleware amqpMiddleware.ExchangeBind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeBind = middleware(handlers.exchangeBind)
}

func (handlers *channelHandlers) AddExchangeUnbind(
	middleware amqpMiddleware.ExchangeUnbind,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.exchangeUnbind = middleware(handlers.exchangeUnbind)
}

func (handlers *channelHandlers) AddPublish(
	middleware amqpMiddleware.Publish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.publish = middleware(handlers.publish)
}

func (handlers *channelHandlers) AddGet(
	middleware amqpMiddleware.Get,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.get = middleware(handlers.get)
}

func (handlers *channelHandlers) AddAck(
	middleware amqpMiddleware.Ack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.ack = middleware(handlers.ack)
}

func (handlers *channelHandlers) AddNack(
	middleware amqpMiddleware.Nack,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.nack = middleware(handlers.nack)
}

func (handlers *channelHandlers) AddReject(
	middleware amqpMiddleware.Reject,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reject = middleware(handlers.reject)
}

func (handlers *channelHandlers) AddNotifyPublish(
	middleware amqpMiddleware.NotifyPublish,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublish = middleware(handlers.notifyPublish)
}

func (handlers *channelHandlers) AddNotifyPublishEvent(
	middleware amqpMiddleware.NotifyPublishEvent,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.notifyPublishEventMiddleware = append(
		handlers.notifyPublishEventMiddleware, middleware,
	)
}

func (handlers *channelHandlers) AddConsumeEvent(
	middleware amqpMiddleware.ConsumeEvent,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.consumeEventMiddleware = append(
		handlers.consumeEventMiddleware, middleware,
	)
}

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
