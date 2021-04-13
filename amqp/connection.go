package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/defaultmiddlewares"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"testing"
)

// Connection manages the serialization and deserialization of frames from IO
// and dispatches the frames to the appropriate channel.  All RPC methods and
// asynchronous Publishing, Delivery, Ack, Nack and Return messages are
// multiplexed on this channel.  There must always be active receivers for
// every asynchronous message on this connection.
//
// ---
//
// ROGER NOTE: A robust connection acts as a normal connection except that is
// automatically re-dials the broker when the underlying connection is lost.
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
	// Embedded streadway/amqp.Connection
	underlyingConn *BasicConnection

	// dialURL is the address to dial for the broker.
	dialURL string

	// dialConfig is the Config passed to our constructor method.
	dialConfig Config

	// streadwayConfig is extracted from the settings on dialConfig.
	streadwayConfig BasicConfig

	// handlerReconnect is the reconnect handler with caller-supplied middleware
	// applied.
	handlerReconnect amqpmiddleware.HandlerConnectionReconnect

	// transportManager manages the lifetime of the connection: such as automatic
	// reconnects, connection status events, and closing.
	transportManager
}

// transportType implements reconnects and returns "CONNECTION"
func (conn *Connection) transportType() amqpmiddleware.TransportType {
	return amqpmiddleware.TransportTypeConnection
}

// underlyingTransport implements reconnects and returns the underlying
// streadway.Connection as a livesOnce interface.
func (conn *Connection) underlyingTransport() livesOnce {
	return conn.underlyingConn
}

// cleanup implements reconnects. Does nothing for transportConnection.
func (conn *Connection) cleanup() error {
	return nil
}

// tryReconnect implements reconnects and tries to re-dial the broker one time.
func (conn *Connection) tryReconnect(
	ctx context.Context, attempt uint64,
) error {
	args := amqpmiddleware.ArgsConnectionReconnect{
		Ctx:     ctx,
		Attempt: attempt,
		Logger:  conn.dialConfig.Logger,
	}
	basicConn, err := conn.handlerReconnect(args)
	if err != nil {
		return err
	}

	conn.underlyingConn = basicConn
	return nil
}

// basicReconnectHandler is the innermost reconnection handler for the
// transportConnection.
func (conn *Connection) basicReconnectHandler(
	args amqpmiddleware.ArgsConnectionReconnect,
) (*streadway.Connection, error) {
	return streadway.DialConfig(conn.dialURL, conn.streadwayConfig)
}

// getStreadwayChannel gets a streadway/amqp.Channel from the current underlying
// connection.
func (conn *Connection) getStreadwayChannel(ctx context.Context) (
	channel *BasicChannel, err error,
) {
	operation := func() error {
		var channelErr error
		channel, channelErr = conn.underlyingConn.Channel()
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

---

ROGER NOTE: Unlike the normal channels, roger channels will automatically reconnect on
all errors until Channel.Close() is called.
*/
func (conn *Connection) Channel() (*Channel, error) {
	rogerChannel := &Channel{
		underlyingChannel: nil,
		rogerConn:         conn,
		handlers:          channelHandlers{},
		relaySync:         channelRelaySync{},
		logger:            zerolog.Logger{},
		transportManager:  transportManager{},
	}

	chanMiddleware := conn.dialConfig.ChannelMiddleware

	// Invoke any provider factories to make fresh provider values.
	err := chanMiddleware.buildAndAddProviderFactories()
	if err != nil {
		return nil, err
	}

	transportMiddleware := transportManagerMiddleware{
		notifyClose:            chanMiddleware.notifyClose,
		notifyDial:             chanMiddleware.notifyDial,
		notifyDisconnect:       chanMiddleware.notifyDisconnect,
		transportClose:         chanMiddleware.transportClose,
		notifyDialEvents:       chanMiddleware.notifyDialEvents,
		notifyDisconnectEvents: chanMiddleware.notifyDisconnectEvents,
		notifyCloseEvents:      chanMiddleware.notifyCloseEvents,
	}

	// Setup the transport manager.
	rogerChannel.transportManager.setup(
		conn.ctx, rogerChannel, transportMiddleware, conn.logger,
	)
	rogerChannel.logger = rogerChannel.transportManager.logger

	rogerChannel.relaySync = channelRelaySync{
		shared: newSharedSync(rogerChannel),
	}

	rogerChannel.handlers = newChannelHandlers(
		rogerChannel.rogerConn,
		rogerChannel,
		rogerChannel.transportManager.handlers,
		chanMiddleware,
	)

	// Try and establish a channel using the connection's context.
	err = rogerChannel.reconnect(conn.ctx, true)
	if err != nil {
		return nil, err
	}

	return rogerChannel, nil
}

// Test returns a ConnectionTesting object with a number of helper methods for testing
// Connection objects.
func (conn *Connection) Test(t *testing.T) *ConnectionTesting {
	blocks := int32(0)
	return &ConnectionTesting{
		conn: conn,
		TransportTesting: TransportTesting{
			t:       t,
			manager: &conn.transportManager,
			blocks:  &blocks,
		},
	}
}

// newConnection gets a new roger connection for a given url and connection middlewares.
func newConnection(url string, config Config) (*Connection, error) {
	// Extract our config into a streadway.Config
	streadwayConfig := streadway.Config{
		SASL:            config.SASL,
		Vhost:           config.Vhost,
		ChannelMax:      config.ChannelMax,
		FrameSize:       config.FrameSize,
		Heartbeat:       config.Heartbeat,
		TLSClientConfig: config.TLSClientConfig,
		Properties:      config.Properties,
		Locale:          config.Locale,
		Dial:            config.Dial,
	}

	// Add default middleware provider factories. Since the config is passed-by-value,
	// this will not mutate the caller's config.
	if !config.NoDefaultMiddleware {
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewQosMiddleware)
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewFlowMiddleware)
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewConfirmMiddleware)
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewPublishTagsMiddleware)
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewDeliveryTagsMiddleware)
		config.ChannelMiddleware.AddProviderFactory(defaultmiddlewares.NewRouteDeclarationMiddleware)
	}

	// Invoke middleware provider factories.
	err := config.ConnectionMiddleware.buildAndAddProviderFactories()
	if err != nil {
		return nil, err
	}

	middlewares := transportManagerMiddleware{
		notifyClose:            config.ConnectionMiddleware.notifyClose,
		notifyDial:             config.ConnectionMiddleware.notifyDial,
		notifyDisconnect:       config.ConnectionMiddleware.notifyDisconnect,
		transportClose:         config.ConnectionMiddleware.transportClose,
		notifyDialEvents:       config.ConnectionMiddleware.notifyDialEvents,
		notifyDisconnectEvents: config.ConnectionMiddleware.notifyDisconnectEvents,
		notifyCloseEvents:      config.ConnectionMiddleware.notifyCloseEvents,
	}

	// Create a new connection
	conn := &Connection{
		underlyingConn:  nil,
		dialURL:         url,
		dialConfig:      config,
		streadwayConfig: streadwayConfig,

		// We will defer setting these fields since they need to reference the
		// connection
		handlerReconnect: nil,
		transportManager: transportManager{},
	}

	// Setup the transport manager.
	conn.transportManager.setup(
		context.Background(), conn, middlewares, conn.dialConfig.Logger,
	)
	conn.logger = conn.transportManager.logger

	// Create the reconnect handler.
	reconnectHandler := conn.basicReconnectHandler
	for _, thisMiddleware := range config.ConnectionMiddleware.connectionReconnect {
		reconnectHandler = thisMiddleware(reconnectHandler)
	}

	conn.handlerReconnect = reconnectHandler

	return conn, nil
}
