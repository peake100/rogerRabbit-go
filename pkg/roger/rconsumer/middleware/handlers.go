package middleware

import (
	"context"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	streadway "github.com/streadway/amqp"
)

// AmqpRouteManager is an interface that only exposes the Queue and exchange methods for
// a channel. These are the only methods we want SetupChannel to have access to.
type AmqpRouteManager interface {
	QueueDeclare(
		name string,
		durable bool,
		autoDelete bool,
		exclusive bool,
		noWait bool,
		args streadway.Table,
	) (queue amqp.Queue, err error)
	QueueDeclarePassive(
		name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table,
	) (queue amqp.Queue, err error)
	QueueInspect(name string) (queue amqp.Queue, err error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueuePurge(name string, noWait bool) (count int, err error)
	QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (count int, err error)

	ExchangeDeclare(
		name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table,
	) (err error)
	ExchangeDeclarePassive(
		name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table,
	) (err error)
	ExchangeBind(
		destination, key, source string, noWait bool, args amqp.Table,
	) (err error)
	ExchangeDelete(name string, ifUnused, noWait bool) (err error)
}

// HandlerSetupChannel defines a handler type for middleware wrapping
// roger.Consumer.SetupChannel
type HandlerSetupChannel = func(ctx context.Context, amqpChannel AmqpRouteManager) error

// HandlerDelivery defines the handler type for middleware wrapping
// roger.Consumer.HandleDelivery.
type HandlerDelivery = func(ctx context.Context, delivery amqp.Delivery) (requeue bool, err error)

// HandlerCleanupChannel defines the handler type for middleware wrapping
// roger.Consumer.CleanupChannel
type HandlerCleanupChannel = func(ctx context.Context, amqpChannel AmqpRouteManager) error
