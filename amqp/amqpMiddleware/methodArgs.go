package amqpMiddleware

import (
	"github.com/peake100/rogerRabbit-go/amqp/data"
	streadway "github.com/streadway/amqp"
)

// Store queue declare information for re-establishing queues on disconnect.
type ArgsQueueDeclare struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       streadway.Table
}

// Arguments passed to channel delete.
type ArgsQueueDelete struct {
	Name     string
	IfUnused bool
	IfEmpty  bool
	NoWait   bool
}

// Store queue bind information for re-establishing bindings on disconnect.
type ArgsQueueBind struct {
	Name     string
	Key      string
	Exchange string
	NoWait   bool
	Args     streadway.Table
}

type ArgsQueueUnbind struct {
	Name     string
	Key      string
	Exchange string
	Args     streadway.Table
}

// Store exchange declare information for re-establishing queues on disconnect.
type ArgsExchangeDeclare struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       streadway.Table
}

type ArgsExchangeDelete struct {
	Name     string
	IfUnused bool
	NoWait   bool
}

// Store exchange bind information for re-establishing bindings on disconnect.
type ArgsExchangeBind struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        streadway.Table
}

type ArgsExchangeUnbind struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        streadway.Table
}

type ArgsQoS struct {
	PrefetchCount int
	PrefetchSize  int
	Global        bool
}

type ArgsConfirms struct {
	NoWait bool
}

type ArgsPublish struct {
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
	Msg       streadway.Publishing
}

type ArgsGet struct {
	Queue   string
	AutoAck bool
}

type ArgsAck struct {
	Tag      uint64
	Multiple bool
}

type ArgsNack struct {
	Tag      uint64
	Multiple bool
	Requeue  bool
}

type ArgsReject struct {
	Tag     uint64
	Requeue bool
}

type ArgsNotifyPublish struct {
	Confirm chan data.Confirmation
}

type EventNotifyPublish struct {
	Confirmation data.Confirmation
}
