package internal

import streadway "github.com/streadway/amqp"

// Delivery captures the fields for a previously delivered message resident in
// a queue to be delivered by the server to a consumer from Channel.Consume or
// Channel.Get.
//
// --
//
// ROGER NOTE: As streadway/amqp.Delivery, but with additional data specifying the tag
// offset that was applied to this delivery to create a continuous, non-doubled
// delivery tags across disconnect.
type Delivery struct {
	// The original delivery from the underlying channel, embedded in our struct so it
	// can act as a drop-in replacement
	streadway.Delivery

	// TagOffset is the offset applied to our Delivery tag to create a continuous stream
	// of delivery tag values across disconnects.
	TagOffset uint64
}

// NewDelivery created a new Delivery object from an original source streadway.Delivery
// value plus our new Acknowledger.
func NewDelivery(
	orig streadway.Delivery,
	acknowledger streadway.Acknowledger,
) Delivery {
	// Swap out the acknowledger for our robust channel so ack, nack, and reject
	// methods call our robust channel rather than the underlying one.
	orig.Acknowledger = acknowledger

	// Embed the original delivery
	delivery := Delivery{
		Delivery:  orig,
		TagOffset: 0,
	}

	return delivery
}
