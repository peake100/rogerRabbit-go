package amqp

import (
	"crypto/tls"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

// Config is used in DialConfig and Open to specify the desired tuning
// parameters used during a connection open handshake.  The negotiated tuning
// will be stored in the returned connection's Config field.
//
// ---
//
// ROGER NOTE: This config type is a re-implementation of streadway/amqp.Config. We
// any code that can declare such a config will work with this type. In the future this
// type may add additional options for rogerRabbit-go/amqp.
type Config struct {
	// The SASL mechanisms to try in the client request, and the successful
	// mechanism used on the Connection object.
	// If SASL is nil, PlainAuth from the URL is used.
	SASL []Authentication

	// Vhost specifies the namespace of permissions, exchanges, queues and
	// bindings on the server. Dial sets this to the path parsed from the URL.
	Vhost string

	ChannelMax int           // 0 max channels means 2^16 - 1
	FrameSize  int           // 0 max bytes means unlimited
	Heartbeat  time.Duration // less than 1s uses the server's interval

	// TLSClientConfig specifies the client configuration of the TLS connection
	// when establishing a tls livesOnce.
	// If the URL uses an amqps scheme, then an empty tls.Config with the
	// ServerName from the URL is used.
	TLSClientConfig *tls.Config

	// Properties is table of properties that the client advertises to the server.
	// This is an optional setting - if the application does not set this,
	// the underlying library will use a generic set of client properties.
	Properties Table

	// Connection locale that we expect to always be en_US
	// Even though servers must return it as per the AMQP 0-9-1 spec,
	// we are not aware of it being used other than to satisfy the spec requirements
	Locale string

	// Dial returns a net.Conn prepared for a TLS handshake with TSLClientConfig,
	// then an AMQP connection handshake.
	// If Dial is nil, net.DialTimeout with a 30s connection and 30s deadline is
	// used during TLS and AMQP handshaking.
	Dial func(network, addr string) (net.Conn, error)

	// If set to true, the default handlers will not be registered on channels created
	// through the associated connection.
	NoDefaultMiddleware bool

	// ConnectionMiddleware holds middleware to add to connection method and event
	// handlers.
	ConnectionMiddleware ConnectionMiddlewares

	// ChannelMiddleware holds middleware to add to channel method and event handlers.
	ChannelMiddleware ChannelMiddlewares

	// The logger to use for internal logging. If none, the default zerolog logger will
	// be used.
	Logger zerolog.Logger
}

// DefaultConfig returns the default config for Dial() as it is in the streadway
// application.
func DefaultConfig() Config {
	return Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
		Logger:    log.Logger,
	}
}
