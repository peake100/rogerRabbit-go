package amqp

import streadway "github.com/streadway/amqp"

// ConnectionTesting offers methods for running tests with Connection.
type ConnectionTesting struct {
	conn *Connection
	TransportTesting
}

// UnderlyingConn returns the current underlying streadway/amqp.Connection.
func (tester *ConnectionTesting) UnderlyingConn() *streadway.Connection {
	return tester.conn.transportConn.Connection
}
