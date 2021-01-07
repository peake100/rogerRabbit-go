package datamodels

import streadway "github.com/streadway/amqp"

// Confirmation notifies the acknowledgment or negative acknowledgement of a
// publishing identified by its delivery tag.  Use NotifyPublish on the Channel
// to consume these events.
//
// --
//
// ROGER NOTE: As streadway/amqp.Confirmation, but with additional data on whether this
// confirmation was received directly from the broker, or created to fill in a gap in
// delivery tags that occurred due to an unexpected disconnection.
type Confirmation struct {
	// The original confirmation message, embedded in our struct so it can act as a
	// drop-in replacement
	streadway.Confirmation

	// When DisconnectOrphan value is true, this confirmation was never received due to
	// a connection disruption, and is being reported as a NACK. However, it is possible
	// that this message DID successfully reach the server, and only the confirmation
	// response was never received.
	DisconnectOrphan bool
}
