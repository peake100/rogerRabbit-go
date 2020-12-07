package amqpMiddleware

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// HOOK DEFINITIONS

// Signature for handlers triggered when a channel is being re-established.
type HandlerReconnect = func(
	ctx context.Context, logger zerolog.Logger,
) (*streadway.Channel, error)

type HandlerQueueDeclare = func(args *ArgsQueueDeclare) (streadway.Queue, error)

type HandlerQueueDelete = func(args *ArgsQueueDelete) (count int, err error)

type HandlerQueueBind = func(args *ArgsQueueBind) error

type HandlerQueueUnbind = func(args *ArgsQueueUnbind) error

type HandlerExchangeDeclare = func(args *ArgsExchangeDeclare) error

type HandlerExchangeDelete func(args *ArgsExchangeDelete) error

type HandlerExchangeBind func(args *ArgsExchangeBind) error

type HandlerExchangeUnbind func(args *ArgsExchangeUnbind) error

type HandlerQoS func(args *ArgsQoS) error

type HandlerFlow func(args *ArgsFlow) error

type HandlerConfirm func(args *ArgsConfirms) error

type HandlerPublish func(args *ArgsPublish) error

type HandlerGet func(args *ArgsGet) (msg data.Delivery, ok bool, err error)

type HandlerAck func(args *ArgsAck) error

type HandlerNack func(args *ArgsNack) error

type HandlerReject func(args *ArgsReject) error

type HandlerNotifyPublish func(args *ArgsNotifyPublish) chan data.Confirmation

type HandlerNotifyPublishEvent func(event *EventNotifyPublish)

type HandlerConsumeEvent func(event data.Delivery)
