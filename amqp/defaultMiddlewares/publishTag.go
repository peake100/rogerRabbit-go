package defaultMiddlewares

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpMiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
	"sync/atomic"
)

// The publish tags middleware keeps track of client-facing and internal Publishing
// DeliveryTags and applies the correct offset so tags are continuous, even over
// re-connections.
type PublishTagsMiddleware struct {
	// Whether confirmation mode is on.
	confirmMode bool

	// The current delivery tag for this robust connection. Each time a message is
	// successfully published, this value should be atomically incremented. This tag
	// will function like the normal channel tag, AND WILL NOT RESET when the underlying
	// channel is re-established. Whenever we reconnectMiddleware, the broker will reset
	// and begin delivery tags at 1. That means that we are going to need to track how
	// the current underlying channel's delivery tag matches up against our user-facing
	// tags.
	//
	// The goal is to simulate the normal channel's behavior and continue to send an
	// unbroken stream of incrementing delivery tags, even during multiple connection
	// interruptions.
	//
	// We use a pointer here to support atomic operations.
	publishCount *uint64
	// Offset to add to a given tag to get it's actual broker delivery tag for the
	// current channel.
	tagOffset uint64

	// List of functions to send outstanding orphans
	sendOrphans []func()
}

func (middleware *PublishTagsMiddleware) PublishCount() uint64 {
	return *middleware.publishCount
}

func (middleware *PublishTagsMiddleware) TagOffset() uint64 {
	return middleware.tagOffset
}

func (middleware *PublishTagsMiddleware) reconnectSendOrphans() {
	// Send any orphans we are waiting on.
	sendsDone := new(sync.WaitGroup)
	sendsDone.Add(len(middleware.sendOrphans))

	for _, thisSend := range middleware.sendOrphans {
		sendRoutine := func() {
			defer sendsDone.Done()
			thisSend()
		}
		go sendRoutine()
	}

	sendsDone.Wait()
}

func (middleware *PublishTagsMiddleware) Reconnect(
	next amqpMiddleware.HandlerReconnect,
) (handler amqpMiddleware.HandlerReconnect) {
	handler = func(
		ctx context.Context, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		// The current count becomes the offset we apply to tags on this channel.
		middleware.tagOffset = *middleware.publishCount

		sendDone := new(sync.WaitGroup)
		sendDone.Add(1)
		go func() {
			defer sendDone.Done()
			middleware.reconnectSendOrphans()
		}()

		// While those are cooking , we can move forward with getting the channel.
		channel, err := next(ctx, logger)
		// Once the channel returns, wait for all our orphan notifications to be sent
		// out.
		sendDone.Wait()

		// Return the results
		return channel, err
	}

	return handler
}

func (middleware *PublishTagsMiddleware) Confirm(
	next amqpMiddleware.HandlerConfirm,
) (handler amqpMiddleware.HandlerConfirm) {
	handler = func(args *amqpMiddleware.ArgsConfirms) error {
		err := next(args)
		if err != nil {
			return err
		}

		middleware.confirmMode = true
		return nil
	}

	return handler
}

func (middleware *PublishTagsMiddleware) Publish(
	next amqpMiddleware.HandlerPublish,
) (handler amqpMiddleware.HandlerPublish) {
	handler = func(args *amqpMiddleware.ArgsPublish) error {
		err := next(args)
		if err != nil || !middleware.confirmMode {
			return err
		}

		// If there was no error, and we are in confirms mode, increment the current
		// delivery tag. We need to do this atomically so if publish is getting called
		// in more than one goroutine, we don't have a data race condition and
		// under-publishCount our tags.
		atomic.AddUint64(middleware.publishCount, 1)
		return nil
	}

	return handler
}

func (middleware *PublishTagsMiddleware) notifyPublishEventOrphans(
	next amqpMiddleware.HandlerNotifyPublishEvent,
	sentCount uint64,
) uint64 {
	// The goal of this library is to simulate the behavior of streadway/amqp. Since
	// the streadway lib guarantees that all confirms will be in an ascending, ordered,
	// unbroken stream, we need to handle a case where a channel was terminated before
	// all deliveries were acknowledged, and continuing to send confirmations would
	// result in a DeliveryTag gap.
	//
	// It's possible that when the last connection went down, we missed some
	// confirmations. We are going to check that the offset matches the number we
	// have sent so far and, if not, nack the difference. We are only going to do this
	// on re-connections to better mock the behavior of the original lib, where if the
	// channel is forcibly closed, the final messages will not be confirmed.
	for sentCount < middleware.tagOffset {
		confirmation := data.Confirmation{
			Confirmation: streadway.Confirmation{
				DeliveryTag: sentCount + 1,
				Ack:         false,
			},
			DisconnectOrphan: true,
		}
		next(&amqpMiddleware.EventNotifyPublish{Confirmation: confirmation})
		sentCount++
	}

	return sentCount
}

func (middleware *PublishTagsMiddleware) NotifyPublishEvent(
	next amqpMiddleware.HandlerNotifyPublishEvent,
) (handler amqpMiddleware.HandlerNotifyPublishEvent) {
	// We need to know the total number of confirmation that have been sent. We can
	// start with the current tag offset.
	sent := middleware.tagOffset
	first := true

	sendOrphans := func() {
		sent = middleware.notifyPublishEventOrphans(next, sent)
	}
	middleware.sendOrphans = append(middleware.sendOrphans, sendOrphans)

	return func(event *amqpMiddleware.EventNotifyPublish) {
		// If this is the first ever delivery we have received, update sent to
		// be equal to it's current value + this delivery tag - 1.
		//
		// This will get our sent count in line with the current number of sent
		// notifications.
		if first {
			sent += event.Confirmation.DeliveryTag - 1
			first = false
		}

		event.Confirmation.DeliveryTag += middleware.tagOffset
		next(event)
		sent++
	}
}


func NewPublishTagsMiddleware() *PublishTagsMiddleware {
	count := uint64(0)
	return &PublishTagsMiddleware{
		confirmMode:  false,
		publishCount: &count,
		tagOffset:    0,
	}
}
