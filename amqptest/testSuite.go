//revive:disable:import-shadowing

package amqptest

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
	"testing"
	"time"
)

type InnerSuite interface {
	Assertions
	suite.TestingSuite
}

// ChannelSuiteOpts is used to configure AmqpSuite, which can be embedded
// into a testify suite.Suite to gain a number of useful testing methods.
type ChannelSuiteOpts struct {
	dialAddress   string
	dialConfig    amqp.Config
	dialConfigSet bool
}

// WithDialAddress configures the address to dial for our test connections.
// Default: amqp://localhost:57018
func (opts *ChannelSuiteOpts) WithDialAddress(amqpURI string) *ChannelSuiteOpts {
	opts.dialAddress = amqpURI
	return opts
}

// WithDialConfig sets the amqp.Config object to use when dialing the test brocker.
// Default: amqp.DefaultConfig()
func (opts *ChannelSuiteOpts) WithDialConfig(config amqp.Config) *ChannelSuiteOpts {
	opts.dialConfig = config
	opts.dialConfigSet = true
	return opts
}

// NewChannelSuiteOpts returns a new ChannelSuiteOpts with default values and the logger
// set to debug.
func NewChannelSuiteOpts() *ChannelSuiteOpts {
	config := amqp.DefaultConfig()
	config.DefaultLoggerLevel = zerolog.DebugLevel
	return new(ChannelSuiteOpts).
		WithDialAddress(TestDialAddress).
		WithDialConfig(config)
}

// AmqpSuite Embed into other suite types to have a connection and channel
// automatically set up for testing on suite start, and closed on suite shutdown, as
// well as a number of other helper methods for interacting with a test broker and
// handling test setup / teardown.
type AmqpSuite struct {
	// Suite is the embedded suite type.
	InnerSuite

	// Opts is our Options object and can be set on suite instantiation or during setup.
	Opts *ChannelSuiteOpts

	// connConsume holds a connection to be used for consuming tests.
	connConsume *amqp.Connection
	// channelConsume holds a channel to be used for consuming tests.
	channelConsume *amqp.Channel

	// connPublish holds a connection to be used for publication tests.
	connPublish *amqp.Connection
	// channelPublish holds a channel to be used for publication tests.
	channelPublish *amqp.Channel
}

func (amqpSuite *AmqpSuite) SetupTest() {
	amqpSuite.T().Cleanup(func() {
		// Flushing Stdout may help with race conditions at the end of a test.
		os.Stdout.Sync()
	})
}

// SetupSuite implements, suite.SetupAllSuite, and sets suite.Opts to
// NewChannelSuiteOpts if no other opts has been provided.
func (amqpSuite *AmqpSuite) SetupSuite() {
	if amqpSuite.Opts == nil {
		amqpSuite.Opts = NewChannelSuiteOpts()
	}

	if setupSuite, ok := amqpSuite.InnerSuite.(suite.SetupAllSuite); ok {
		setupSuite.SetupSuite()
	}
}

// TearDownSuite implements suite.TearDownAllSuite, and closes all open test connections
// and channels created form this suite's helper methods.
func (amqpSuite *AmqpSuite) TearDownSuite() {
	if amqpSuite.connConsume != nil {
		defer amqpSuite.connConsume.Close()
	}

	if amqpSuite.connPublish != nil {
		defer amqpSuite.connPublish.Close()
	}

	if amqpSuite.channelConsume != nil {
		defer amqpSuite.channelConsume.Close()
	}
	if amqpSuite.channelPublish != nil {
		defer amqpSuite.channelPublish.Close()
	}

	if tearDownSuite, ok := amqpSuite.InnerSuite.(suite.TearDownAllSuite); ok {
		tearDownSuite.TearDownSuite()
	}
}

// dialConnection dials the test connection.
func (amqpSuite *AmqpSuite) dialConnection() *amqp.Connection {
	address := amqpSuite.Opts.dialAddress
	if address == "" {
		address = TestDialAddress
	}

	config := amqpSuite.Opts.dialConfig
	if !amqpSuite.Opts.dialConfigSet {
		config = amqp.DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := amqp.DialConfigCtx(ctx, address, config)
	if err != nil {
		amqpSuite.T().Errorf("error dialing connection: %v", err)
		amqpSuite.T().FailNow()
	}

	return conn
}

// getChannel gets a new channel from conn.
func (amqpSuite *AmqpSuite) getChannel(conn *amqp.Connection) *amqp.Channel {
	channel, err := conn.Channel()
	if err != nil {
		amqpSuite.T().Errorf("error getting channel: %v", err)
		amqpSuite.T().FailNow()
	}

	return channel
}

// ConnConsume returns an *amqp.Connection to be used for consuming methods. The
// returned connection object will be the same each time this methods is called.
func (amqpSuite *AmqpSuite) ConnConsume() *amqp.Connection {
	if amqpSuite.connConsume != nil {
		return amqpSuite.connConsume
	}
	amqpSuite.connConsume = amqpSuite.dialConnection()
	return amqpSuite.connConsume
}

// ChannelConsume returns an *amqp.Channel to be used for consuming methods. The
// returned channel object will be the same each time this methods is called, until
// ReplaceChannels is called.
func (amqpSuite *AmqpSuite) ChannelConsume() *amqp.Channel {
	if amqpSuite.channelConsume != nil {
		return amqpSuite.channelConsume
	}
	amqpSuite.channelConsume = amqpSuite.getChannel(amqpSuite.ConnConsume())
	return amqpSuite.channelConsume
}

// ChannelConsumeTester returns *amqp.ChannelTesting object for ChannelConsume with the
// current suite.T().
func (amqpSuite *AmqpSuite) ChannelConsumeTester() *amqp.ChannelTesting {
	return amqpSuite.ChannelConsume().Test(amqpSuite.T())
}

// ConnPublish returns an *amqp.Connection to be used for publishing methods. The
// returned connection object will be the same each time this methods is called.
func (amqpSuite *AmqpSuite) ConnPublish() *amqp.Connection {
	if amqpSuite.connPublish != nil {
		return amqpSuite.connPublish
	}
	amqpSuite.connPublish = amqpSuite.dialConnection()
	return amqpSuite.connPublish
}

// ChannelPublish returns an *amqp.Channel to be used for consuming methods. The
// returned channel object will be the same each time this methods is called, until
// ReplaceChannels is called.
func (amqpSuite *AmqpSuite) ChannelPublish() *amqp.Channel {
	if amqpSuite.channelPublish != nil {
		return amqpSuite.channelPublish
	}
	amqpSuite.channelPublish = amqpSuite.getChannel(amqpSuite.ConnConsume())
	return amqpSuite.channelPublish
}

// ChannelConsumeTester returns *amqp.ChannelTesting object for ChannelConsume with the
// current suite.T().
func (amqpSuite *AmqpSuite) ChannelPublishTester() *amqp.ChannelTesting {
	return amqpSuite.channelPublish.Test(amqpSuite.T())
}

// ReplaceChannels replace and close the current ChannelConsume and ChannelPublish for
// fresh ones. Most helpfully used as a cleanup function when persisting the current
// channels to the next test method is not desirable.
func (amqpSuite *AmqpSuite) ReplaceChannels() {
	if amqpSuite.channelConsume != nil {
		_ = amqpSuite.channelConsume.Close()
	}

	channel, err := amqpSuite.connConsume.Channel()
	if !amqpSuite.NoError(err, "open new channelConsume for amqpSuite") {
		amqpSuite.FailNow(
			"could not open channelConsume for test amqpSuite",
		)
	}

	amqpSuite.channelConsume = channel

	if amqpSuite.channelPublish != nil {
		_ = amqpSuite.channelPublish.Close()
	}
	channel, err = amqpSuite.connConsume.Channel()
	if !amqpSuite.NoError(err, "open new channelPublish for amqpSuite") {
		amqpSuite.FailNow(
			"could not open channelPublish for test amqpSuite",
		)
	}

	amqpSuite.channelPublish = channel
}

func (amqpSuite *AmqpSuite) createSingleTestQueue(
	name string,
	exchange string,
	exchangeKey string,
	cleanup bool,
	channel *amqp.Channel,
) amqp.Queue {
	queue, err := channel.QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)

	if !amqpSuite.NoError(err, "create queue") {
		amqpSuite.T().FailNow()
	}

	if cleanup {
		amqpSuite.T().Cleanup(func() {
			_, _ = channel.QueueDelete(
				name, false, false, false,
			)
		})
	}
	if exchange == "" || exchangeKey == "" {
		return queue
	}

	err = channel.QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !amqpSuite.NoError(err, "error binding queue") {
		amqpSuite.T().FailNow()
	}

	_, err = channel.QueuePurge(queue.Name, false)
	if !amqpSuite.NoError(err, "error purging queue") {
		amqpSuite.T().FailNow()
	}

	return queue
}

// CreateTestQueue creates a basic test queue on both ChannelConsume and ChannelPublish.
// The created queue can be optionally bound to an exchange (which should have been
// previously created with CreateTestExchange).
//
// If cleanup is true, then a cleanup function will be registered on the current
// suite.T() to delete the queues at the end of the test.
func (amqpSuite *AmqpSuite) CreateTestQueue(
	name string, exchange string, exchangeKey string, cleanup bool,
) amqp.Queue {
	queue := amqpSuite.createSingleTestQueue(
		name, exchange, exchangeKey, cleanup, amqpSuite.ChannelPublish(),
	)
	_ = amqpSuite.createSingleTestQueue(
		name, exchange, exchangeKey, cleanup, amqpSuite.ChannelConsume(),
	)

	return queue
}

// CreateTestExchange creates a basic test exchange on both test channels.
//
// If cleanup is set to true, a cleanup function will be registered with the current
// suite.T() that will delete the exchange at the end of the test.
func (amqpSuite *AmqpSuite) CreateTestExchange(
	name string, kind string, cleanup bool,
) {
	err := amqpSuite.channelPublish.ExchangeDeclare(
		name,
		kind,
		false,
		false,
		false,
		false,
		nil,
	)

	if !amqpSuite.NoError(err, "create publish queue") {
		amqpSuite.T().FailNow()
	}

	if cleanup {
		amqpSuite.T().Cleanup(func() {
			_ = amqpSuite.channelPublish.ExchangeDelete(
				name, false, false,
			)
		})
	}

	err = amqpSuite.channelConsume.ExchangeDeclare(
		name,
		kind,
		false,
		false,
		false,
		false,
		nil,
	)

	if !amqpSuite.NoError(err, "create consume queue") {
		amqpSuite.T().FailNow()
	}

	if cleanup {
		amqpSuite.T().Cleanup(func() {
			_ = amqpSuite.channelConsume.ExchangeDelete(
				name, false, false,
			)
		})
	}
}

func (amqpSuite *AmqpSuite) publishMessagesSend(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	channel := amqpSuite.ChannelPublish()
	for i := 0; i < count; i++ {
		msg := strconv.Itoa(i)
		err := channel.Publish(
			exchange,
			key,
			true,
			false,
			amqp.Publishing{
				MessageId: msg,
				Body:      []byte(msg),
			},
		)
		if !assert.NoErrorf(err, "publish %v", i) {
			t.FailNow()
		}
	}
}

func (amqpSuite *AmqpSuite) publishMessagesConfirm(
	ctx context.Context,
	t *testing.T,
	count int,
	confirmations <-chan amqp.Confirmation,
	returns <-chan amqp.Return,
	allConfirmed chan error,
) {
	assert := assert.New(t)
	defer close(allConfirmed)

	confirmCount := 0
	for {
		select {
		case confirmation := <-confirmations:
			if !assert.Truef(confirmation.Ack, "message %v acked", confirmCount) {
				allConfirmed <- fmt.Errorf("message %v nacked", confirmCount)
				return
			}
			confirmCount++
		case <-returns:
			t.Error("message returned by broker")
			allConfirmed <- errors.New("message returned by broker")
			return
		case <-ctx.Done():
			return
		}
		if confirmCount == count {
			return
		}
	}
}

// PublishMessages publishes messages on the given exchange and Key, waits for
// confirmations, then returns. Test is failed immediately if any of these steps fails.
//
// Messages bodies are simply the index of the message starting at 0, so publishing
// 3 messages would result with a message bodies '0', '1', and '2'.
func (amqpSuite *AmqpSuite) PublishMessages(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	err := amqpSuite.ChannelPublish().Confirm(false)
	if !assert.NoError(err, "put publish channel into confirm mode") {
		t.FailNow()
	}

	confirmations := make(chan datamodels.Confirmation, count)
	amqpSuite.ChannelPublish().NotifyPublish(confirmations)
	returns := make(chan amqp.Return)
	amqpSuite.ChannelPublish().NotifyReturn(returns)

	// Listen for confirmations in a routine.
	allConfirmed := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond*time.Duration(count))
	defer cancel()
	go amqpSuite.publishMessagesConfirm(ctx, t, count, confirmations, returns, allConfirmed)

	amqpSuite.publishMessagesSend(t, exchange, key, count)

	select {
	case err = <-allConfirmed:
	case <-ctx.Done():
		t.Error("publish confirmations timed out")
		t.FailNow()
	}

	if !assert.NoError(err, "all messages confirmed") {
		t.FailNow()
	}
}

// GetMessage gets a single message, failing the test immediately if there is not a
// message waiting or the get message fails.
func (amqpSuite *AmqpSuite) GetMessage(
	queueName string, autoAck bool,
) datamodels.Delivery {
	delivery, ok, err := amqpSuite.ChannelConsume().Get(queueName, autoAck)
	if !amqpSuite.NoError(err, "get message") {
		amqpSuite.T().FailNow()
	}

	if !amqpSuite.True(ok, "message was fetched") {
		amqpSuite.T().FailNow()
	}

	return delivery
}

func NewAmqpSuite(innerSuite InnerSuite, opts *ChannelSuiteOpts) AmqpSuite {
	return AmqpSuite{
		InnerSuite:     innerSuite,
		Opts:           opts,
		connConsume:    nil,
		channelConsume: nil,
		connPublish:    nil,
		channelPublish: nil,
	}
}
