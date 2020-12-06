package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"sync"
)

type channelHandlers struct {
	reconnect    amqpMiddleware.HandlerReconnect
	queueDeclare amqpMiddleware.HandlerQueueDeclare
	queueDelete  amqpMiddleware.HandlerQueueDelete
	queueBind    amqpMiddleware.HandlerQueueBind
	queueUnbind  amqpMiddleware.HandlerQueueUnbind

	exchangeDeclare amqpMiddleware.HandlerExchangeDeclare
	exchangeDelete  amqpMiddleware.HandlerExchangeDelete
	exchangeBind    amqpMiddleware.HandlerExchangeBind
	exchangeUnbind  amqpMiddleware.HandlerExchangeUnbind

	qos amqpMiddleware.HandlerQoS

	lock *sync.RWMutex
}

func (handlers *channelHandlers) AddReconnect(middleware amqpMiddleware.Reconnect) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.reconnect = middleware(handlers.reconnect)
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

func (handlers *channelHandlers) AddQoS(
	middleware amqpMiddleware.QoS,
) {
	handlers.lock.Lock()
	defer handlers.lock.Unlock()

	handlers.qos = middleware(handlers.qos)
}

func newChannelHandlers(conn *Connection) *channelHandlers {
	baseBuilder := &middlewareBaseBuilder{
		connection: conn,
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
		qos: 			 baseBuilder.createBaseHandlerQoS(),
		lock:            new(sync.RWMutex),
	}
}
