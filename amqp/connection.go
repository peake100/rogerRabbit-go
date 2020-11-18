package amqp

import (
	"context"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync"
)

// Implements transport for *streadway.Connection.
type transportConnection struct {
	*streadway.Connection
	dialUrl string
	dialConfig *Config
}

func (transport *transportConnection) cleanup() error {
	return nil
}

func (transport *transportConnection) tryReconnect(ctx context.Context) error {
	conn, err := streadway.DialConfig(transport.dialUrl, *transport.dialConfig)
	if err != nil {
		return err
	}

	transport.Connection = conn
	return nil
}

// A robust connection acts as a normal connection except that is automatically
// re-dials the broker when the underlying connection is lost.
//
// Unless otherwise noted at the beginning of their descriptions, all methods work
// exactly as their streadway counterparts, but will automatically re-attempt on
// ErrClosed errors. All other errors will be returned as normal. Descriptions have
// been copy-pasted from the streadway library for convenience.
//
// As this library evolves, other error types may be added to the list of errors we will
// automatically suppress and re-establish connection for, but in these early days,
// ErrClosed seems like a good place to start.
type Connection struct {
	// A transport object that contains our current underlying connection.
	transportConn *transportConnection

	// Manages the lifetime of the connection: such as automatic reconnects, connection
	// status events, and closing.
	*transportManager
}

// Gets a streadway channelConsume from the current connection.
func (conn *Connection) getStreadwayChannel(ctx context.Context) (
	channel *streadway.Channel, err error,
) {
	operation := func() error {
		var channelErr error
		channel, channelErr = conn.transportConn.Channel()
		return channelErr
	}

	err = conn.retryOperationOnClosed(ctx, operation, true)

	// Return the channelConsume and error.
	return channel, err
}

/*
Channel opens a unique, concurrent server channelConsume to process the bulk of AMQP
messages.  Any error from methods on this receiver will render the receiver
invalid and a new Channel should be opened.

*/
func (conn *Connection) Channel() (*Channel, error) {
	initialPublishCount := uint64(0)
	initialPublishOffset := uint64(0)
	initialConsumeCount := uint64(0)
	initialConsumeOffset := uint64(0)

	transportChan := &transportChannel{
		Channel:   nil,
		rogerConn: conn,
		settings: channelSettings{
			publisherConfirms: false,
			tagPublishCount:   &initialPublishCount,
			tagPublishOffset:  &initialPublishOffset,
			tagConsumeCount:   &initialConsumeCount,
			tagConsumeOffset:  &initialConsumeOffset,
		},
		declareQueues:            new(sync.Map),
		eventRelaysRunning:       new(sync.WaitGroup),
		eventRelaysRunSetup:      new(sync.WaitGroup),
		eventRelaysSetupComplete: new(sync.WaitGroup),
		eventRelaysGo:            new(sync.WaitGroup),
		logger:                   zerolog.Logger{},
	}

	// Add 1 to this WaitGroup so it can be released on the initial channel establish.
	transportChan.eventRelaysRunSetup.Add(1)

	manager := newTransportManager(conn.ctx, transportChan, "CHANNEL")
	transportChan.logger = manager.logger

	// Create our more robust channelConsume wrapper.
	rogerChannel := &Channel{
		transportChannel: transportChan,
		transportManager: manager,
	}

	// Try and establish a channelConsume using the connection's context, but returning an
	// error if the operation fails rather than retrying.
	err := rogerChannel.reconnect(conn.ctx, true)
	if err != nil {
		return nil, err
	}

	return rogerChannel, nil
}

// Get a new roger connection for a given url and connection config.
func newConnection(url string, config *Config) *Connection {
	// Create our robust connection object
	transportConn := &transportConnection{
		Connection: nil,
		dialUrl:    url,
		dialConfig: config,
	}

	transportManager := newTransportManager(
		context.Background(), transportConn, "CONNECTION",
	)


	conn := &Connection{
		transportConn:    transportConn,
		transportManager: transportManager,
	}

	return conn
}

// Get the default config for Dial() as it is in the streadway application.
func defaultConfig() *Config {
	return &Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
	}
}

// Dials a new robust connection. The initial dial will not retry endlessly, but report
// an error if it fails. After this method has returned a connection that has made a
// successful connection, it will be reconnected silently in the background in case
// of a disconnect.
func DialConfig(url string, config Config) (*Connection, error) {
	// Create the robust connection object.
	conn := newConnection(url, &config)
	// Make our initial connection
	err := conn.reconnect(conn.ctx, false)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Dials a new robust connection. The initial dial will not retry endlessly, but report
// an error if it fails. After this method has returned a connection that has made a
// successful connection, it will be reconnected silently in the background in case
// of a disconnect.
func Dial(url string) (*Connection, error) {
	// Use the same default config as streadway/amqp.
	config := defaultConfig()

	// Make our initial connection.
	return DialConfig(url, *config)
}

// As DialConfig, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialConfigCtx(
	ctx context.Context, url string, config Config,
) (*Connection, error) {
	conn := newConnection(url, &config)
	err := conn.reconnect(ctx,true)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// As DialConfigCtx, but with default configuration.
func DialCtx(
	ctx context.Context, url string,
) (*Connection, error) {
	// Use the same default config as streadway/amqp.
	config := defaultConfig()

	// Dial the connection.
	return DialConfigCtx(ctx, url, *config)
}
