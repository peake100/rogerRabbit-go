package amqp

import streadway "github.com/streadway/amqp"

type Delivery struct {
	streadway.Delivery

	tagOffset uint64
}

func newDelivery(orig streadway.Delivery, tagOffset *uint64) Delivery {
	delivery := Delivery{
		orig,
		*tagOffset,
	}
	delivery.DeliveryTag += delivery.tagOffset
	return delivery
}
