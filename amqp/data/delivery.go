package data

import streadway "github.com/streadway/amqp"

type Delivery struct {
	streadway.Delivery
	TagOffset uint64
}

func NewDelivery(
	orig streadway.Delivery,
	tagOffset uint64,
	acknowledger streadway.Acknowledger,
) Delivery {
	delivery := Delivery{
		Delivery: orig,
		TagOffset: tagOffset,
	}
	delivery.DeliveryTag += tagOffset

	// Swap out the acknowledger for our robust channel so ack, nack, and reject
	// methods call our robust channel rather than the underlying one.
	delivery.Acknowledger = acknowledger

	return delivery
}
