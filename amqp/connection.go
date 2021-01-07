package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/defaultMiddlewares"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"testing"
)

// transportConnection implements transport for streadway/amqp.Connection.
type transportConnection struct {
	// Embedded streadway/amqp.Connection
	*BasicConnection

	// dialURL is the address to dial for the broker.
	dialURL string

	// dialConfig is the Config passed to our constructor method.
	dialConfig *Config

	// streadwayConfig is extracted from the settings on dialConfig.
	streadwayConfig *BasicConfig
}

// cleanup implements transportManager. Does nothing for transportConnection.
func (transport *transportConnection) cleanup() error {
	return nil
}

// tryReconnect implements transportManager and tries to re-dial the broker one time.
func (transport *transportConnection) tryReconnect(ctx context.Context) error {
	conn, err := streadway.DialConfig(transport.dialURL, *transport.streadwayConfig)
	if err != nil {
		return err
	}

	transport.BasicConnection = conn
	return nil
}

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
	// transportConn is a transport object that contains our current underlying
	// connection.
	transportConn *transportConnection

	// transportManager manages the lifetime of the connection: such as automatic
	// reconnects, connection status events, and closing.
	*transportManager
}

// getStreadwayChannel gets a streadway/amqp.Channel from the current underlying
// connection.
func (conn *Connection) getStreadwayChannel(ctx context.Context) (
	channel *BasicChannel, err error,
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

// newChannelApplyDefaultMiddleware applies the default middleware to a new Channel.
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
Channel opens a unique, concurrent server channelConsume to process the bulk of AMQP
messages.  Any error from methods on this receiver will render the receiver
invalid and a new Channel should be opened.

---

ROGER NOTE: Unlike the normal channels, roger channels will automatically reconnect on
all errors until Channel.Close() is called.
*/
func (conn *Connection) Channel() (*Channel, error) {
	transportChan := &transportChannel{
		BasicChannel:       nil,
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

// Test returns a ConnectionTesting object with a number of helper methods for testing
// Connection objects.
func (conn *Connection) Test(t *testing.T) *ConnectionTesting {
	blocks := int32(0)
	return &ConnectionTesting{
		conn: conn,
		TransportTesting: TransportTesting{
			t:       t,
			manager: conn.transportManager,
			blocks:  &blocks,
		},
	}
}

// newConnection gets a new roger connection for a given url and connection config.
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
		BasicConnection: nil,
		dialURL:         url,
		dialConfig:      config,
		streadwayConfig: streadwayConfig,
	}

	manager := newTransportManager(
		context.Background(), transportConn, "CONNECTION",
	)
	// Use the logger from the config
	manager.logger = config.Logger

	conn := &Connection{
		transportConn:    transportConn,
		transportManager: manager,
	}

	return conn
}
