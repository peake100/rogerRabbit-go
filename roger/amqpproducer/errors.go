package amqpproducer

import "fmt"

// ErrPublish is the error returned when trying to publish through the broker
// channel results in an error.
type ErrPublish struct {
	AmqpErr error
}

// Unwrap implements xerrors.Wrapper and returns the original Amqp error.
func (err ErrPublish) Unwrap() error {
	return err.AmqpErr
}

// Error implements builtins.error.
func (err ErrPublish) Error() string {
	return fmt.Sprintf(
		"error publishing message through broker channel: %v", err.AmqpErr,
	)
}

// ErrNack is the error returned when trying to publish through the broker
// channel results in an error.
type ErrNack struct {
	// Orphan is whether this nack was a result of being an orphaned message. Orphaned
	// messages MAY have been successfully published, but there is no way to know for
	// sure that the broker received it.
	Orphan bool
}

// Error implements builtins.error.
func (err ErrNack) Error() string {
	return fmt.Sprintf(
		"message was nacked by server. orphan status: %v", err.Orphan,
	)
}
