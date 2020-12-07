package defaultMiddlewares

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync/atomic"
)

type DeliveryTagsMiddleware struct {
	// As tagPublishCount, but for delivery tags of delivered messages.
	tagConsumeCount *uint64
	// As tagConsumeCount, but for consumption tags.
	tagConsumeOffset uint64
	// The highest ack we have received
	tagLatestDeliveryAck uint64
}

func (middleware *DeliveryTagsMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	handler = func(
		ctx context.Context, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		middleware.tagConsumeOffset = *middleware.tagConsumeCount

		channel, err := next(ctx, logger)
		if err != nil {
			return channel, err

		}

		return channel, err
	}

	return handler
}


func (middleware *DeliveryTagsMiddleware) Get(
	next amqpMiddleware.HandlerGet,
) (handler amqpMiddleware.HandlerGet) {
	handler = func(
		args *amqpMiddleware.ArgsGet,
	) (msg data.Delivery, ok bool, err error) {
		msg, ok, err = next(args)
		if err != nil {
			return msg, ok, err
		}

		// Apply the offset if there was not an error
		msg.TagOffset = middleware.tagConsumeOffset
		msg.DeliveryTag += middleware.tagConsumeOffset

		atomic.AddUint64(middleware.tagConsumeCount, 1)

		return msg, ok, err
	}

	return handler
}

func (middleware *DeliveryTagsMiddleware) Ack(
	next amqpMiddleware.HandlerAck,
) (handler amqpMiddleware.HandlerAck) {
	handler = func(args *amqpMiddleware.ArgsAck) error {
		return next(args)
	}

	return handler
}

func (middleware *DeliveryTagsMiddleware) Nack(
	next amqpMiddleware.HandlerNack,
) (handler amqpMiddleware.HandlerNack) {
	handler = func(args *amqpMiddleware.ArgsNack) error {
		return next(args)
	}

	return handler
}

func (middleware *DeliveryTagsMiddleware) Nack(
	next amqpMiddleware.HandlerNack,
) (handler amqpMiddleware.HandlerNack) {
	handler = func(args *amqpMiddleware.ArgsNack) error {
		return next(args)
	}

	return handler
}

func (middleware *DeliveryTagsMiddleware) ConsumeEvent(
	next amqpMiddleware.HandlerConsumeEvent,
) (handler amqpMiddleware.HandlerConsumeEvent) {
	handler = func(event data.Delivery) {
		event.TagOffset = middleware.tagConsumeOffset
		event.DeliveryTag += middleware.tagConsumeOffset

		atomic.AddUint64(middleware.tagConsumeCount, 1)

		next(event)
	}

	return handler
}

func NewDeliveryTagsMiddleware() *DeliveryTagsMiddleware {
	tagConsumeCount := uint64(0)

	return &DeliveryTagsMiddleware{
		tagConsumeCount:      &tagConsumeCount,
		tagConsumeOffset:     0,
		tagLatestDeliveryAck: 0,
	}
}