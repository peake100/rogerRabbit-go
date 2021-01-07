package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// middlewareBaseBuilder builds the base method handlers for a given robust
// connection + channel
type middlewareBaseBuilder struct {
	connection     *Connection
	channel        *Channel
	underlyingChan *streadway.Channel
}

// createBaseHandlerReconnect returns the base handler invoked on a Channel
// reconnection.
func (builder *middlewareBaseBuilder) createBaseHandlerReconnect(
	connection *Connection,
) (handler amqpmiddleware.HandlerReconnect) {
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

// createBaseHandlerQueueDeclare returns the base handler for Channel.QueueDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueDeclare() (
	handler amqpmiddleware.HandlerQueueDeclare,
) {
	handler = func(args *amqpmiddleware.ArgsQueueDeclare) (Queue, error) {
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

// createBaseHandlerQueueDelete returns the base handler for Channel.QueueDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueDelete() (
	handler amqpmiddleware.HandlerQueueDelete,
) {
	handler = func(args *amqpmiddleware.ArgsQueueDelete) (int, error) {
		return builder.underlyingChan.QueueDelete(
			args.Name,
			args.IfUnused,
			args.IfEmpty,
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerQueueBind returns the base handler for Channel.QueueBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueBind() (
	handler amqpmiddleware.HandlerQueueBind,
) {
	handler = func(args *amqpmiddleware.ArgsQueueBind) error {
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

// createBaseHandlerQueueUnbind returns the base handler for Channel.QueueUnbind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueUnbind() (
	handler amqpmiddleware.HandlerQueueUnbind,
) {
	handler = func(args *amqpmiddleware.ArgsQueueUnbind) error {
		return builder.underlyingChan.QueueUnbind(
			args.Name,
			args.Key,
			args.Exchange,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerExchangeDeclare returns the base handler for Channel.ExchangeDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDeclare() (
	handler amqpmiddleware.HandlerExchangeDeclare,
) {
	handler = func(args *amqpmiddleware.ArgsExchangeDeclare) error {
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

// createBaseHandlerExchangeDelete returns the base handler for Channel.ExchangeDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDelete() (
	handler amqpmiddleware.HandlerExchangeDelete,
) {
	handler = func(args *amqpmiddleware.ArgsExchangeDelete) error {
		return builder.underlyingChan.ExchangeDelete(
			args.Name,
			args.IfUnused,
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerExchangeBind returns the base handler for Channel.ExchangeBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeBind() (
	handler amqpmiddleware.HandlerExchangeBind,
) {
	handler = func(args *amqpmiddleware.ArgsExchangeBind) error {
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

// createBaseHandlerExchangeUnbind returns the base handler for Channel.ExchangeUnbind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeUnbind() (
	handler amqpmiddleware.HandlerExchangeUnbind,
) {
	handler = func(args *amqpmiddleware.ArgsExchangeUnbind) error {
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

// createBaseHandlerQoS returns the base handler for Channel.Qos that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQoS() (
	handler amqpmiddleware.HandlerQoS,
) {
	handler = func(args *amqpmiddleware.ArgsQoS) error {
		return builder.underlyingChan.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
	}

	return handler
}

// createBaseHandlerFlow returns the base handler for Channel.Flow that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerFlow() (
	handler amqpmiddleware.HandlerFlow,
) {
	handler = func(args *amqpmiddleware.ArgsFlow) error {
		return builder.underlyingChan.Flow(args.Active)
	}

	return handler
}

// createBaseHandlerConfirm returns the base handler for Channel.Confirm that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerConfirm() (
	handler amqpmiddleware.HandlerConfirm,
) {
	handler = func(args *amqpmiddleware.ArgsConfirms) error {
		return builder.underlyingChan.Confirm(
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerPublish returns the base handler for Channel.Publish that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerPublish() (
	handler amqpmiddleware.HandlerPublish,
) {
	handler = func(args *amqpmiddleware.ArgsPublish) error {
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

// createBaseHandlerGet returns the base handler for Channel.Get that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerGet() (
	handler amqpmiddleware.HandlerGet,
) {
	handler = func(args *amqpmiddleware.ArgsGet) (msg Delivery, ok bool, err error) {
		var msgOrig streadway.Delivery
		msgOrig, ok, err = builder.underlyingChan.Get(args.Queue, args.AutoAck)
		msg = datamodels.NewDelivery(msgOrig, builder.channel)
		return msg, ok, err
	}

	return handler
}

// createBaseHandlerAck returns the base handler for Channel.Ack that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerAck() (
	handler amqpmiddleware.HandlerAck,
) {
	handler = func(args *amqpmiddleware.ArgsAck) error {
		return builder.underlyingChan.Ack(args.Tag, args.Multiple)
	}

	return handler
}

// createBaseHandlerNack returns the base handler for Channel.Nack that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNack() (
	handler amqpmiddleware.HandlerNack,
) {
	handler = func(args *amqpmiddleware.ArgsNack) error {
		return builder.underlyingChan.Nack(args.Tag, args.Multiple, args.Requeue)
	}

	return handler
}

// createBaseHandlerReject returns the base handler for Channel.Nack that invokes the
// method of the underlying streadway/amqp.Reject.
func (builder *middlewareBaseBuilder) createBaseHandlerReject() (
	handler amqpmiddleware.HandlerReject,
) {
	handler = func(args *amqpmiddleware.ArgsReject) error {
		return builder.underlyingChan.Reject(args.Tag, args.Requeue)
	}

	return handler
}

// createBaseHandlerNotifyPublish returns the base handler for Channel.NotifyPublish
// that invokes the method of the underlying streadway/amqp.Reject.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyPublish() (
	handler amqpmiddleware.HandlerNotifyPublish,
) {
	handler = func(args *amqpmiddleware.ArgsNotifyPublish) chan Confirmation {
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
