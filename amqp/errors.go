package amqp

import "fmt"

// Retuned when an acknowledgement method (ack, nack, reject) cannot be completed
// because the original channel it was consumed from has been closed and replaced with a
// new one. When part of a multi-ack, it's possible that SOME tags will be orphaned and
// some will succeed, this error contains detailed information on both groups
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

// The number of tags orphaned (will always be 1 or greater or there would be no error).
func (err *ErrCantAcknowledgeOrphans) OrphanCount() uint64 {
	if err.OrphanTagFirst == 0 {
		return 0
	}
	return err.OrphanTagLast - err.OrphanTagFirst + 1
}

// The number of tags successfully acknowledged.
func (err *ErrCantAcknowledgeOrphans) SuccessCount() uint64 {
	if err.SuccessTagFirst == 0 {
		return 0
	}
	return err.SuccessTagLast - err.SuccessTagFirst + 1
}

// Implements builtins.error
func (err *ErrCantAcknowledgeOrphans) Error() string {
	if err == nil {
		return ""
	}

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

// Make a new error when one or more tags cannot be acknowledged because they have been
// orphaned. This method assumes that there is an error to report, and will always
// result in a non-nil error object.
func newErrCantAcknowledgeOrphans(
	latestAck uint64,
	thisAck uint64,
	offset uint64,
	multiple bool,
) error {
	err := new(ErrCantAcknowledgeOrphans)

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
