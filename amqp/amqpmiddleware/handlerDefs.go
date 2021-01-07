package amqpmiddleware

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// HOOK DEFINITIONS

// HandlerReconnect: signature for handlers triggered when a channel is being
// re-established.
type HandlerReconnect = func(
	ctx context.Context, logger zerolog.Logger,
) (*streadway.Channel, error)

// HandlerQueueDeclare: signature for handlers invoked when amqp.Channel.QueueDeclare()
// is called.
type HandlerQueueDeclare = func(args *ArgsQueueDeclare) (streadway.Queue, error)

// HandlerQueueDelete: signature for handlers invoked when amqp.Channel.QueueDelete()
// is called.
type HandlerQueueDelete = func(args *ArgsQueueDelete) (count int, err error)

// HandlerQueueBind: signature for handlers invoked when amqp.Channel.QueueBind()
// is called.
type HandlerQueueBind = func(args *ArgsQueueBind) error

// HandlerQueueUnbind: signature for handlers invoked when amqp.Channel.QueueUnbind()
// is called.
type HandlerQueueUnbind = func(args *ArgsQueueUnbind) error

// HandlerExchangeDeclare: signature for handlers invoked when
// amqp.Channel.ExchangeDeclare() is called.
type HandlerExchangeDeclare = func(args *ArgsExchangeDeclare) error

// HandlerExchangeDelete: signature for handlers invoked when
// amqp.Channel.ExchangeDelete() is called.
type HandlerExchangeDelete func(args *ArgsExchangeDelete) error

// HandlerExchangeBind: signature for handlers invoked when
// amqp.Channel.ExchangeBind() is called.
type HandlerExchangeBind func(args *ArgsExchangeBind) error

// HandlerExchangeUnbind: signature for handlers invoked when
// amqp.Channel.ExchangeUnbind() is called.
type HandlerExchangeUnbind func(args *ArgsExchangeUnbind) error

// HandlerQoS: signature for handlers invoked when amqp.Channel.QoS() is called.
type HandlerQoS func(args *ArgsQoS) error

// HandlerFlow: signature for handlers invoked when amqp.Channel.Flow() is called.
type HandlerFlow func(args *ArgsFlow) error

// HandlerConfirm: signature for handlers invoked when amqp.Channel.Confirm() is called.
type HandlerConfirm func(args *ArgsConfirms) error

// HandlerPublish: signature for handlers invoked when amqp.Channel.Publish() is called.
type HandlerPublish func(args *ArgsPublish) error

// HandlerGet: signature for handlers invoked when amqp.Channel.Get() is called.
type HandlerGet func(args *ArgsGet) (msg datamodels.Delivery, ok bool, err error)

// HandlerAck: signature for handlers invoked when amqp.Channel.Ack() is called.
type HandlerAck func(args *ArgsAck) error

// HandlerNack: signature for handlers invoked when amqp.Channel.Nack() is called.
type HandlerNack func(args *ArgsNack) error

// HandlerReject: signature for handlers invoked when amqp.Channel.Reject() is called.
type HandlerReject func(args *ArgsReject) error

// HandlerNotifyPublish: signature for handlers invoked when
// amqp.Channel.NotifyPublish() is called.
type HandlerNotifyPublish func(args *ArgsNotifyPublish) chan datamodels.Confirmation

// HandlerNotifyPublishEvent: signature for handlers invoked when an event from a
// amqp.Channel.NotifyPublish() is being processed before beings sent to the caller.
type HandlerNotifyPublishEvent func(event *EventNotifyPublish)

// HandlerConsumeEvent: signature for handlers invoked when an event from a
// amqp.Channel.Consume() is being processed before beings sent to the caller.
type HandlerConsumeEvent func(event *EventConsume)
