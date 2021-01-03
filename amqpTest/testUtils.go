//revive:disable:import-shadowing

package amqpTest

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqp/dataModels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

const (
	TestDialAddress = "amqp://localhost:57018"
)

// GetTestConnection creates a new connection to amqp://localhost:57018, where our
// test broker will be listening.
func GetTestConnection(t *testing.T) *amqp.Connection {
	assert := assert.New(t)

	conn, err := amqp.DialCtx(context.Background(), TestDialAddress)
	if !assert.NoError(err, "dial connection") {
		t.FailNow()
	}

	if !assert.NotNil(conn, "connection is not nil") {
		t.FailNow()
	}

	t.Cleanup(
		func() {
			_ = conn.Close()
		},
	)

	return conn
}

// ChannelSuiteOpts is used to configure ChannelSuiteBase, which can be embedded
// into a testify suite.Suite to gain a number of useful testing methods.
type ChannelSuiteOpts struct {
	dialAddress string
	dialConfig  *amqp.Config
}

// WithDialAddress configures the address to dial for our test connections.
// Default: amqp://localhost:57018
func (opts *ChannelSuiteOpts) WithDialAddress(amqpURI string) *ChannelSuiteOpts {
	opts.dialAddress = amqpURI
	return opts
}

// WithDialConfig sets the amqp.Config object to use when dialing the test brocker.
// Default: amqp.DefaultConfig()
func (opts *ChannelSuiteOpts) WithDialConfig(config *amqp.Config) *ChannelSuiteOpts {
	opts.dialConfig = config
	return opts
}

// NewChannelSuiteOpts returns a new ChannelSuiteOpts with default values
func NewChannelSuiteOpts() *ChannelSuiteOpts {
	return new(ChannelSuiteOpts).
		WithDialAddress(TestDialAddress).
		WithDialConfig(amqp.DefaultConfig())
}

// ChannelSuiteBase Embed into other suite types to have a connection and channel
// automatically set up for testing on suite start, and closed on suite shutdown, as
// well as a number of other helper methods for interacting with a test broker and
// handling test setup / teardown.
type ChannelSuiteBase struct {
	suite.Suite

	Opts *ChannelSuiteOpts

	connConsume    *amqp.Connection
	channelConsume *amqp.Channel

	connPublish    *amqp.Connection
	channelPublish *amqp.Channel
}

// Dials the test connection.
func (suite *ChannelSuiteBase) dialConnection() *amqp.Connection {
	address := suite.Opts.dialAddress
	if address == "" {
		address = TestDialAddress
	}

	config := suite.Opts.dialConfig
	if config == nil {
		config = amqp.DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := amqp.DialConfigCtx(ctx, address, *config)
	if err != nil {
		suite.T().Errorf("error dialing connection: %v", err)
		suite.T().FailNow()
	}

	return conn
}

func (suite *ChannelSuiteBase) getChannel(conn *amqp.Connection) *amqp.Channel {
	channel, err := conn.Channel()
	if err != nil {
		suite.T().Errorf("error getting channel: %v", err)
		suite.T().FailNow()
	}

	return channel
}

// ConnConsume returns an *amqp.Connection to be used for consuming methods. The
// returned connection object will be the same each time this methods is called.
func (suite *ChannelSuiteBase) ConnConsume() *amqp.Connection {
	if suite.connConsume != nil {
		return suite.connConsume
	}
	suite.connConsume = suite.dialConnection()
	return suite.connConsume
}

// ChannelConsume returns an *amqp.Channel to be used for consuming methods. The
// returned channel object will be the same each time this methods is called, until
// ReplaceChannels is called.
func (suite *ChannelSuiteBase) ChannelConsume() *amqp.Channel {
	if suite.channelConsume != nil {
		return suite.channelConsume
	}
	suite.channelConsume = suite.getChannel(suite.ConnConsume())
	return suite.channelConsume
}

// ChannelConsumeTester returns *amqp.ChannelTesting object for ChannelConsume with the
// current suite.T().
func (suite *ChannelSuiteBase) ChannelConsumeTester() *amqp.ChannelTesting {
	return suite.ChannelConsume().Test(suite.T())
}

// ConnPublish returns an *amqp.Connection to be used for publishing methods. The
// returned connection object will be the same each time this methods is called.
func (suite *ChannelSuiteBase) ConnPublish() *amqp.Connection {
	if suite.connPublish != nil {
		return suite.connPublish
	}
	suite.connPublish = suite.dialConnection()
	return suite.connPublish
}

// ChannelPublish returns an *amqp.Channel to be used for consuming methods. The
// returned channel object will be the same each time this methods is called, until
// ReplaceChannels is called.
func (suite *ChannelSuiteBase) ChannelPublish() *amqp.Channel {
	if suite.channelPublish != nil {
		return suite.channelPublish
	}
	suite.channelPublish = suite.getChannel(suite.ConnConsume())
	return suite.channelPublish
}

// ChannelConsumeTester returns *amqp.ChannelTesting object for ChannelConsume with the
// current suite.T().
func (suite *ChannelSuiteBase) ChannelPublishTester() *amqp.ChannelTesting {
	return suite.channelPublish.Test(suite.T())
}

// ReplaceChannels replace and close the current ChannelConsume and ChannelPublish for
// fresh ones. Most helpfully used as a cleanup function when persisting the current
// channels to the next test method is not desirable.
func (suite *ChannelSuiteBase) ReplaceChannels() {
	if suite.channelConsume != nil {
		_ = suite.channelConsume.Close()
	}

	channel, err := suite.connConsume.Channel()
	if !suite.NoError(err, "open new channelConsume for suite") {
		suite.FailNow("could not open channelConsume for test suite")
	}

	suite.channelConsume = channel

	if suite.channelPublish != nil {
		_ = suite.channelPublish.Close()
	}
	channel, err = suite.connConsume.Channel()
	if !suite.NoError(err, "open new channelPublish for suite") {
		suite.FailNow("could not open channelPublish for test suite")
	}

	suite.channelPublish = channel
}

func (suite *ChannelSuiteBase) createSingleTestQueue(
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

	if !suite.NoError(err, "create queue") {
		suite.T().FailNow()
	}

	if cleanup {
		suite.T().Cleanup(func() {
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
	if !suite.NoError(err, "error binding queue") {
		suite.T().FailNow()
	}

	return queue
}

// CreateTestQueue creates a basic test queue on both ChannelConsume and ChannelPublish.
// The created queue can be optionally bound to an exchange (which should have been
// previously created with CreateTestExchange).
//
// If cleanup is true, then a cleanup function will be registered on the current
// suite.T() to delete the queues at the end of the test.
func (suite *ChannelSuiteBase) CreateTestQueue(
	name string, exchange string, exchangeKey string, cleanup bool,
) amqp.Queue {
	queue := suite.createSingleTestQueue(
		name, exchange, exchangeKey, cleanup, suite.ChannelPublish(),
	)
	_ = suite.createSingleTestQueue(
		name, exchange, exchangeKey, cleanup, suite.ChannelConsume(),
	)

	return queue
}

// CreateTestExchange creates a basic test exchange on both test channels.
//
// If cleanup is set to true, a cleanup function will be registered with the current
// suite.T() that will delete the exchange at the end of the test.
func (suite *ChannelSuiteBase) CreateTestExchange(
	name string, kind string, cleanup bool,
) {
	err := suite.channelPublish.ExchangeDeclare(
		name,
		kind,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "create publish queue") {
		suite.T().FailNow()
	}

	if cleanup {
		suite.T().Cleanup(func() {
			_ = suite.channelPublish.ExchangeDelete(
				name, false, false,
			)
		})
	}

	err = suite.channelConsume.ExchangeDeclare(
		name,
		kind,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "create consume queue") {
		suite.T().FailNow()
	}

	if cleanup {
		suite.T().Cleanup(func() {
			_ = suite.channelConsume.ExchangeDelete(
				name, false, false,
			)
		})
	}
}

func (suite *ChannelSuiteBase) publishMessagesSend(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	for i := 0; i < count; i++ {
		err := suite.ChannelPublish().Publish(
			exchange,
			key,
			true,
			false,
			amqp.Publishing{
				Body: []byte(fmt.Sprintf("%v", i)),
			},
		)

		if !assert.NoErrorf(err, "publish %v", i) {
			t.FailNow()
		}
	}
}

func (suite *ChannelSuiteBase) publishMessagesConfirm(
	t *testing.T,
	count int,
	confirmationEvents <-chan dataModels.Confirmation,
	allConfirmed chan struct{},
) {
	assert := assert.New(t)

	confirmCount := 0
	for confirmation := range confirmationEvents {
		if !assert.Truef(confirmation.Ack, "message %v acked", confirmCount) {
			t.FailNow()
		}
		confirmCount++
		if confirmCount == count {
			close(allConfirmed)
		}
	}
}

// PublishMessages publishes messages on the given exchange and Key, waits for
// confirmations, then returns. Test is failed immediately if any of these steps fails.
//
// Messages bodies are simply the index of the message starting at 0, so publishing
// 3 messages would result with a message bodies '0', '1', and '2'.
func (suite *ChannelSuiteBase) PublishMessages(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	err := suite.ChannelPublish().Confirm(false)
	if !assert.NoError(err, "put publish channel into confirm mode") {
		t.FailNow()
	}

	confirmationEvents := make(chan dataModels.Confirmation, count)
	suite.ChannelPublish().NotifyPublish(confirmationEvents)

	go suite.publishMessagesSend(t, exchange, key, count)

	allConfirmed := make(chan struct{})
	go suite.publishMessagesConfirm(t, count, confirmationEvents, allConfirmed)

	select {
	case <-allConfirmed:
	case <-time.NewTimer(100 * time.Millisecond * time.Duration(count)).C:
		t.Error("publish confirmations timed out")
		t.FailNow()
	}
}

// GetMessage gets a single message, failing the test immediately if there is not a
// message waiting or the get message fails.
func (suite *ChannelSuiteBase) GetMessage(
	queueName string, autoAck bool,
) dataModels.Delivery {
	delivery, ok, err := suite.ChannelConsume().Get(queueName, autoAck)
	if !suite.NoError(err, "get message") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "message was fetched") {
		suite.T().FailNow()
	}

	return delivery
}

// SetupSuite implements, suite.SetupAllSuite, and sets suite.Opts to
// NewChannelSuiteOpts if no other opts has been provided.
func (suite *ChannelSuiteBase) SetupSuite() {
	if suite.Opts == nil {
		suite.Opts = NewChannelSuiteOpts()
	}
}

// TearDownSuite implements suite.TearDownAllSuite, and closes all open test connections
// and channels created form this suite's helper methods.
func (suite *ChannelSuiteBase) TearDownSuite() {
	if suite.connConsume != nil {
		defer suite.connConsume.Close()
	}

	if suite.connPublish != nil {
		defer suite.connPublish.Close()
	}

	if suite.channelConsume != nil {
		defer suite.channelConsume.Close()
	}
	if suite.channelPublish != nil {
		defer suite.channelPublish.Close()
	}
}
