package amqp

import (
	"context"
	"crypto/tls"
	"github.com/peake100/rogerRabbit-go/amqp/defaultMiddlewares"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"testing"
)

// Implements transport for *streadway.Connection.
type transportConnection struct {
	*streadway.Connection
	dialURL string

	// The config option passed to this connection
	dialConfig *Config

	// Teh streadway config extracted from the settings on dialConfig
	streadwayConfig *streadway.Config
}

func (transport *transportConnection) cleanup() error {
	return nil
}

func (transport *transportConnection) tryReconnect(ctx context.Context) error {
	conn, err := streadway.DialConfig(transport.dialURL, *transport.streadwayConfig)
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

func newChannelApplyDefaultMiddleware(channel *Channel, config *Config) {
	if config.NoDefaultMiddleware {
		return
	}

	handlers := channel.transportChannel.handlers
	middlewareStorage := channel.transportChannel.defaultMiddlewares

	// Qos middleware
	qosMiddleware := defaultMiddlewares.NewQosMiddleware()
	handlers.AddReconnect(qosMiddleware.Reconnect)
	handlers.AddQoS(qosMiddleware.Qos)
	middlewareStorage.QoS = qosMiddleware

	// Flow middleware
	flowMiddleware := defaultMiddlewares.NewFlowMiddleware()
	handlers.AddReconnect(flowMiddleware.Reconnect)
	handlers.AddFlow(flowMiddleware.Flow)
	middlewareStorage.Flow = flowMiddleware

	// Confirmation middleware
	confirmMiddleware := defaultMiddlewares.NewConfirmMiddleware()
	handlers.AddReconnect(confirmMiddleware.Reconnect)
	handlers.AddConfirm(confirmMiddleware.Confirm)
	middlewareStorage.Confirm = confirmMiddleware

	// Publish Tags middleware
	publishTagsMiddleware := defaultMiddlewares.NewPublishTagsMiddleware()
	handlers.AddReconnect(publishTagsMiddleware.Reconnect)
	handlers.AddConfirm(publishTagsMiddleware.Confirm)
	handlers.AddPublish(publishTagsMiddleware.Publish)
	handlers.AddNotifyPublishEvent(publishTagsMiddleware.NotifyPublishEvent)
	middlewareStorage.PublishTags = publishTagsMiddleware

	// Delivery Tags middleware
	deliveryTagsMiddleware := defaultMiddlewares.NewDeliveryTagsMiddleware()
	handlers.AddReconnect(deliveryTagsMiddleware.Reconnect)
	handlers.AddGet(deliveryTagsMiddleware.Get)
	handlers.AddConsumeEvent(deliveryTagsMiddleware.ConsumeEvent)
	handlers.AddAck(deliveryTagsMiddleware.Ack)
	handlers.AddNack(deliveryTagsMiddleware.Nack)
	handlers.AddReject(deliveryTagsMiddleware.Reject)
	middlewareStorage.DeliveryTags = deliveryTagsMiddleware

	// Route declaration middleware
	declarationMiddleware := defaultMiddlewares.NewRouteDeclarationMiddleware()
	handlers.AddReconnect(declarationMiddleware.Reconnect)
	handlers.AddQueueDeclare(declarationMiddleware.QueueDeclare)
	handlers.AddQueueDelete(declarationMiddleware.QueueDelete)
	handlers.AddQueueBind(declarationMiddleware.QueueBind)
	handlers.AddQueueUnbind(declarationMiddleware.QueueUnbind)
	handlers.AddExchangeDeclare(declarationMiddleware.ExchangeDeclare)
	handlers.AddExchangeDelete(declarationMiddleware.ExchangeDelete)
	handlers.AddExchangeBind(declarationMiddleware.ExchangeBind)
	handlers.AddExchangeUnbind(declarationMiddleware.ExchangeUnbind)
	middlewareStorage.RouteDeclaration = declarationMiddleware
}

/*
ROGER NOTE: Unlike the normal channels, roger channels will automatically reconnect on
all errors until Channel.Close() is called.

--

Channel opens a unique, concurrent server channelConsume to process the bulk of AMQP
messages.  Any error from methods on this receiver will render the receiver
invalid and a new Channel should be opened.

*/
func (conn *Connection) Channel() (*Channel, error) {
	transportChan := &transportChannel{
		Channel:            nil,
		rogerConn:          conn,
		handlers:           nil,
		defaultMiddlewares: new(ChannelTestingDefaultMiddlewares),
		logger:             zerolog.Logger{},
	}

	manager := newTransportManager(conn.ctx, transportChan, "CHANNEL")
	transportChan.logger = manager.logger

	// Create our more robust channelConsume wrapper.
	rogerChannel := &Channel{
		transportChannel: transportChan,
		transportManager: manager,
	}

	transportChan.relaySync = channelRelaySync{
		shared: newSharedSync(rogerChannel),
	}

	// Initialize our channel handlers with the new channel
	transportChan.handlers = newChannelHandlers(conn, rogerChannel)
	// Add default middleware around these handlers.
	newChannelApplyDefaultMiddleware(rogerChannel, conn.transportConn.dialConfig)

	// Try and establish a channel using the connection's context.
	err := rogerChannel.reconnect(conn.ctx, true)
	if err != nil {
		return nil, err
	}

	return rogerChannel, nil
}

func (conn *Connection) Test(t *testing.T) *ConnectionTesting {
	blocks := int32(0)
	return &ConnectionTesting{
		conn: conn,
		transportTesting: transportTesting{
			t:       t,
			manager: conn.transportManager,
			blocks:  &blocks,
		},
	}
}

// Get a new roger connection for a given url and connection config.
func newConnection(url string, config *Config) *Connection {
	streadwayConfig := &streadway.Config{
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

	// Create our robust connection object
	transportConn := &transportConnection{
		Connection:      nil,
		dialURL:         url,
		dialConfig:      config,
		streadwayConfig: streadwayConfig,
	}

	transportManager := newTransportManager(
		context.Background(), transportConn, "CONNECTION",
	)
	// Use the logger from the config
	transportManager.logger = config.Logger

	conn := &Connection{
		transportConn:    transportConn,
		transportManager: transportManager,
	}

	return conn
}

// Dial accepts a string in the AMQP URI format and returns a new Connection
// over TCP using PlainAuth.  Defaults to a server heartbeat interval of 10
// seconds and sets the handshake deadline to 30 seconds. After handshake,
// deadlines are cleared.
//
// Dial uses the zero value of tls.Config when it encounters an amqps://
// scheme.  It is equivalent to calling DialTLS(amqp, nil).
func Dial(url string) (*Connection, error) {
	// Use the same default config as streadway/amqp.
	config := DefaultConfig()

	// Make our initial connection.
	return DialConfig(url, *config)
}

// DialConfig accepts a string in the AMQP URI format and a configuration for
// the transport and connection setup, returning a new Connection.  Defaults to
// a server heartbeat interval of 10 seconds and sets the initial read deadline
// to 30 seconds.
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

// DialTLS accepts a string in the AMQP URI format and returns a new Connection
// over TCP using PlainAuth.  Defaults to a server heartbeat interval of 10
// seconds and sets the initial read deadline to 30 seconds.
//
// DialTLS uses the provided tls.Config when encountering an amqps:// scheme.
func DialTLS(url string, amqps *tls.Config) (*Connection, error) {
	config := DefaultConfig()
	config.TLSClientConfig = amqps

	return DialConfig(url, *config)
}

// As DialConfig, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialConfigCtx(
	ctx context.Context, url string, config Config,
) (*Connection, error) {
	conn := newConnection(url, &config)
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
	// Use the same default config as streadway/amqp.
	config := DefaultConfig()

	// Dial the connection.
	return DialConfigCtx(ctx, url, *config)
}

// As DialTLS, but endlessly redials the connection until ctx is cancelled. Once
// returned, cancelling ctx does not affect the connection.
func DialTLSCtx(
	ctx context.Context, url string, amqps *tls.Config,
) (*Connection, error) {
	config := DefaultConfig()
	config.TLSClientConfig = amqps

	return DialConfigCtx(ctx, url, *config)
}
