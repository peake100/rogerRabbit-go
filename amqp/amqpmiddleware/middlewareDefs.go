package amqpmiddleware

// Middleware definitions for channel methods.

// Reconnect defines signature for middleware called on underlying channel reconnection
// event.
type Reconnect = func(next HandlerReconnect) HandlerReconnect

// QueueDeclare defines signature for middleware invoked on *amqp.Channel.QueueDeclare()
// call.
type QueueDeclare = func(next HandlerQueueDeclare) HandlerQueueDeclare

// QueueDelete defines signature for middleware invoked on *amqp.Channel.QueueDelete()
// call.
type QueueDelete = func(next HandlerQueueDelete) HandlerQueueDelete

// QueueBind defines signature for middleware invoked on *amqp.Channel.QueueBind()
// call.
type QueueBind = func(next HandlerQueueBind) HandlerQueueBind

// QueueUnbind defines signature for middleware invoked on *amqp.Channel.QueueUnbind()
// call.
type QueueUnbind = func(next HandlerQueueUnbind) HandlerQueueUnbind

// ExchangeDeclare defines signature for middleware invoked on
// *amqp.Channel.ExchangeDeclare() call.
type ExchangeDeclare func(next HandlerExchangeDeclare) HandlerExchangeDeclare

// ExchangeDelete defines signature for middleware invoked on
// *amqp.Channel.ExchangeDelete() call.
type ExchangeDelete func(next HandlerExchangeDelete) HandlerExchangeDelete

// ExchangeBind defines signature for middleware invoked on
// *amqp.Channel.ExchangeBind() call.
type ExchangeBind func(next HandlerExchangeBind) HandlerExchangeBind

// ExchangeUnbind defines signature for middleware invoked on
// *amqp.Channel.ExchangeUnbind() call.
type ExchangeUnbind func(next HandlerExchangeUnbind) HandlerExchangeUnbind

// QoS defines signature for middleware invoked on *amqp.Channel.QoS() call.
type QoS func(next HandlerQoS) HandlerQoS

// Flow defines signature for middleware invoked on *amqp.Channel.Flow() call.
type Flow func(next HandlerFlow) HandlerFlow

// Confirm defines signature for middleware invoked on *amqp.Channel.Confirm() call.
type Confirm func(next HandlerConfirm) HandlerConfirm

// Publish defines signature for middleware invoked on *amqp.Channel.Publish() call.
type Publish func(next HandlerPublish) HandlerPublish

// Get defines signature for middleware invoked on *amqp.Channel.Get() call.
type Get func(next HandlerGet) HandlerGet

// Ack defines signature for middleware invoked on *amqp.Channel.Ack() call.
type Ack func(next HandlerAck) HandlerAck

// Nack defines signature for middleware invoked on *amqp.Channel.Nack() call.
type Nack func(next HandlerNack) HandlerNack

// Reject defines signature for middleware invoked on *amqp.Channel.Reject() call.
type Reject func(next HandlerReject) HandlerReject

// NotifyPublish defines signature for middleware invoked on
// *amqp.Channel.NotifyPublish() call.
type NotifyPublish func(next HandlerNotifyPublish) HandlerNotifyPublish

// NotifyPublishEvent defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyPublish() events.
type NotifyPublishEvent func(next HandlerNotifyPublishEvent) HandlerNotifyPublishEvent

// ConsumeEvent defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.Consume() events.
type ConsumeEvent func(next HandlerConsumeEvent) HandlerConsumeEvent
