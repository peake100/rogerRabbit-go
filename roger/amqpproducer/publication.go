package amqpproducer

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
)

// Publication is the order for a publications.
type Publication struct {
	// ctx is a context for this order from the original caller.
	ctx context.Context

	// publishArgs are the embedded publication args.
	args publishArgs

	// publicationTag is set by the publishing routine after a successful publication.
	publicationTag uint64
	tagSet         bool

	// result us a channel we will send publishing results back to the original caller
	// with.
	result chan error
}

// WaitOnConfirmation blocks until the message is confirmed / nacked by the broker
// or the context of the publication cancels.
func (order *Publication) WaitOnConfirmation() error {
	// Block until we have a final result, then return it.
	select {
	case result := <-order.result:
		return result
	case <-order.ctx.Done():
		return fmt.Errorf("message cancelled: %w", order.ctx.Err())
	}
}

// publishArgs are the args we are going to call channel.Publish with. See that methods
// documentation for details on each args meaning.
type publishArgs struct {
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
	Msg       amqp.Publishing
}
