package amqpmiddleware

type ProviderTypeID string

// ProvidesMiddleware must be implemented by any middleware provider type. Provider
// types expose methods which implement Connection or Channel middleware and can be
// useful for creating complex middlewares that interact with multiple methods.
//
// The DialConfig type has helper methods on it's Middleware configuration fields for
// automatically registering ProviderMethods as middleware without having to do so by
// hand.
type ProvidesMiddleware interface {
	// ProviderTypeID is an ID for the provider type, so the instance of the provider
	// can be retrieved and inspected during testing.
	TypeID() ProviderTypeID
}

// SHARED PROVIDERS ##################
// ###################################

// ProvidesClose provides Close as a method.
type ProvidesClose interface {
	ProvidesMiddleware
	Close(next HandlerClose) HandlerClose
}

// ProvidesNotifyClose provides NotifyClose as a method.
type ProvidesNotifyClose interface {
	ProvidesMiddleware
	NotifyClose(next HandlerNotifyClose) HandlerNotifyClose
}

// ProvidesNotifyDial provides NotifyDial as a method.
type ProvidesNotifyDial interface {
	ProvidesMiddleware
	NotifyDial(next HandlerNotifyDial) HandlerNotifyDial
}

// ProvidesNotifyDisconnect provides NotifyDisconnect as a method.
type ProvidesNotifyDisconnect interface {
	ProvidesMiddleware
	NotifyDisconnect(next HandlerNotifyDisconnect) HandlerNotifyDisconnect
}

// SHARED EVENT PROVIDERS ###############################
// ######################################################

// ProvidesNotifyDialEvents provides NotifyDialEvents as a method.
type ProvidesNotifyDialEvents interface {
	ProvidesMiddleware
	NotifyDialEvents(next HandlerNotifyDialEvents) HandlerNotifyDialEvents
}

// ProvidesNotifyDisconnectEvents provides NotifyDisconnectEvents as a method.
type ProvidesNotifyDisconnectEvents interface {
	ProvidesMiddleware
	NotifyDisconnectEvents(next HandlerNotifyDisconnectEvents) HandlerNotifyDisconnectEvents
}

// ProvidesNotifyCloseEvents provides NotifyCloseEvents as a method.
type ProvidesNotifyCloseEvents interface {
	ProvidesMiddleware
	NotifyCloseEvents(next HandlerNotifyCloseEvents) HandlerNotifyCloseEvents
}

// CONNECTION PROVIDERS ################################
// #####################################################

// ProvidesConnectionReconnect provides ConnectionReconnect as a method.
type ProvidesConnectionReconnect interface {
	ProvidesMiddleware
	ConnectionReconnect(next HandlerConnectionReconnect) HandlerConnectionReconnect
}

// CHANNEL PROVIDERS ##################################
// ####################################################

// ProvidesChannelReconnect provides ChannelReconnect as a method.
type ProvidesChannelReconnect interface {
	ProvidesMiddleware
	ChannelReconnect(next HandlerChannelReconnect) HandlerChannelReconnect
}

// ProvidesQueueDeclare provides QueueDeclare as a method.
type ProvidesQueueDeclare interface {
	ProvidesMiddleware
	QueueDeclare(next HandlerQueueDeclare) HandlerQueueDeclare
}

// ProvidesQueueDeclarePassive provides QueueDeclare as a method to be used for
// amqp.Channel.QueueDeclarePassive handlers.
type ProvidesQueueDeclarePassive interface {
	ProvidesMiddleware
	QueueDeclarePassive(next HandlerQueueDeclare) HandlerQueueDeclare
}

// ProvidesQueueInspect provides QueueInspect as a method.
type ProvidesQueueInspect interface {
	ProvidesMiddleware
	QueueInspect(next HandlerQueueInspect) HandlerQueueInspect
}

// ProvidesQueueDelete provides QueueDelete as a method.
type ProvidesQueueDelete interface {
	ProvidesMiddleware
	QueueDelete(next HandlerQueueDelete) HandlerQueueDelete
}

// ProvidesQueueBind provides QueueBind as a method.
type ProvidesQueueBind interface {
	ProvidesMiddleware
	QueueBind(next HandlerQueueBind) HandlerQueueBind
}

// ProvidesQueueUnbind provides QueueUnbind as a method.
type ProvidesQueueUnbind interface {
	ProvidesMiddleware
	QueueUnbind(next HandlerQueueUnbind) HandlerQueueUnbind
}

// ProvidesQueuePurge provides QueuePurge as a method.
type ProvidesQueuePurge interface {
	ProvidesMiddleware
	QueuePurge(next HandlerQueuePurge) HandlerQueuePurge
}

// ProvidesExchangeDeclare provides ExchangeDeclare as a method.
type ProvidesExchangeDeclare interface {
	ProvidesMiddleware
	ExchangeDeclare(next HandlerExchangeDeclare) HandlerExchangeDeclare
}

// ProvidesExchangeDeclare provides ExchangeDeclare as a method.
type ProvidesExchangeDeclarePassive interface {
	ProvidesMiddleware
	ExchangeDeclarePassive(next HandlerExchangeDeclare) HandlerExchangeDeclare
}

// ProvidesExchangeDelete provides ExchangeDelete as a method.
type ProvidesExchangeDelete interface {
	ProvidesMiddleware
	ExchangeDelete(next HandlerExchangeDelete) HandlerExchangeDelete
}

// ProvidesExchangeBind provides ExchangeBind as a method.
type ProvidesExchangeBind interface {
	ProvidesMiddleware
	ExchangeBind(next HandlerExchangeBind) HandlerExchangeBind
}

// ProvidesExchangeUnbind provides ExchangeUnbind as a method.
type ProvidesExchangeUnbind interface {
	ProvidesMiddleware
	ExchangeUnbind(next HandlerExchangeUnbind) HandlerExchangeUnbind
}

// ProvidesQoS provides QoS as a method.
type ProvidesQoS interface {
	ProvidesMiddleware
	QoS(next HandlerQoS) HandlerQoS
}

// ProvidesFlow provides Flow as a method.
type ProvidesFlow interface {
	ProvidesMiddleware
	Flow(next HandlerFlow) HandlerFlow
}

// ProvidesConfirm provides Confirm as a method.
type ProvidesConfirm interface {
	ProvidesMiddleware
	Confirm(next HandlerConfirm) HandlerConfirm
}

// ProvidesPublish provides Publish as a method.
type ProvidesPublish interface {
	ProvidesMiddleware
	Publish(next HandlerPublish) HandlerPublish
}

// ProvidesGet provides Get as a method.
type ProvidesGet interface {
	ProvidesMiddleware
	Get(next HandlerGet) HandlerGet
}

// ProvidesConsume provides Consume as a method.
type ProvidesConsume interface {
	ProvidesMiddleware
	Consume(next HandlerConsume) HandlerConsume
}

// ProvidesAck provides Ack as a method.
type ProvidesAck interface {
	ProvidesMiddleware
	Ack(next HandlerAck) HandlerAck
}

// ProvidesNack provides Nack as a method.
type ProvidesNack interface {
	ProvidesMiddleware
	Nack(next HandlerNack) HandlerNack
}

// ProvidesReject provides Reject as a method.
type ProvidesReject interface {
	ProvidesMiddleware
	Reject(next HandlerReject) HandlerReject
}

// ProvidesNotifyPublish provides NotifyPublish as a method.
type ProvidesNotifyPublish interface {
	ProvidesMiddleware
	NotifyPublish(next HandlerNotifyPublish) HandlerNotifyPublish
}

// ProvidesNotifyConfirm provides NotifyConfirm as a method.
type ProvidesNotifyConfirm interface {
	ProvidesMiddleware
	NotifyConfirm(next HandlerNotifyConfirm) HandlerNotifyConfirm
}

// ProvidesNotifyConfirmOrOrphaned provides NotifyConfirmOrOrphaned as a method.
type ProvidesNotifyConfirmOrOrphaned interface {
	ProvidesMiddleware
	NotifyConfirmOrOrphaned(next HandlerNotifyConfirmOrOrphaned) HandlerNotifyConfirmOrOrphaned
}

// ProvidesNotifyReturn provides NotifyReturn as a method.
type ProvidesNotifyReturn interface {
	ProvidesMiddleware
	NotifyReturn(next HandlerNotifyReturn) HandlerNotifyReturn
}

// ProvidesNotifyCancel provides NotifyCancel as a method.
type ProvidesNotifyCancel interface {
	ProvidesMiddleware
	NotifyCancel(next HandlerNotifyCancel) HandlerNotifyCancel
}

// ProvidesNotifyFlow provides NotifyFlow as a method.
type ProvidesNotifyFlow interface {
	ProvidesMiddleware
	NotifyFlow(next HandlerNotifyFlow) HandlerNotifyFlow
}

// CHANNEL EVENT PROVIDERS ############################
// ####################################################

// ProvidesNotifyPublishEvents provides NotifyPublishEvents as a method.
type ProvidesNotifyPublishEvents interface {
	ProvidesMiddleware
	NotifyPublishEvents(next HandlerNotifyPublishEvents) HandlerNotifyPublishEvents
}

// ProvidesConsumeEvents provides ConsumeEvents as a method.
type ProvidesConsumeEvents interface {
	ProvidesMiddleware
	ConsumeEvents(next HandlerConsumeEvents) HandlerConsumeEvents
}

// ProvidesNotifyConfirmEvents provides NotifyConfirmEvents as a method.
type ProvidesNotifyConfirmEvents interface {
	ProvidesMiddleware
	NotifyConfirmEvents(next HandlerNotifyConfirmEvents) HandlerNotifyConfirmEvents
}

// ProvidesNotifyConfirmOrOrphanedEvents provides NotifyConfirmOrOrphanedEvents as a
// method.
type ProvidesNotifyConfirmOrOrphanedEvents interface {
	ProvidesMiddleware
	NotifyConfirmOrOrphanedEvents(next HandlerNotifyConfirmOrOrphanedEvents) HandlerNotifyConfirmOrOrphanedEvents
}

// ProvidesNotifyReturnEvents provides NotifyReturnEvents as a method.
type ProvidesNotifyReturnEvents interface {
	ProvidesMiddleware
	NotifyReturnEvents(next HandlerNotifyReturnEvents) HandlerNotifyReturnEvents
}

// ProvidesNotifyCancelEvents provides NotifyCancelEvents as a method.
type ProvidesNotifyCancelEvents interface {
	ProvidesMiddleware
	NotifyCancelEvents(next HandlerNotifyCancelEvents) HandlerNotifyCancelEvents
}

// ProvidesNotifyFlowEvents provides NotifyCancelEvents as a method.
type ProvidesNotifyFlowEvents interface {
	ProvidesMiddleware
	NotifyFlowEvents(next HandlerNotifyFlowEvents) HandlerNotifyFlowEvents
}

// PROVIDES ALL INTERFACES ############################
// ####################################################

// ProvidesAllConnection is a convenience interface for generating middleware providers
// that implement all Connection middleware providers interfaces.
type ProvidesAllConnection interface {
	ProvidesClose
	ProvidesNotifyClose
	ProvidesNotifyDial
	ProvidesNotifyDisconnect
	ProvidesNotifyDialEvents
	ProvidesNotifyDisconnectEvents
	ProvidesNotifyCloseEvents
	ProvidesConnectionReconnect
}

// ProvidesAllChannel is a convenience interface for generating middleware providers
// that implement all Channel middleware provider interfaces.
type ProvidesAllChannel interface {
	ProvidesClose
	ProvidesNotifyClose
	ProvidesNotifyDial
	ProvidesNotifyDisconnect
	ProvidesNotifyCloseEvents
	ProvidesNotifyDialEvents
	ProvidesNotifyDisconnectEvents

	ProvidesChannelReconnect
	ProvidesQueueDeclare
	ProvidesQueueDeclarePassive
	ProvidesQueueInspect
	ProvidesQueueDelete
	ProvidesQueueBind
	ProvidesQueueUnbind
	ProvidesQueuePurge
	ProvidesExchangeDeclare
	ProvidesExchangeDeclarePassive
	ProvidesExchangeDelete
	ProvidesExchangeBind
	ProvidesExchangeUnbind
	ProvidesQoS
	ProvidesFlow
	ProvidesConfirm
	ProvidesPublish
	ProvidesGet
	ProvidesConsume
	ProvidesAck
	ProvidesNack
	ProvidesReject
	ProvidesNotifyPublish
	ProvidesNotifyConfirm
	ProvidesNotifyConfirmOrOrphaned
	ProvidesNotifyReturn
	ProvidesNotifyCancel
	ProvidesNotifyFlow

	ProvidesNotifyPublishEvents
	ProvidesConsumeEvents
	ProvidesNotifyConfirmEvents
	ProvidesNotifyConfirmOrOrphanedEvents
	ProvidesNotifyReturnEvents
	ProvidesNotifyFlowEvents
}

// ProvidesAllMiddleware is a convenience interface for generating middleware providers
// that implement all Connection and Channel middleware provider interfaces.
type ProvidesAllMiddleware interface {
	ProvidesAllConnection
	ProvidesAllChannel
}
