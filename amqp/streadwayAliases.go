package amqp

import streadway "github.com/streadway/amqp"

/*
In this file we are going ot create type aliases for all streadway types we are NOT
re-implementing
*/

// Copy over the error codes
const (
	ContentTooLarge    = streadway.ContentTooLarge
	NoRoute            = streadway.NoRoute
	NoConsumers        = streadway.NoConsumers
	ConnectionForced   = streadway.ConnectionForced
	InvalidPath        = streadway.InvalidPath
	AccessRefused      = streadway.AccessRefused
	NotFound           = streadway.NotFound
	ResourceLocked     = streadway.ResourceLocked
	PreconditionFailed = streadway.PreconditionFailed
	FrameError         = streadway.FrameError
	SyntaxError        = streadway.SyntaxError
	CommandInvalid     = streadway.CommandInvalid
	ChannelError       = streadway.ChannelError
	UnexpectedFrame    = streadway.UnexpectedFrame
	ResourceError      = streadway.ResourceError
	NotAllowed         = streadway.NotAllowed
	NotImplemented     = streadway.NotImplemented
	InternalError      = streadway.InternalError
)

// Aliases to sentinel errors
var (
	ErrChannelMax      = streadway.ErrChannelMax
	ErrClosed          = streadway.ErrClosed
	ErrCommandInvalid  = streadway.ErrCommandInvalid
	ErrCredentials     = streadway.ErrCredentials
	ErrFieldType       = streadway.ErrFieldType
	ErrFrame           = streadway.ErrFrame
	ErrSASL            = streadway.ErrSASL
	ErrSyntax          = streadway.ErrSyntax
	ErrUnexpectedFrame = streadway.ErrUnexpectedFrame
	ErrVhost           = streadway.ErrVhost
)

// DeliveryMode.  Transient means higher throughput but messages will not be
// restored on broker restart.  The delivery mode of publishings is unrelated
// to the durability of the queues they reside on.  Transient messages will
// not be restored to durable queues, persistent messages will be restored to
// durable queues and lost on non-durable queues during server restart.
//
// This remains typed as uint8 to match Publishing.DeliveryMode.  Other
// delivery modes specific to custom queue implementations are not enumerated
// here.
const (
	Persistent = streadway.Persistent
	Transient  = streadway.Transient
)

// Constants for standard AMQP 0-9-1 exchange types.
const (
	ExchangeDirect  = streadway.ExchangeDirect
	ExchangeFanout  = streadway.ExchangeFanout
	ExchangeHeaders = streadway.ExchangeHeaders
	ExchangeTopic   = streadway.ExchangeTopic
)

// Type Aliases
type (
	Acknowledger = streadway.Acknowledger

	// Authentication interface provides a means for different SASL authentication
	// mechanisms to be used during connection tuning.
	Authentication = streadway.Authentication

	// Blocking notifies the server's TCP flow control of the Connection.  When a
	// server hits a memory or disk alarm it will block all connections until the
	// resources are reclaimed.  Use NotifyBlock on the Connection to receive these
	// events.
	Blocking = streadway.Blocking

	// Decimal matches the AMQP decimal type.  Scale is the number of decimal
	// digits Scale == 2, Value == 12345, Decimal == 123.45
	Decimal = streadway.Decimal

	// Error captures the code and reason a channelConsume or connection has been closed
	// by the server.
	Error = streadway.Error

	// Publishing captures the client message sent to the server.  The fields
	// outside of the Headers table included in this struct mirror the underlying
	// fields in the content frame.  They use native types for convenience and
	// efficiency.
	Publishing = streadway.Publishing

	// Queue captures the current server state of the queue on the server returned
	// from Channel.QueueDeclare or Channel.QueueInspect.
	Queue = streadway.Queue

	// Return captures a flattened struct of fields returned by the server when a
	// Publishing is unable to be delivered either due to the `mandatory` flag set
	// and no route found, or `immediate` flag set and no free consumer.
	Return = streadway.Return

	// Table stores user supplied fields of the following types:
	//
	//   bool
	//   byte
	//   float32
	//   float64
	//   int
	//   int16
	//   int32
	//   int64
	//   nil
	//   string
	//   time.Time
	//   amqp.Decimal
	//   amqp.Table
	//   []byte
	//   []interface{} - containing above types
	//
	// Functions taking a table will immediately fail when the table contains a
	// value of an unsupported type.
	//
	// The caller must be specific in which precision of integer it wishes to
	// encode.
	//
	// Use a type assertion when reading values from a table for type conversion.
	//
	// RabbitMQ expects int32 for integer values.
	//
	Table = streadway.Table
)
