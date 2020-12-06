package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Builds the base method handlers for a given robust connection + channel
type middlewareBaseBuilder struct {
	connection  *Connection
	currentChan *streadway.Channel
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
		builder.currentChan = channel
		return channel, nil
	}

	return handler
}

func (builder *middlewareBaseBuilder) createBaseHandlerQueueDeclare() (
	handler amqpMiddleware.HandlerQueueDeclare,
) {
	handler = func(args *amqpMiddleware.ArgsQueueDeclare) (Queue, error) {
		return builder.currentChan.QueueDeclare(
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
		return builder.currentChan.QueueDelete(
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
		return builder.currentChan.QueueBind(
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
		return builder.currentChan.QueueUnbind(
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
		return builder.currentChan.ExchangeDeclare(
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
		return builder.currentChan.ExchangeDelete(
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
		return builder.currentChan.ExchangeBind(
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
		return builder.currentChan.ExchangeUnbind(
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
		return builder.currentChan.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
	}

	return handler
}
