package data

import streadway "github.com/streadway/amqp"

type Delivery struct {
	streadway.Delivery
	TagOffset uint64
}

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
