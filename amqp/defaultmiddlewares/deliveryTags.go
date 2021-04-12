package defaultmiddlewares

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
	"sync/atomic"
)

// DeliveryTagsMiddleware creates continuous delivery tags across reconnections.
type DeliveryTagsMiddleware struct {
	// As tagPublishCount, but for delivery tags of delivered messages.
	tagConsumeCount *uint64
	// As tagConsumeCount, but for consumption tags.
	tagConsumeOffset uint64

	// The highest ack we have received
	tagLatestMultiAck uint64
	// Lock for tagLatestMultiAck and orphansResolved
	orphanCheckLock *sync.Mutex
}

// Reconnect establishes our current delivery tag offset based on how many deliveries
// have been consumed across all of our connections so far.
func (middleware *DeliveryTagsMiddleware) Reconnect(
	next amqpmiddleware.HandlerReconnect,
) (handler amqpmiddleware.HandlerReconnect) {
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

// Get applies our current delivery tag offset and increments our delivery count
// whenever amqp.Channel.Get() is called.
func (middleware *DeliveryTagsMiddleware) Get(
	next amqpmiddleware.HandlerGet,
) (handler amqpmiddleware.HandlerGet) {
	handler = func(
		args amqpmiddleware.ArgsGet,
	) (msg datamodels.Delivery, ok bool, err error) {
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

// Determines whether an ACK, NACK, etc request includes orphan tags.
func (middleware *DeliveryTagsMiddleware) containsOrphans(
	tag uint64, multiple bool,
) bool {
	// If the tag is above our current offset and not part of a multi-ack, it's not an
	// orphan
	if tag > middleware.tagConsumeOffset && !multiple {
		return false
		// If the tag is below the current offset, it is an orphan, regardless of
		// whether if is a multiple ack.
	} else if tag < middleware.tagConsumeOffset {
		return true
	}

	// If the latest delivery ack is greater than than the current offset, the
	// multi-ack includes no orphans
	if middleware.tagLatestMultiAck >= middleware.tagConsumeOffset {
		return false
	}

	// Otherwise, the ack os split between orphans and non-orphans, so it contains them
	return true
}

// Updates the latest multi-tag ack.
func (middleware *DeliveryTagsMiddleware) updateOrphanTracking(tag uint64) {
	// Set the latest multi-ack tag to this tag.
	middleware.tagLatestMultiAck = tag
}

// Returns an error if an ACK, NACK, etc request involves orphans.
func (middleware *DeliveryTagsMiddleware) resolveOrphans(
	tag uint64, multiple bool,
) error {
	// We only need to lock the orphan resources if this is a multi-ack
	if multiple {
		// Acquire the lock
		middleware.orphanCheckLock.Lock()
		defer middleware.orphanCheckLock.Unlock()
		// Update the multi-ack tag tracking
		defer middleware.updateOrphanTracking(tag)
	}

	// If our acknowledgement does not contain orphans, there is no error.
	if !middleware.containsOrphans(tag, multiple) {
		return nil
	}

	// Otherwise, there are orphans involved. Build and return an orphan error.
	return NewErrCantAcknowledgeOrphans(
		middleware.tagLatestMultiAck, tag, middleware.tagConsumeOffset, multiple,
	)
}

// Generic method for running an ACK, NACK, or REJECT request with orphan handling.
func (middleware *DeliveryTagsMiddleware) runAckMethod(
	method func() error, tag uint64, multiple bool,
) error {
	// If the tag is from this connection, we will handle it, otherwise we know right
	// off the bat it's an orphan.
	if tag > middleware.tagConsumeOffset {
		err := method()
		if err != nil {
			return err
		}
	}

	// Resolve whether the above command involved an orphan delivery, and if so, return
	// an error.
	return middleware.resolveOrphans(tag, multiple)
}

// Ack is invoked when amqp.Channel.Ack() is called, and handles converting the delivery
// tag back to the original value for the underlying channel, as well as returning
// errors on an attempt to ACK an orphan.
func (middleware *DeliveryTagsMiddleware) Ack(
	next amqpmiddleware.HandlerAck,
) (handler amqpmiddleware.HandlerAck) {
	handler = func(args amqpmiddleware.ArgsAck) error {
		method := func() error {
			return next(args)
		}
		return middleware.runAckMethod(method, args.Tag, args.Multiple)
	}

	return handler
}

// Nack is invoked when amqp.Channel.Nack() is called, and handles converting the
// delivery tag back to the original value for the underlying channel, as well as
// returning errors on an attempt to NACK an orphan.
func (middleware *DeliveryTagsMiddleware) Nack(
	next amqpmiddleware.HandlerNack,
) (handler amqpmiddleware.HandlerNack) {
	handler = func(args amqpmiddleware.ArgsNack) error {
		method := func() error {
			return next(args)
		}
		return middleware.runAckMethod(method, args.Tag, args.Multiple)
	}

	return handler
}

// Reject is invoked when amqp.Channel.Reject() is called, and handles converting the
// delivery tag back to the original value for the underlying channel, as well as
// returning errors on an attempt to NACK an orphan.
func (middleware *DeliveryTagsMiddleware) Reject(
	next amqpmiddleware.HandlerReject,
) (handler amqpmiddleware.HandlerReject) {
	handler = func(args amqpmiddleware.ArgsReject) error {
		method := func() error {
			return next(args)
		}
		return middleware.runAckMethod(method, args.Tag, false)
	}

	return handler
}

// ConsumeEvent is invoked whenever an event is sent to a caller of
// amqp.Channel.Consume(), and handles applying the delivery tag offset.
func (middleware *DeliveryTagsMiddleware) ConsumeEvent(
	next amqpmiddleware.HandlerConsumeEvents,
) (handler amqpmiddleware.HandlerConsumeEvents) {
	handler = func(event amqpmiddleware.EventConsume) {
		// Apply the offset to our delivery
		event.Delivery.TagOffset = middleware.tagConsumeOffset
		event.Delivery.DeliveryTag += middleware.tagConsumeOffset

		// Increment the counter
		atomic.AddUint64(middleware.tagConsumeCount, 1)

		next(event)
	}

	return handler
}

// NewDeliveryTagsMiddleware creates a new DeliveryTagsMiddleware for an amqp.Channel.
func NewDeliveryTagsMiddleware() *DeliveryTagsMiddleware {
	tagConsumeCount := uint64(0)

	return &DeliveryTagsMiddleware{
		tagConsumeCount:   &tagConsumeCount,
		tagConsumeOffset:  0,
		tagLatestMultiAck: 0,
		orphanCheckLock:   new(sync.Mutex),
	}
}

// ErrCantAcknowledgeOrphans is returned when an acknowledgement method
// (ack, nack, reject) cannot be completed because the original channel it was consumed
// from has been closed and replaced with a new one. When part of a multi-ack, it's
// possible that SOME tags will be orphaned and some will succeed, this error contains
// detailed information on both groups
type ErrCantAcknowledgeOrphans struct {
	// The first tag that could not be acknowledged because it's original channel
	// had been closed
	OrphanTagFirst uint64
	// The last tag that could not be acknowledged because it's original channel had
	// been closed. Inclusive. May be the same value as OrphanTagFirst if only one tag
	// was orphaned
	OrphanTagLast uint64

	// The first tag that was successfully acknowledged. Will be 0 if multiple was set
	// to false or if all tags were orphans.
	SuccessTagFirst uint64
	// The last tag that was successfully acknowledged. Will be 0 if multiple was set
	// to false or if all tags were orphans. May be the same as AckTagFirst if only one
	// tags was successfully acknowledged.
	SuccessTagLast uint64
}

// OrphanCount returns number of tags orphaned (will always be 1 or greater or there
// would be no error).
func (err ErrCantAcknowledgeOrphans) OrphanCount() uint64 {
	if err.OrphanTagFirst == 0 {
		return 0
	}
	return err.OrphanTagLast - err.OrphanTagFirst + 1
}

// SuccessCount returns the number of tags successfully acknowledged.
func (err ErrCantAcknowledgeOrphans) SuccessCount() uint64 {
	if err.SuccessTagFirst == 0 {
		return 0
	}
	return err.SuccessTagLast - err.SuccessTagFirst + 1
}

// Error implements builtins.error
func (err ErrCantAcknowledgeOrphans) Error() string {
	successDetails := ""
	if err.SuccessCount() > 0 {
		successDetails = fmt.Sprintf(
			" (%v - %v)", err.SuccessTagFirst, err.SuccessTagLast,
		)
	}

	return fmt.Sprintf(
		"%v tags orphaned (%v - %v), %v tags successfully acknowledged%v",
		err.OrphanCount(),
		err.OrphanTagFirst,
		err.OrphanTagLast,
		err.SuccessCount(),
		successDetails,
	)
}

// NewErrCantAcknowledgeOrphans creates a new error when one or more tags cannot be
// acknowledged because they have been orphaned. This method assumes that there is an
// error to report, and will always result in a non-nil error object.
func NewErrCantAcknowledgeOrphans(
	latestAck uint64,
	thisAck uint64,
	offset uint64,
	multiple bool,
) error {
	err := ErrCantAcknowledgeOrphans{}

	// If only a single tag was involved, then it is the first and last orphan tag and
	// there were no success tags.
	if !multiple {
		err.OrphanTagFirst = thisAck
		err.OrphanTagLast = thisAck
		return err
	}

	// Otherwise the orphan tags will start at the tag after the latest tag
	// acknowledgement we have handled
	err.OrphanTagFirst = latestAck + 1

	// If the tag we are acking is less than or equal to the current offset (before the)
	// range of the current channel, it is the last orphaned tag involved in this
	// operation.
	if thisAck <= offset {
		err.OrphanTagLast = thisAck
		return err
	}

	// Otherwise, we have some orphans and some successes. The last orphaned tag is
	// equal to the offset (first tag of current channel -1), the first successful tag
	// is the tag after that, and the current tag is the last successful ack.
	err.OrphanTagLast = offset
	err.SuccessTagFirst = offset + 1
	err.SuccessTagLast = thisAck
	return err
}
