package amqpmiddleware

// Middleware definitions for channel methods.

// Reconnect defines signature for middleware called on underlying channel reconnection
// event.
type Reconnect = func(next HandlerReconnect) HandlerReconnect

// QueueDeclare defines signature for middleware invoked on *amqp.Channel.QueueDeclare()
// and *amqp.Channel.QueueDeclarePassive() call.
type QueueDeclare = func(next HandlerQueueDeclare) HandlerQueueDeclare

// QueueInspect defines signature for middleware invoked on *amqp.Channel.QueueInspect()
// call.
type QueueInspect = func(next HandlerQueueInspect) HandlerQueueInspect

// QueueDelete defines signature for middleware invoked on *amqp.Channel.QueueDelete()
// call.
type QueueDelete = func(next HandlerQueueDelete) HandlerQueueDelete

// QueueBind defines signature for middleware invoked on *amqp.Channel.QueueBind()
// call.
type QueueBind = func(next HandlerQueueBind) HandlerQueueBind

// QueueUnbind defines signature for middleware invoked on *amqp.Channel.QueueUnbind()
// call.
type QueueUnbind = func(next HandlerQueueUnbind) HandlerQueueUnbind

// QueuePurge defines signature for middleware invoked on *amqp.Channel.QueuePurge()
// call.
type QueuePurge = func(next HandlerQueuePurge) HandlerQueuePurge

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

// Consume defines signature for middleware invoked on *amqp.Channel.Get() call.
type Consume func(next HandlerConsume) HandlerConsume

// Ack defines signature for middleware invoked on *amqp.Channel.Ack() call.
type Ack func(next HandlerAck) HandlerAck

// Nack defines signature for middleware invoked on *amqp.Channel.Nack() call.
type Nack func(next HandlerNack) HandlerNack

// Reject defines signature for middleware invoked on *amqp.Channel.Reject() call.
type Reject func(next HandlerReject) HandlerReject

// NotifyPublish defines signature for middleware invoked on
// *amqp.Channel.NotifyPublish() call.
type NotifyPublish func(next HandlerNotifyPublish) HandlerNotifyPublish

// NotifyConfirm defines signature for middleware invoked on
// *amqp.Channel.NotifyConfirm() call.
type NotifyConfirm func(next HandlerNotifyConfirm) HandlerNotifyConfirm

// NotifyConfirmOrOrphaned defines signature for middleware invoked on
// *amqp.Channel.NotifyConfirmOrOrphaned() call.
type NotifyConfirmOrOrphaned func(
	next HandlerNotifyConfirmOrOrphaned,
) HandlerNotifyConfirmOrOrphaned

// NotifyReturn defines signature for middleware invoked on *amqp.Channel.NotifyReturn()
// call.
type NotifyReturn func(next HandlerNotifyReturn) HandlerNotifyReturn

// NotifyCancel defines signature for middleware invoked on *amqp.Channel.NotifyCancel()
// call.
type NotifyCancel func(next HandlerNotifyCancel) HandlerNotifyCancel

// NotifyFlow defines signature for middleware invoked on *amqp.Channel.NotifyFlow()
// call.
type NotifyFlow func(next HandlerNotifyFlow) HandlerNotifyFlow

// NotifyPublishEvent defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyPublish() events.
type NotifyPublishEvent func(next HandlerNotifyPublishEvent) HandlerNotifyPublishEvent

// ConsumeEvent defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.Consume() events.
type ConsumeEvent func(next HandlerConsumeEvent) HandlerConsumeEvent
