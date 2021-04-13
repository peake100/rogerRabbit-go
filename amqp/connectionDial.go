package amqp

import (
	"context"
	"crypto/tls"
)

// Dial accepts a string in the AMQP URI format and returns a new Connection
// over TCP using PlainAuth.  Defaults to a server heartbeat interval of 10
// seconds and sets the handshake deadline to 30 seconds. After handshake,
// deadlines are cleared.
//
// Dial uses the zero value of tls.Config when it encounters an amqps://
// scheme.  It is equivalent to calling DialTLS(amqp, nil).
func Dial(url string) (*Connection, error) {
	// Use the same default middlewares as streadway/amqp.
	config := DefaultConfig()

	// Make our initial connection.
	return DialConfig(url, config)
}

// DialConfig accepts a string in the AMQP URI format and a configuration for
// the livesOnce and connection setup, returning a new Connection.  Defaults to
// a server heartbeat interval of 10 seconds and sets the initial read deadline
// to 30 seconds.
func DialConfig(url string, config Config) (*Connection, error) {
	// Create the robust connection object.
	conn := newConnection(url, config)
	// Make our initial connection
	err := conn.reconnect(conn.ctx, false)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// DialTLS accepts a string in the AMQP URI format and returns a new Connection
// over TCP using PlainAuth.  Defaults to a server heartbeat interval of 10
// seconds and sets the initial read deadline to 30 seconds.
//
// DialTLS uses the provided tls.Config when encountering an amqps:// scheme.
func DialTLS(url string, amqps *tls.Config) (*Connection, error) {
	config := DefaultConfig()
	config.TLSClientConfig = amqps

	return DialConfig(url, config)
}

// As DialConfig, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialConfigCtx(
	ctx context.Context, url string, config Config,
) (*Connection, error) {
	conn := newConnection(url, config)
	err := conn.reconnect(ctx, true)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// As Dial, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialCtx(
	ctx context.Context, url string,
) (*Connection, error) {
	// Use the same default middlewares as streadway/amqp.
	config := DefaultConfig()

	// Dial the connection.
	return DialConfigCtx(ctx, url, config)
}

// As DialTLS, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialTLSCtx(
	ctx context.Context, url string, amqps *tls.Config,
) (*Connection, error) {
	config := DefaultConfig()
	config.TLSClientConfig = amqps

	return DialConfigCtx(ctx, url, config)
}
