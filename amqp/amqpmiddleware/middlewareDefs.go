package amqpmiddleware

// Middleware definitions for channel methods.

// ConnectionReconnect defines signature for middleware called on underlying channel
// reconnection event.
type ConnectionReconnect = func(
	next HandlerConnectionReconnect,
) HandlerConnectionReconnect

// ChannelReconnect defines signature for middleware called on underlying channel
// reconnection event.
type ChannelReconnect = func(next HandlerChannelReconnect) HandlerChannelReconnect

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

// NotifyClose defines signature for middleware invoked on the NotifyClose method of
// an amqp.Connection or amqp.Channel.
type NotifyClose func(next HandlerNotifyClose) HandlerNotifyClose

// NotifyDial defines signature for middleware invoked on the NotifyDial method of
// an amqp.Connection or amqp.Channel.
type NotifyDial func(next HandlerNotifyDial) HandlerNotifyDial

// NotifyDisconnect defines signature for middleware invoked on the NotifyDisconnect
// method of an amqp.Connection or amqp.Channel.
type NotifyDisconnect func(next HandlerNotifyDisconnect) HandlerNotifyDisconnect

// Close defines signature for middleware invoked on the Close method of an
// amqp.Connection or amqp.Channel.
type Close func(next HandlerClose) HandlerClose

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

// EVENT MIDDLEWARE #####################################
// ######################################################

// NotifyDialEvents defines signature for middleware invoked on event processed
// during relay of NotifyConnect() events on either an amqp.Connection or amqp.Channel.
type NotifyDialEvents func(
	next HandlerNotifyDialEvents,
) HandlerNotifyDialEvents

// NotifyDisconnectEvents defines signature for middleware invoked on event processed
// during relay of NotifyDisconnect() events on either an amqp.Connection or
// amqp.Channel.
type NotifyDisconnectEvents func(
	next HandlerNotifyDisconnectEvents,
) HandlerNotifyDisconnectEvents

// NotifyCloseEvents defines signature for middleware invoked on event processed
// during relay of NotifyClose() events on either an amqp.Connection or amqp.Channel.
type NotifyCloseEvents func(
	next HandlerNotifyCloseEvents,
) HandlerNotifyCloseEvents

// NotifyPublishEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyPublish() events.
type NotifyPublishEvents func(
	next HandlerNotifyPublishEvents,
) HandlerNotifyPublishEvents

// ConsumeEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.Consume() events.
type ConsumeEvents func(next HandlerConsumeEvents) HandlerConsumeEvents

// NotifyConfirmEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyConfirm() events.
type NotifyConfirmEvents func(
	next HandlerNotifyConfirmEvents,
) HandlerNotifyConfirmEvents

// NotifyConfirmOrOrphanedEvents defines signature for middleware invoked on event
// processed during relay of *amqp.Channel.NotifyConfirmOrOrphaned() events.
type NotifyConfirmOrOrphanedEvents func(
	next HandlerNotifyConfirmOrOrphanedEvents,
) HandlerNotifyConfirmOrOrphanedEvents

// NotifyReturnEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyReturn() events.
type NotifyReturnEvents func(next HandlerNotifyReturnEvents) HandlerNotifyReturnEvents

// NotifyCancelEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyCancel() events.
type NotifyCancelEvents func(next HandlerNotifyCancelEvents) HandlerNotifyCancelEvents

// NotifyFlowEvents defines signature for middleware invoked on event processed
// during relay of *amqp.Channel.NotifyFlow() events.
type NotifyFlowEvents func(next HandlerNotifyFlowEvents) HandlerNotifyFlowEvents
