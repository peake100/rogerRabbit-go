package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/data"
	streadway "github.com/streadway/amqp"
)

func (channel *Channel) NewDelivery(orig streadway.Delivery) data.Delivery {
	tagOffset := channel.transportChannel.settings.tagConsumeOffset
	return data.NewDelivery(orig, tagOffset, channel)
}
