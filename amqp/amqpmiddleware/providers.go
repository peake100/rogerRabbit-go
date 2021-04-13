package amqpmiddleware

// SHARED PROVIDERS ##################
// ###################################

// ProvidesClose provides Close as a method.
type ProvidesClose interface {
	Close(next HandlerClose) HandlerClose
}

// ProvidesNotifyClose provides NotifyClose as a method.
type ProvidesNotifyClose interface {
	NotifyClose(next HandlerNotifyClose) HandlerNotifyClose
}

// ProvidesNotifyDial provides NotifyDial as a method.
type ProvidesNotifyDial interface {
	NotifyDial(next HandlerNotifyDial) HandlerNotifyDial
}

// ProvidesNotifyDisconnect provides NotifyDisconnect as a method.
type ProvidesNotifyDisconnect interface {
	NotifyDisconnect(next HandlerNotifyDisconnect) HandlerNotifyDisconnect
}

// SHARED EVENT PROVIDERS ###############################
// ######################################################

// ProvidesNotifyDialEvents provides NotifyDialEvents as a method.
type ProvidesNotifyDialEvents interface {
	NotifyDialEvents(next HandlerNotifyDialEvents) HandlerNotifyDialEvents
}

// ProvidesNotifyDisconnectEvents provides NotifyDisconnectEvents as a method.
type ProvidesNotifyDisconnectEvents interface {
	NotifyDisconnectEvents(next HandlerNotifyDisconnectEvents) HandlerNotifyDisconnectEvents
}

// ProvidesNotifyCloseEvents provides NotifyCloseEvents as a method.
type ProvidesNotifyCloseEvents interface {
	NotifyCloseEvents(next HandlerNotifyCloseEvents) HandlerNotifyCloseEvents
}

// CONNECTION PROVIDERS ################################
// #####################################################

// ProvidesConnectionReconnect provides ConnectionReconnect as a method.
type ProvidesConnectionReconnect interface {
	ConnectionReconnect(next HandlerConnectionReconnect) HandlerConnectionReconnect
}

// CHANNEL PROVIDERS ##################################
// ####################################################

// ProvidesChannelReconnect provides ChannelReconnect as a method.
type ProvidesChannelReconnect interface {
	ChannelReconnect(next HandlerChannelReconnect) HandlerChannelReconnect
}

// ProvidesQueueDeclare provides QueueDeclare as a method.
type ProvidesQueueDeclare interface {
	QueueDeclare(next HandlerQueueDeclare) HandlerQueueDeclare
}

// ProvidesQueueDeclarePassive provides QueueDeclare as a method to be used for
// amqp.Channel.QueueDeclarePassive handlers.
type ProvidesQueueDeclarePassive interface {
	QueueDeclarePassive(next HandlerQueueDeclare) HandlerQueueDeclare
}

// ProvidesQueueInspect provides QueueInspect as a method.
type ProvidesQueueInspect interface {
	QueueInspect(next HandlerQueueInspect) HandlerQueueInspect
}

// ProvidesQueueDelete provides QueueDelete as a method.
type ProvidesQueueDelete interface {
	QueueDelete(next HandlerQueueDelete) HandlerQueueDelete
}

// ProvidesQueueBind provides QueueBind as a method.
type ProvidesQueueBind interface {
	QueueBind(next HandlerQueueBind) HandlerQueueBind
}

// ProvidesQueueUnbind provides QueueUnbind as a method.
type ProvidesQueueUnbind interface {
	QueueUnbind(next HandlerQueueUnbind) HandlerQueueUnbind
}

// ProvidesQueuePurge provides QueuePurge as a method.
type ProvidesQueuePurge interface {
	QueuePurge(next HandlerQueuePurge) HandlerQueuePurge
}

// ProvidesExchangeDeclare provides ExchangeDeclare as a method.
type ProvidesExchangeDeclare interface {
	ExchangeDeclare(next HandlerExchangeDeclare) HandlerExchangeDeclare
}

// ProvidesExchangeDeclare provides ExchangeDeclare as a method.
type ProvidesExchangeDeclarePassive interface {
	ExchangeDeclarePassive(next HandlerExchangeDeclare) HandlerExchangeDeclare
}

// ProvidesExchangeDelete provides ExchangeDelete as a method.
type ProvidesExchangeDelete interface {
	ExchangeDelete(next HandlerExchangeDelete) HandlerExchangeDelete
}

// ProvidesExchangeBind provides ExchangeBind as a method.
type ProvidesExchangeBind interface {
	ExchangeBind(next HandlerExchangeBind) HandlerExchangeBind
}

// ProvidesExchangeUnbind provides ExchangeUnbind as a method.
type ProvidesExchangeUnbind interface {
	ExchangeUnbind(next HandlerExchangeUnbind) HandlerExchangeUnbind
}

// ProvidesQoS provides QoS as a method.
type ProvidesQoS interface {
	QoS(next HandlerQoS) HandlerQoS
}

// ProvidesFlow provides Flow as a method.
type ProvidesFlow interface {
	Flow(next HandlerFlow) HandlerFlow
}

// ProvidesConfirm provides Confirm as a method.
type ProvidesConfirm interface {
	Confirm(next HandlerConfirm) HandlerConfirm
}

// ProvidesPublish provides Publish as a method.
type ProvidesPublish interface {
	Publish(next HandlerPublish) HandlerPublish
}

// ProvidesGet provides Get as a method.
type ProvidesGet interface {
	Get(next HandlerGet) HandlerGet
}

// ProvidesConsume provides Consume as a method.
type ProvidesConsume interface {
	Consume(next HandlerConsume) HandlerConsume
}

// ProvidesAck provides Ack as a method.
type ProvidesAck interface {
	Ack(next HandlerAck) HandlerAck
}

// ProvidesNack provides Nack as a method.
type ProvidesNack interface {
	Nack(next HandlerNack) HandlerNack
}

// ProvidesReject provides Reject as a method.
type ProvidesReject interface {
	Reject(next HandlerReject) HandlerReject
}

// ProvidesNotifyPublish provides NotifyPublish as a method.
type ProvidesNotifyPublish interface {
	NotifyPublish(next HandlerNotifyPublish) HandlerNotifyPublish
}

// ProvidesNotifyConfirm provides NotifyConfirm as a method.
type ProvidesNotifyConfirm interface {
	NotifyConfirm(next HandlerNotifyConfirm) HandlerNotifyConfirm
}

// ProvidesNotifyConfirmOrOrphaned provides NotifyConfirmOrOrphaned as a method.
type ProvidesNotifyConfirmOrOrphaned interface {
	NotifyConfirmOrOrphaned(next HandlerNotifyConfirmOrOrphaned) HandlerNotifyConfirmOrOrphaned
}

// ProvidesNotifyReturn provides NotifyReturn as a method.
type ProvidesNotifyReturn interface {
	NotifyReturn(next HandlerNotifyReturn) HandlerNotifyReturn
}

// ProvidesNotifyCancel provides NotifyCancel as a method.
type ProvidesNotifyCancel interface {
	NotifyCancel(next HandlerNotifyCancel) HandlerNotifyCancel
}

// ProvidesNotifyFlow provides NotifyFlow as a method.
type ProvidesNotifyFlow interface {
	NotifyFlow(next HandlerNotifyFlow) HandlerNotifyFlow
}

// CHANNEL EVENT PROVIDERS ############################
// ####################################################

// ProvidesNotifyPublishEvents provides NotifyPublishEvents as a method.
type ProvidesNotifyPublishEvents interface {
	NotifyPublishEvents(next HandlerNotifyPublishEvents) HandlerNotifyPublishEvents
}

// ProvidesConsumeEvents provides ConsumeEvents as a method.
type ProvidesConsumeEvents interface {
	ConsumeEvents(next HandlerConsumeEvents) HandlerConsumeEvents
}

// ProvidesNotifyConfirmEvents provides NotifyConfirmEvents as a method.
type ProvidesNotifyConfirmEvents interface {
	NotifyConfirmEvents(next HandlerNotifyConfirmEvents) HandlerNotifyConfirmEvents
}

// ProvidesNotifyConfirmOrOrphanedEvents provides NotifyConfirmOrOrphanedEvents as a
// method.
type ProvidesNotifyConfirmOrOrphanedEvents interface {
	NotifyConfirmOrOrphanedEvents(next HandlerNotifyConfirmOrOrphanedEvents) HandlerNotifyConfirmOrOrphanedEvents
}

// ProvidesNotifyReturnEvents provides NotifyReturnEvents as a method.
type ProvidesNotifyReturnEvents interface {
	NotifyReturnEvents(next HandlerNotifyReturnEvents) HandlerNotifyReturnEvents
}

// ProvidesNotifyCancelEvents provides NotifyCancelEvents as a method.
type ProvidesNotifyCancelEvents interface {
	NotifyCancelEvents(next HandlerNotifyCancelEvents) HandlerNotifyCancelEvents
}

// ProvidesNotifyFlowEvents provides NotifyCancelEvents as a method.
type ProvidesNotifyFlowEvents interface {
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
