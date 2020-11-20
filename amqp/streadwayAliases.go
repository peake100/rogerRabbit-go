/*
In this file we are going ot create type aliases for all streadway types we are NOT
re-implementing
 */
package amqp

import streadway "github.com/streadway/amqp"

// Config is used in DialConfig and Open to specify the desired tuning
// parameters used during a connection open handshake.  The negotiated tuning
// will be stored in the returned connection's Config field.
type Config = streadway.Config

// Error captures the code and reason a channelConsume or connection has been closed
// by the server.
type Error = streadway.Error

// Publishing captures the client message sent to the server.  The fields
// outside of the Headers table included in this struct mirror the underlying
// fields in the content frame.  They use native types for convenience and
// efficiency.
type Publishing = streadway.Publishing

// Queue captures the current server state of the queue on the server returned
// from Channel.QueueDeclare or Channel.QueueInspect.
type Queue = streadway.Queue

// Return captures a flattened struct of fields returned by the server when a
// Publishing is unable to be delivered either due to the `mandatory` flag set
// and no route found, or `immediate` flag set and no free consumer.
type Return = streadway.Return

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
type Table = streadway.Table
