package amqpmiddleware

import (
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	streadway "github.com/streadway/amqp"
)

// ArgsQueueDeclare stores args to amqp.Channel.QueueDeclare() for middleware to
// inspect.
type ArgsQueueDeclare struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       streadway.Table
}

// ArgsQueueInspect stores args to amqp.Channel.QueueInspect() for middleware to
// inspect.
type ArgsQueueInspect struct {
	Name string
}

// ArgsQueueDelete stores args to amqp.Channel.QueueDelete() for middleware to
// inspect.
type ArgsQueueDelete struct {
	Name     string
	IfUnused bool
	IfEmpty  bool
	NoWait   bool
}

// ArgsQueueBind stores args to amqp.Channel.QueueBind() for middleware to
// inspect.
type ArgsQueueBind struct {
	Name     string
	Key      string
	Exchange string
	NoWait   bool
	Args     streadway.Table
}

// ArgsQueueUnbind stores args to amqp.Channel.QueueUnbind() for middleware to
// inspect.
type ArgsQueueUnbind struct {
	Name     string
	Key      string
	Exchange string
	Args     streadway.Table
}

// ArgsQueuePurge stores args to amqp.Channel.QueuePurge() for middleware to
// inspect.
type ArgsQueuePurge struct {
	Name   string
	NoWait bool
}

// ArgsExchangeDeclare stores args to amqp.Channel.ExchangeDeclare() for middleware to
// inspect.
type ArgsExchangeDeclare struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       streadway.Table
}

// ArgsExchangeDelete stores args to amqp.Channel.ExchangeDelete() for middleware to
// inspect.
type ArgsExchangeDelete struct {
	Name     string
	IfUnused bool
	NoWait   bool
}

// ArgsExchangeBind stores args to amqp.Channel.ExchangeBind() for middleware to
// inspect.
type ArgsExchangeBind struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        streadway.Table
}

// ArgsExchangeUnbind stores args to amqp.Channel.ExchangeUnbind() for middleware to
// inspect.
type ArgsExchangeUnbind struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        streadway.Table
}

// ArgsQoS stores args to amqp.Channel.QoS() for middleware to inspect.
type ArgsQoS struct {
	PrefetchCount int
	PrefetchSize  int
	Global        bool
}

// ArgsFlow stores args to amqp.Channel.Flow() for middleware to inspect.
type ArgsFlow struct {
	Active bool
}

// ArgsConfirms stores args to amqp.Channel.Confirms() for middleware to inspect.
type ArgsConfirms struct {
	NoWait bool
}

// ArgsPublish stores args to amqp.Channel.Publish() for middleware to inspect.
type ArgsPublish struct {
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
	Msg       streadway.Publishing
}

// ArgsGet stores args to amqp.Channel.Get() for middleware to inspect.
type ArgsGet struct {
	Queue   string
	AutoAck bool
}

// ArgsConsume stores args to amqp.Channel.Consume() for middleware to inspect.
type ArgsConsume struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      streadway.Table
}

// ArgsAck stores args to amqp.Channel.Ack() for middleware to inspect.
type ArgsAck struct {
	Tag      uint64
	Multiple bool
}

// ArgsNack stores args to amqp.Channel.Nack() for middleware to inspect.
type ArgsNack struct {
	Tag      uint64
	Multiple bool
	Requeue  bool
}

// ArgsReject stores args to amqp.Channel.Reject() for middleware to inspect.
type ArgsReject struct {
	Tag     uint64
	Requeue bool
}

// ArgsNotifyPublish stores args to amqp.Channel.NotifyPublish() for middleware to
// inspect.
type ArgsNotifyPublish struct {
	Confirm chan datamodels.Confirmation
}

// ArgsNotifyConfirm stores args to amqp.Channel.NotifyConfirm() for middleware to
// inspect.
type ArgsNotifyConfirm struct {
	Ack  chan uint64
	Nack chan uint64
}

// ArgsNotifyConfirmOrOrphaned stores args to amqp.Channel.NotifyConfirmOrOrphaned() for
// middleware to inspect.
type ArgsNotifyConfirmOrOrphaned struct {
	Ack      chan uint64
	Nack     chan uint64
	Orphaned chan uint64
}

// ArgsNotifyReturn stores the args to amqp.Channel.NotifyReturn for middleware to
// inspect.
type ArgsNotifyReturn struct {
	Returns chan streadway.Return
}

// ArgsNotifyCancel stores the args to amqp.Channel.NotifyCancel for middleware to
// inspect.
type ArgsNotifyCancel struct {
	Cancellations chan string
}

// ArgsNotifyFlow stores the args to amqp.Channel.NotifyFlow for middleware to inspect.
type ArgsNotifyFlow struct {
	FlowNotifications chan bool
}

// EventNotifyPublish passes event information from a  to amqp.Channel.NotifyPublish()
// event for middleware to inspect / modify before the event is passed to the caller.
type EventNotifyPublish struct {
	Confirmation datamodels.Confirmation
}

// EventConsume passes event information from a to amqp.Channel.Consume()
// event for middleware to inspect / modify before the event is passed to the caller.
type EventConsume struct {
	Delivery datamodels.Delivery
}
