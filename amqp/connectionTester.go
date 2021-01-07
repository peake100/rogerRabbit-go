package amqp

// ConnectionTesting offers methods for running tests with Connection.
type ConnectionTesting struct {
	conn *Connection
	TransportTesting
}

// UnderlyingConn returns the current underlying streadway/amqp.Connection.
func (tester *ConnectionTesting) UnderlyingConn() *BasicConnection {
	return tester.conn.transportConn.BasicConnection
}
