package amqp

import streadway "github.com/streadway/amqp"

// ROGER NOTE: As streadway/amqp.Confirmation, but with additional data on whether this
// confirmation was received directly from the broker, or created to fill in a gap in
// delivery tags that occurred due to an unexpected disconnection.
//
// --
//
// Confirmation notifies the acknowledgment or negative acknowledgement of a
// publishing identified by its delivery tag.  Use NotifyPublish on the Channel
// to consume these events.
type Confirmation struct {
	// The original confirmation message, embedded to this struct can act as a drop-in
	// replacement
	streadway.Confirmation

	// When this value is true, this confirmation was never received due to a connection
	// disruption, and is being reported as a NACK. However, it is possible that this
	// message DID successfully reach the server, and only the confirmation response
	// was never received.
	DisconnectOrphan bool
}
