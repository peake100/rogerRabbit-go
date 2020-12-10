package amqp

import streadway "github.com/streadway/amqp"

type ConnectionTesting struct {
	conn *Connection
	transportTesting
}

// Returns the current underlying streadway/amqp connection object.
func (tester *ConnectionTesting) UnderlyingConn() *streadway.Connection {
	return tester.conn.transportConn.Connection
}
