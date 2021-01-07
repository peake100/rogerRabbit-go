package amqp

import "github.com/peake100/rogerRabbit-go/amqp/defaultMiddlewares"

// ErrCantAcknowledgeOrphans is returned when an acknowledgement method
// (ack, nack, reject) cannot be completed because the original channel it was consumed
// from has been closed and replaced with a new one. When part of a multi-ack, it's
// possible that SOME tags will be orphaned and some will succeed, this error contains
// detailed information on both groups
type ErrCantAcknowledgeOrphans = defaultMiddlewares.ErrCantAcknowledgeOrphans
