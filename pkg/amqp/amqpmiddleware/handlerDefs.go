package amqpmiddleware

import (
	"context"
)

// TransportType is passed into handlers who's definition is shared between
// amqp.Channel values and amqp.Connection values.
type TransportType string

// TransportTypeConnection is passed into handlers by amqp.Connection values.
const TransportTypeConnection = "CONNECTION"

// TransportTypeChannel is passed into handlers by amqp.Channel values.
const TransportTypeChannel = "CHANNEL"

// SHARED METHOD HANDLERS ##############################
// #####################################################

// HandlerClose is the signature for handlers invoked when the Close method is called on
// an amqp.Connection or amqp.Channel.
type HandlerClose func(ctx context.Context, args ArgsClose) error

// HandlerNotifyClose is the signature for handlers invoked when the NotifyClose method
// is called on an amqp.Connection or amqp.Channel.
type HandlerNotifyClose func(ctx context.Context, args ArgsNotifyClose) ResultsNotifyClose

// HandlerNotifyDial is the signature for handlers invoked when the NotifyDial method
// is called on an amqp.Connection or amqp.Channel.
type HandlerNotifyDial func(ctx context.Context, args ArgsNotifyDial) error

// HandlerNotifyDisconnect is the signature for handlers invoked when the
// NotifyDisconnect method is called on an amqp.Connection or amqp.Channel.
type HandlerNotifyDisconnect func(ctx context.Context, args ArgsNotifyDisconnect) error

// SHARED EVENT HANDLERS ###############################
// #####################################################

// HandlerNotifyDialEvents is the signature for handlers invoked when an event from a
// NotifyDial of an amqp.Connection or amqp.Channel is being processed before
// being sent to the caller.
type HandlerNotifyDialEvents func(metadata EventMetadata, event EventNotifyDial)

// HandlerNotifyDisconnectEvents is the signature for handlers invoked when an event
// from a NotifyDisconnect of an amqp.Connection or amqp.Channel is being processed
// before being sent to the caller.
type HandlerNotifyDisconnectEvents func(metadata EventMetadata, event EventNotifyDisconnect)

// HandlerNotifyCloseEvents is the signature for handlers invoked when an event from a
// NotifyClose of an amqp.Connection or amqp.Channel is being processed before
// being sent to the caller.
type HandlerNotifyCloseEvents func(metadata EventMetadata, event EventNotifyClose)

// CONNECTION METHOD HANDLERS ##########################
// #####################################################

// HandlerConnectionReconnect is the signature for handlers triggered when a channel is
// being re-established.
//
// Attempt is the attempt number, including all previous failures and successes.
type HandlerConnectionReconnect = func(
	ctx context.Context, args ArgsConnectionReconnect,
) (ResultsConnectionReconnect, error)

// CHANNEL METHOD HANDLERS #############################
// #####################################################

// HandlerChannelReconnect is the signature for handlers triggered when a channel is
// being re-established.
//
// Attempt is the attempt number, including all previous failures and successes.
type HandlerChannelReconnect = func(ctx context.Context, args ArgsChannelReconnect) (ResultsChannelReconnect, error)

// HandlerQueueDeclare is the signature for handlers invoked when
// amqp.Channel.QueueDeclare is called.
type HandlerQueueDeclare = func(ctx context.Context, args ArgsQueueDeclare) (ResultsQueueDeclare, error)

// HandlerQueueInspect is the signature for handlers invoked when
// amqp.Channel.QueueDeclare is called.
type HandlerQueueInspect = func(ctx context.Context, args ArgsQueueInspect) (ResultsQueueInspect, error)

// HandlerQueueDelete is the signature for handlers invoked when
// amqp.Channel.QueueDelete is called.
type HandlerQueueDelete = func(ctx context.Context, args ArgsQueueDelete) (ResultsQueueDelete, error)

// HandlerQueueBind is the signature for handlers invoked when amqp.Channel.QueueBind
// is called.
type HandlerQueueBind = func(ctx context.Context, args ArgsQueueBind) error

// HandlerQueueUnbind is the signature for handlers invoked when
// amqp.Channel.QueueUnbind is called.
type HandlerQueueUnbind = func(ctx context.Context, args ArgsQueueUnbind) error

// HandlerQueuePurge is the signature for handlers invoked when amqp.Channel.QueuePurge
// is called.
type HandlerQueuePurge = func(ctx context.Context, args ArgsQueuePurge) (ResultsQueuePurge, error)

// HandlerExchangeDeclare is the signature for handlers invoked when
// amqp.Channel.ExchangeDeclare is called.
type HandlerExchangeDeclare = func(ctx context.Context, args ArgsExchangeDeclare) error

// HandlerExchangeDelete is the signature for handlers invoked when
// amqp.Channel.ExchangeDelete is called.
type HandlerExchangeDelete func(ctx context.Context, args ArgsExchangeDelete) error

// HandlerExchangeBind is the signature for handlers invoked when
// amqp.Channel.ExchangeBind is called.
type HandlerExchangeBind func(ctx context.Context, args ArgsExchangeBind) error

// HandlerExchangeUnbind is the signature for handlers invoked when
// amqp.Channel.ExchangeUnbind is called.
type HandlerExchangeUnbind func(ctx context.Context, args ArgsExchangeUnbind) error

// HandlerQoS is the signature for handlers invoked when amqp.Channel.QoS is called.
type HandlerQoS func(ctx context.Context, args ArgsQoS) error

// HandlerFlow is the signature for handlers invoked when amqp.Channel.Flow is called.
type HandlerFlow func(ctx context.Context, args ArgsFlow) error

// HandlerConfirm is the signature for handlers invoked when amqp.Channel.Confirm is
// called.
type HandlerConfirm func(ctx context.Context, args ArgsConfirms) error

// HandlerPublish is the signature for handlers invoked when amqp.Channel.Publish is
// called.
type HandlerPublish func(ctx context.Context, args ArgsPublish) error

// HandlerGet is the signature for handlers invoked when amqp.Channel.Get is called.
type HandlerGet func(ctx context.Context, args ArgsGet) (results ResultsGet, err error)

// HandlerConsume is the signature for handlers invoked when amqp.Channel.Consume is
// called.
//
// NOTE: this is separate from HandlerConsumeEvents, which handles each event. This
// handler only fires on the initial call
type HandlerConsume func(ctx context.Context, args ArgsConsume) (results ResultsConsume, err error)

// HandlerAck is the signature for handlers invoked when amqp.Channel.Ack is called.
type HandlerAck func(ctx context.Context, args ArgsAck) error

// HandlerNack is the signature for handlers invoked when amqp.Channel.Nack is called.
type HandlerNack func(ctx context.Context, args ArgsNack) error

// HandlerReject is the signature for handlers invoked when amqp.Channel.Reject is
// called.
type HandlerReject func(ctx context.Context, args ArgsReject) error

// HandlerNotifyConfirm is the signature for handlers invoked when
// amqp.Channel.NotifyConfirm is called.
type HandlerNotifyConfirm func(ctx context.Context, args ArgsNotifyConfirm) ResultsNotifyConfirm

// HandlerNotifyConfirmOrOrphaned is the signature for handlers invoked when
// amqp.Channel.NotifyConfirmOrOrphaned is called.
type HandlerNotifyConfirmOrOrphaned func(
	ctx context.Context, args ArgsNotifyConfirmOrOrphaned,
) ResultsNotifyConfirmOrOrphaned

// HandlerNotifyReturn signature for handlers invoked when amqp.Channel.NotifyReturn
// is called.
type HandlerNotifyReturn func(ctx context.Context, args ArgsNotifyReturn) ResultsNotifyReturn

// HandlerNotifyCancel signature for handlers invoked when amqp.Channel.NotifyReturn
// is called.
type HandlerNotifyCancel func(ctx context.Context, args ArgsNotifyCancel) ResultsNotifyCancel

// HandlerNotifyFlow signature for handlers invoked when amqp.Channel.NotifyFlow is
// called.
type HandlerNotifyFlow func(ctx context.Context, args ArgsNotifyFlow) ResultsNotifyFlow

// HandlerNotifyPublish is the signature for handlers invoked when
// amqp.Channel.NotifyPublish is called.
type HandlerNotifyPublish func(ctx context.Context, args ArgsNotifyPublish) ResultsNotifyPublish

// CHANNEL EVENT HANDLERS ##############################
// #####################################################

// HandlerNotifyPublishEvents is the signature for handlers invoked when an event from
// an amqp.Channel.NotifyPublish is being processed before being sent to the caller.
type HandlerNotifyPublishEvents func(metadata EventMetadata, event EventNotifyPublish)

// HandlerConsumeEvents is the signature for handlers invoked when an event from an
// amqp.Channel.Consume is being processed before being sent to the caller.
type HandlerConsumeEvents func(metadata EventMetadata, event EventConsume)

// HandlerNotifyConfirmEvents is the signature for handlers invoked when an event from
// an amqp.Channel.NotifyConfirm is being processed before being sent to the caller.
type HandlerNotifyConfirmEvents func(metadata EventMetadata, event EventNotifyConfirm)

// HandlerNotifyConfirmOrOrphanedEvents is the signature for handlers invoked when an
// event from ab amqp.Channel.NotifyConfirmOrOrphaned is being processed before being
// sent to the caller.
type HandlerNotifyConfirmOrOrphanedEvents func(metadata EventMetadata, event EventNotifyConfirmOrOrphaned)

// HandlerNotifyReturnEvents is the signature for handlers invoked when an event from an
// amqp.Channel.NotifyReturn is being processed before being sent to the caller.
type HandlerNotifyReturnEvents func(metadata EventMetadata, event EventNotifyReturn)

// HandlerNotifyCancelEvents is the signature for handlers invoked when an event from an
// amqp.Channel.NotifyReturn is being processed before being sent to the caller.
type HandlerNotifyCancelEvents func(metadata EventMetadata, event EventNotifyCancel)

// HandlerNotifyFlowEvents is the signature for handlers invoked when an event from an
// amqp.Channel.NotifyFlow is being processed before being sent to the caller.
type HandlerNotifyFlowEvents func(metadata EventMetadata, event EventNotifyFlow)
