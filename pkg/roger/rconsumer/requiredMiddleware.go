package rconsumer

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/pears-go/pkg/pears"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/roger/rconsumer/middleware"
)

// DefaultRecoverDeliveryPanic is used as the innermost middleware for deliveries. It catches panics
// and converts them to pears.PanicError errors.
var recoverPanicMiddleware = func(next middleware.HandlerDelivery) middleware.HandlerDelivery {
	return func(ctx context.Context, delivery amqp.Delivery) (requeue bool, err error) {
		err = pears.CatchPanic(func() (innerErr error) {
			requeue, innerErr = next(ctx, delivery)
			return innerErr
		})
		return requeue, err
	}
}

// mewAckNackMiddleware returns a new middleware.Delivery that handles acking or nacking
// deliveries after the core handler processes them. We want to implement this as a
// middleware so that other middlewares, like logging middlewares, have access to any
// ack or nack errors.
func mewAckNackMiddleware(autoAck bool) middleware.Delivery {
	return func(next middleware.HandlerDelivery) middleware.HandlerDelivery {
		return func(ctx context.Context, delivery amqp.Delivery) (requeue bool, err error) {
			requeue, err = next(ctx, delivery)
			// If we are auto-acking, continue.
			if autoAck {
				return requeue, err
			}

			err = handleAck(delivery, requeue, err)
			return requeue, err
		}
	}
}

func handleAck(delivery amqp.Delivery, requeue bool, err error) error {
	// Otherwise ack non-error returns and nack error returns.
	if err == nil {
		ackErr := delivery.Ack(false)
		if ackErr != nil {
			err = fmt.Errorf("error acking delivery: %w", err)
		}
	} else {
		// Recovered panics will always result in a delivery being nacked.
		if requeue && errors.As(err, &pears.PanicError{}) {
			requeue = false
		}
		nackErr := delivery.Nack(false, requeue)
		if nackErr != nil {
			nackErr = fmt.Errorf("error nacking delivery: %w", err)
			// Save both errors.
			err = pears.BatchErrors{
				MatchMode: pears.BatchMatchFirst,
				Errs: []error{
					err,
					nackErr,
				},
			}
		}

	}

	return err
}
