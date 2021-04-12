package amqpmiddleware

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// TransportType is passed into handlers who's definition is shared between
// amqp.Channel values and amqp.Connection values.
type TransportType string

// TransportTypeConnection is passed into handlers by amqp.Connection values.
const TransportTypeConnection = "CONNECTION"

// TransportTypeChannel is passed into handlers by amqp.Channel values.
const TransportTypeChannel = "CHANNEL"

// HOOK DEFINITIONS

// HandlerReconnect: signature for handlers triggered when a channel is being
// re-established.
//
// Attempt is the attempt number, including all previous failures and successes.
type HandlerReconnect = func(
	ctx context.Context,
	transportType TransportType,
	attempt uint64,
	logger zerolog.Logger,
) (*streadway.Channel, error)

// HandlerQueueDeclare: signature for handlers invoked when amqp.Channel.QueueDeclare()
// is called.
type HandlerQueueDeclare = func(args ArgsQueueDeclare) (streadway.Queue, error)

// HandlerQueueInspect: signature for handlers invoked when amqp.Channel.QueueDeclare()
// is called.
type HandlerQueueInspect = func(args ArgsQueueInspect) (streadway.Queue, error)

// HandlerQueueDelete: signature for handlers invoked when amqp.Channel.QueueDelete()
// is called.
type HandlerQueueDelete = func(args ArgsQueueDelete) (count int, err error)

// HandlerQueueBind: signature for handlers invoked when amqp.Channel.QueueBind()
// is called.
type HandlerQueueBind = func(args ArgsQueueBind) error

// HandlerQueueUnbind: signature for handlers invoked when amqp.Channel.QueueUnbind()
// is called.
type HandlerQueueUnbind = func(args ArgsQueueUnbind) error

// HandlerQueuePurge: signature for handlers invoked when amqp.Channel.QueuePurge()
// is called.
type HandlerQueuePurge = func(args ArgsQueuePurge) (int, error)

// HandlerExchangeDeclare: signature for handlers invoked when
// amqp.Channel.ExchangeDeclare() is called.
type HandlerExchangeDeclare = func(args ArgsExchangeDeclare) error

// HandlerExchangeDelete: signature for handlers invoked when
// amqp.Channel.ExchangeDelete() is called.
type HandlerExchangeDelete func(args ArgsExchangeDelete) error

// HandlerExchangeBind: signature for handlers invoked when
// amqp.Channel.ExchangeBind() is called.
type HandlerExchangeBind func(args ArgsExchangeBind) error

// HandlerExchangeUnbind: signature for handlers invoked when
// amqp.Channel.ExchangeUnbind() is called.
type HandlerExchangeUnbind func(args ArgsExchangeUnbind) error

// HandlerQoS: signature for handlers invoked when amqp.Channel.QoS() is called.
type HandlerQoS func(args ArgsQoS) error

// HandlerFlow: signature for handlers invoked when amqp.Channel.Flow() is called.
type HandlerFlow func(args ArgsFlow) error

// HandlerConfirm: signature for handlers invoked when amqp.Channel.Confirm() is called.
type HandlerConfirm func(args ArgsConfirms) error

// HandlerPublish: signature for handlers invoked when amqp.Channel.Publish() is called.
type HandlerPublish func(args ArgsPublish) error

// HandlerGet: signature for handlers invoked when amqp.Channel.Get() is called.
type HandlerGet func(args ArgsGet) (msg datamodels.Delivery, ok bool, err error)

// HandlerConsume: signature for handlers invoked when amqp.Channel.Consume() is called.
//
// NOTE: this is separate from HandlerConsumeEvents, which handles each event. This
// handler only fires on the initial call
type HandlerConsume func(args ArgsConsume) (
	deliveryChan <-chan datamodels.Delivery, err error,
)

// HandlerAck: signature for handlers invoked when amqp.Channel.Ack() is called.
type HandlerAck func(args ArgsAck) error

// HandlerNack: signature for handlers invoked when amqp.Channel.Nack() is called.
type HandlerNack func(args ArgsNack) error

// HandlerReject: signature for handlers invoked when amqp.Channel.Reject() is called.
type HandlerReject func(args ArgsReject) error

// HandlerNotifyConfirm: signature for handlers invoked when
// amqp.Channel.NotifyConfirm() is called.
type HandlerNotifyConfirm func(args ArgsNotifyConfirm) (chan uint64, chan uint64)

// HandlerNotifyConfirmOrOrphaned: signature for handlers invoked when
// amqp.Channel.NotifyConfirmOrOrphaned() is called.
type HandlerNotifyConfirmOrOrphaned func(args ArgsNotifyConfirmOrOrphaned) (
	chan uint64, chan uint64, chan uint64,
)

// HandlerNotifyReturn signature for handlers invoked when amqp.Channel.NotifyReturn()
// is called.
type HandlerNotifyReturn func(args ArgsNotifyReturn) chan streadway.Return

// HandlerNotifyCancel signature for handlers invoked when amqp.Channel.NotifyReturn()
// is called.
type HandlerNotifyCancel func(args ArgsNotifyCancel) chan string

// HandlerNotifyFlow signature for handlers invoked when amqp.Channel.NotifyFlow() is
// called.
type HandlerNotifyFlow func(args ArgsNotifyFlow) chan bool

// HandlerNotifyPublish: signature for handlers invoked when
// amqp.Channel.NotifyPublish() is called.
type HandlerNotifyPublish func(args ArgsNotifyPublish) chan datamodels.Confirmation

// HandlerNotifyPublishEvents: signature for handlers invoked when an event from an
// amqp.Channel.NotifyPublish() is being processed before beings sent to the caller.
type HandlerNotifyPublishEvents func(event EventNotifyPublish)

// HandlerConsumeEvents: signature for handlers invoked when an event from an
// amqp.Channel.Consume() is being processed before beings sent to the caller.
type HandlerConsumeEvents func(event EventConsume)

// HandlerNotifyConfirmEvents: signature for handlers invoked when an event from an
// amqp.Channel.NotifyConfirm() is being processed before beings sent to the caller.
type HandlerNotifyConfirmEvents func(event EventNotifyConfirm)

// HandlerNotifyConfirmOrOrphanedEvents: signature for handlers invoked when an event
// from ab amqp.Channel.NotifyConfirmOrOrphaned() is being processed before beings sent
// to the caller.
type HandlerNotifyConfirmOrOrphanedEvents func(event EventNotifyConfirmOrOrphaned)

// HandlerNotifyReturnEvents: signature for handlers invoked when an event from an
// amqp.Channel.NotifyReturn() is being processed before beings sent to the caller.
type HandlerNotifyReturnEvents func(event EventNotifyReturn)

// HandlerNotifyCancelEvents: signature for handlers invoked when an event from an
// amqp.Channel.NotifyReturn() is being processed before beings sent to the caller.
type HandlerNotifyCancelEvents func(event EventNotifyCancel)

// HandlerNotifyFlowEvents: signature for handlers invoked when an event from an
// amqp.Channel.NotifyFlow() is being processed before beings sent to the caller.
type HandlerNotifyFlowEvents func(event EventNotifyFlow)
