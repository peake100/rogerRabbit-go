package amqp

import "time"

// Copy of defaults from streadway amqp
const (
	maxChannelMax = (2 << 15) - 1

	defaultHeartbeat         = 10 * time.Second
	defaultConnectionTimeout = 30 * time.Second
	defaultProduct           = "https://github.com/streadway/amqp"
	defaultVersion           = "β"
	// Safer default that makes ChannelConsume leaks a lot easier to spot
	// before they create operational headaches. See https://github.com/rabbitmq/rabbitmq-server/issues/1593.
	defaultChannelMax = (2 << 10) - 1
	defaultLocale     = "en_US"
)

type logData = map[string]interface{}
