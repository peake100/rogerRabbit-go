package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Builds the base method handlers for a given robust connection + channel
type middlewareBaseBuilder struct {
	connection     *Connection
	channel        *Channel
	underlyingChan *streadway.Channel
}

func (builder *middlewareBaseBuilder) createBaseHandlerReconnect(
	connection *Connection,
) (handler amqpMiddleware.HandlerReconnect) {
	handler = func(
		ctx context.Context, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := connection.getStreadwayChannel(ctx)
		if err != nil {
			return nil, err
		}
		builder.underlyingChan = channel
		return channel, nil
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQueueDeclare() (
	handler amqpMiddleware.HandlerQueueDeclare,
) {
	handler = func(args *amqpMiddleware.ArgsQueueDeclare) (Queue, error) {
		return builder.underlyingChan.QueueDeclare(
			args.Name,
			args.Durable,
			args.AutoDelete,
			args.Exclusive,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQueueDelete() (
	handler amqpMiddleware.HandlerQueueDelete,
) {
	handler = func(args *amqpMiddleware.ArgsQueueDelete) (int, error) {
		return builder.underlyingChan.QueueDelete(
			args.Name,
			args.IfUnused,
			args.IfEmpty,
			args.NoWait,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQueueBind() (
	handler amqpMiddleware.HandlerQueueBind,
) {
	handler = func(args *amqpMiddleware.ArgsQueueBind) error {
		return builder.underlyingChan.QueueBind(
			args.Name,
			args.Key,
			args.Exchange,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQueueUnbind() (
	handler amqpMiddleware.HandlerQueueUnbind,
) {
	handler = func(args *amqpMiddleware.ArgsQueueUnbind) error {
		return builder.underlyingChan.QueueUnbind(
			args.Name,
			args.Key,
			args.Exchange,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDeclare() (
	handler amqpMiddleware.HandlerExchangeDeclare,
) {
	handler = func(args *amqpMiddleware.ArgsExchangeDeclare) error {
		return builder.underlyingChan.ExchangeDeclare(
			args.Name,
			args.Kind,
			args.Durable,
			args.AutoDelete,
			args.Internal,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDelete() (
	handler amqpMiddleware.HandlerExchangeDelete,
) {
	handler = func(args *amqpMiddleware.ArgsExchangeDelete) error {
		return builder.underlyingChan.ExchangeDelete(
			args.Name,
			args.IfUnused,
			args.NoWait,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerExchangeBind() (
	handler amqpMiddleware.HandlerExchangeBind,
) {
	handler = func(args *amqpMiddleware.ArgsExchangeBind) error {
		return builder.underlyingChan.ExchangeBind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerExchangeUnbind() (
	handler amqpMiddleware.HandlerExchangeUnbind,
) {
	handler = func(args *amqpMiddleware.ArgsExchangeUnbind) error {
		return builder.underlyingChan.ExchangeUnbind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQoS() (
	handler amqpMiddleware.HandlerQoS,
) {
	handler = func(args *amqpMiddleware.ArgsQoS) error {
		return builder.underlyingChan.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerConfirm() (
	handler amqpMiddleware.HandlerConfirm,
) {
	handler = func(args *amqpMiddleware.ArgsConfirms) error {
		return builder.underlyingChan.Confirm(
			args.NoWait,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerPublish() (
	handler amqpMiddleware.HandlerPublish,
) {
	handler = func(args *amqpMiddleware.ArgsPublish) error {
		return builder.underlyingChan.Publish(
			args.Exchange,
			args.Key,
			args.Mandatory,
			args.Immediate,
			args.Msg,
		)
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerGet() (
	handler amqpMiddleware.HandlerGet,
) {
	handler = func(
		args *amqpMiddleware.ArgsGet,
	) (msg data.Delivery, ok bool, err error) {
		var msgOrig streadway.Delivery
		msgOrig, ok, err = builder.underlyingChan.Get(args.Queue, args.AutoAck)
		msg = data.NewDelivery(msgOrig, 0, builder.channel)
		return msg, ok, err
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerNotifyPublish() (
	handler amqpMiddleware.HandlerNotifyPublish,
) {
	handler = func(args *amqpMiddleware.ArgsNotifyPublish) chan data.Confirmation {
		channel := builder.channel

		relay := newNotifyPublishRelay(
			args.Confirm,
			channel.transportChannel.handlers.notifyPublishEventMiddleware,
		)

		err := channel.setupAndLaunchEventRelay(relay)
		// On an error, close the channel.
		if err != nil {
			channel.logger.Error().
				Err(err).
				Msg("error setting up NotifyPublish event relay")
			close(args.Confirm)
		}
		return args.Confirm
	}

	return handler
}
