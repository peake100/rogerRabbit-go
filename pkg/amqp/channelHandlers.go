package amqp

import (
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
)

// channelHandlers hols a Channel's method handlers with applied middleware.
type channelHandlers struct {
	transportManagerHandlers

	// LIFETIME HANDLERS
	// ----------------

	// channelReconnect is the handler invoked on a reconnection event.
	channelReconnect amqpmiddleware.HandlerChannelReconnect

	// MODE HANDLERS
	// -------------

	// qos is the handler for Channel.Qos.
	qos amqpmiddleware.HandlerQoS
	// flow is the handler for Channel.Flow.
	flow amqpmiddleware.HandlerFlow
	// confirm is the handler for Channel.Confirm.
	confirm amqpmiddleware.HandlerConfirm

	// QUEUE HANDLERS
	// --------------

	// queueDeclare is the handler for Channel.QueueDeclare.
	queueDeclare amqpmiddleware.HandlerQueueDeclare
	// queueDeclarePassive is the handler for Channel.QueueDeclare.
	queueDeclarePassive amqpmiddleware.HandlerQueueDeclare
	// queueInspect is the handler for Channel.QueueInspect.
	queueInspect amqpmiddleware.HandlerQueueInspect
	// queueDelete is the handler for Channel.QueueDelete.
	queueDelete amqpmiddleware.HandlerQueueDelete
	// queueBind is the handler for Channel.QueueBind.
	queueBind amqpmiddleware.HandlerQueueBind
	// queueUnbind is the handler for Channel.QueueUnbind.
	queueUnbind amqpmiddleware.HandlerQueueUnbind
	// queueInspect is the handler for Channel.QueueInspect.
	queuePurge amqpmiddleware.HandlerQueuePurge

	// EXCHANGE HANDLERS
	// -----------------

	// exchangeDeclare is the handler for Channel.ExchangeDeclare.
	exchangeDeclare amqpmiddleware.HandlerExchangeDeclare
	// exchangeDeclarePassive is the handler for Channel.ExchangeDeclare.
	exchangeDeclarePassive amqpmiddleware.HandlerExchangeDeclare
	// exchangeDelete is the handler for Channel.ExchangeDelete.
	exchangeDelete amqpmiddleware.HandlerExchangeDelete
	// exchangeBind is the handler for Channel.ExchangeBind.
	exchangeBind amqpmiddleware.HandlerExchangeBind
	// exchangeUnbind is the handler for Channel.ExchangeUnbind.
	exchangeUnbind amqpmiddleware.HandlerExchangeUnbind

	// NOTIFY HANDLERS
	// ---------------

	// notifyPublish is the handler for Channel.NotifyPublish.
	notifyPublish amqpmiddleware.HandlerNotifyPublish
	// consume is the handler for Channel.Consume.
	consume amqpmiddleware.HandlerConsume
	// notifyConfirm is the handler for Channel.NotifyConfirm.
	notifyConfirm amqpmiddleware.HandlerNotifyConfirm
	// notifyConfirmOrOrphaned is the handler for Channel.NotifyConfirmOrOrphaned.
	notifyConfirmOrOrphaned amqpmiddleware.HandlerNotifyConfirmOrOrphaned
	// notifyReturn is the handler for Channel.NotifyReturn.
	notifyReturn amqpmiddleware.HandlerNotifyReturn
	// notifyCancel is the handler for Channel.NotifyCancel.
	notifyCancel amqpmiddleware.HandlerNotifyCancel
	// notifyFlow is the handler for Channel.NotifyFlow.
	notifyFlow amqpmiddleware.HandlerNotifyFlow

	// MESSAGING HANDLERS
	// ------------------

	// publish is the handler for Channel.Publish.
	publish amqpmiddleware.HandlerPublish
	// get is the handler for Channel.Get.
	get amqpmiddleware.HandlerGet

	// ACK HANDLERS
	// ------------

	// ack is the handler for Channel.Ack.
	ack amqpmiddleware.HandlerAck
	// nack is the handler for Channel.Nack.
	nack amqpmiddleware.HandlerNack
	// reject is the handler for Channel.Reject.
	reject amqpmiddleware.HandlerReject

	// providers holds a map of ProviderID -> Provider Value for fetching during
	// tests.
	providers map[amqpmiddleware.ProviderTypeID]amqpmiddleware.ProvidesMiddleware
}

// newChannelHandlers created a new channelHandlers with all base handlers added.
func newChannelHandlers(
	conn *Connection,
	channel *Channel,
	transportHandlers transportManagerHandlers,
	config ChannelMiddlewares,
) channelHandlers {
	baseBuilder := channelHandlerBuilder{
		connection:  conn,
		channel:     channel,
		middlewares: config,
	}

	return channelHandlers{
		transportManagerHandlers: transportHandlers,

		channelReconnect:        baseBuilder.createChannelReconnect(),
		queueDeclare:            baseBuilder.createQueueDeclare(),
		queueDeclarePassive:     baseBuilder.createQueueDeclarePassive(),
		queueInspect:            baseBuilder.createQueueInspect(),
		queueDelete:             baseBuilder.createQueueDelete(),
		queueBind:               baseBuilder.createQueueBind(),
		queueUnbind:             baseBuilder.createQueueUnbind(),
		queuePurge:              baseBuilder.createQueuePurge(),
		exchangeDeclare:         baseBuilder.createExchangeDeclare(),
		exchangeDeclarePassive:  baseBuilder.createExchangeDeclarePassive(),
		exchangeDelete:          baseBuilder.createExchangeDelete(),
		exchangeBind:            baseBuilder.createExchangeBind(),
		exchangeUnbind:          baseBuilder.createExchangeUnbind(),
		qos:                     baseBuilder.createQoS(),
		flow:                    baseBuilder.createFlow(),
		confirm:                 baseBuilder.createConfirm(),
		publish:                 baseBuilder.createPublish(),
		get:                     baseBuilder.createGet(),
		consume:                 baseBuilder.createConsume(),
		ack:                     baseBuilder.createAck(),
		nack:                    baseBuilder.createNack(),
		reject:                  baseBuilder.createReject(),
		notifyPublish:           baseBuilder.createNotifyPublish(),
		notifyConfirm:           baseBuilder.createNotifyConfirm(),
		notifyConfirmOrOrphaned: baseBuilder.createNotifyConfirmOrOrphaned(),
		notifyReturn:            baseBuilder.createNotifyReturn(),
		notifyCancel:            baseBuilder.createNotifyCancel(),
		notifyFlow:              baseBuilder.createNotifyFlow(),

		providers: config.providers,
	}
}
