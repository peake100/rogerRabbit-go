package amqp

import streadway "github.com/streadway/amqp"

type Delivery struct {
	streadway.Delivery

	tagOffset uint64
}

func (channel *Channel) newDelivery(orig streadway.Delivery) Delivery {
	tagOffset := channel.transportChannel.settings.tagConsumeOffset

	delivery := Delivery{
		orig,
		tagOffset,
	}
	delivery.DeliveryTag += tagOffset

	// Swap out the acknowledger for our robust channel so ack, nack, and reject
	// methods call our robust channel rather than the underlying one.
	delivery.Acknowledger = channel
	return delivery
}
