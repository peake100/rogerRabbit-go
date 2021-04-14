package amqpmiddleware

import (
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	streadway "github.com/streadway/amqp"
)

// ResultsNotifyClose are the result values from a call to the NotifyClose method on
// a Connection or Channel.
type ResultsNotifyClose struct {
	CallerChan chan *streadway.Error
}

// ResultsChannelReconnect are the result values from a Connection reconnection event
// for middleware to inspect.
type ResultsConnectionReconnect struct {
	Connection *streadway.Connection
}

// ResultsChannelReconnect are the result values from a Channel reconnection event for
// middleware to inspect.
type ResultsChannelReconnect struct {
	Channel *streadway.Channel
}

// ResultsQueueDeclare are the result values from a Channel.QueuePurge call for
// middleware to inspect.
type ResultsQueueDeclare struct {
	Queue streadway.Queue
}

// ResultsQueueInspect are the result values from a Channel.QueueInspect call for
// middleware to inspect.
type ResultsQueueInspect struct {
	Queue streadway.Queue
}

// ResultsQueuePurge are the result values from a Channel.QueuePurge call for middleware
// to inspect.
type ResultsQueuePurge struct {
	Count int
}

// ResultsQueueDelete are the result values from a Channel.QueueDelete call for
// middleware to inspect.
type ResultsQueueDelete struct {
	Count int
}

// ResultsGet are the result values from a Channel.Get call for middleware to inspect.
type ResultsGet struct {
	Msg datamodels.Delivery
	Ok  bool
}

// ResultsConsume are the result values from a Channel.Consume call for middleware to
// inspect.
type ResultsConsume struct {
	DeliveryChan <-chan datamodels.Delivery
}

// ResultsNotifyConfirm are the result values from a Channel.NotifyConfirm call for
// middleware to inspect.
type ResultsNotifyConfirm struct {
	Ack  chan uint64
	Nack chan uint64
}

// ResultsNotifyConfirmOrOrphaned are the result values from a
// Channel.NotifyConfirmOrOrphaned call for middleware to inspect.
type ResultsNotifyConfirmOrOrphaned struct {
	Ack      chan uint64
	Nack     chan uint64
	Orphaned chan uint64
}

// ResultsNotifyReturn stores the results from amqp.Channel.NotifyReturn for middleware
// to inspect.
type ResultsNotifyReturn struct {
	Returns chan streadway.Return
}

// ResultsNotifyCancel stores the results from amqp.Channel.NotifyCancel for middleware
// to inspect.
type ResultsNotifyCancel struct {
	Cancellations chan string
}

// ResultsNotifyFlow stores the results from amqp.Channel.NotifyFlow for middleware to
// inspect.
type ResultsNotifyFlow struct {
	FlowNotifications chan bool
}

// ResultsNotifyPublish stores the results from amqp.Channel.NotifyPublish() for
// middleware to inspect.
type ResultsNotifyPublish struct {
	Confirm chan datamodels.Confirmation
}
