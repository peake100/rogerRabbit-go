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

// Get a new connection for testing.
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

type ChannelSuiteOpts struct {
	dialAddress string
	dialConfig  *amqp.Config
}

// Embed into other suite types to have a connection and channel automatically set
// up for testing on suite start, and closed on suite shutdown.
type ChannelSuiteBase struct {
	suite.Suite

	opts *ChannelSuiteOpts

	connConsume    *amqp.Connection
	channelConsume *amqp.Channel

	connPublish    *amqp.Connection
	channelPublish *amqp.Channel
}

func (suite *ChannelSuiteBase) dialConnection() *amqp.Connection {
	address := suite.opts.dialAddress
	if address == "" {
		address = TestDialAddress
	}

	config := suite.opts.dialConfig
	if config == nil {
		config = amqp.DefaultConfig()
	}

	conn, err := amqp.DialConfig(address, *config)
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

func (suite *ChannelSuiteBase) ConnConsume() *amqp.Connection {
	if suite.connConsume != nil {
		return suite.connConsume
	}
	suite.connConsume = suite.dialConnection()
	return suite.connConsume
}

func (suite *ChannelSuiteBase) ChannelConsume() *amqp.Channel {
	if suite.channelConsume != nil {
		return suite.channelConsume
	}
	suite.channelConsume = suite.getChannel(suite.ConnConsume())
	return suite.channelConsume
}

// Returns a channel tester for the consume channel using the current test.
func (suite *ChannelSuiteBase) ChannelConsumeTester() *amqp.ChannelTesting {
	return suite.ChannelConsume().Test(suite.T())
}

func (suite *ChannelSuiteBase) ConnPublish() *amqp.Connection {
	if suite.connPublish != nil {
		return suite.connPublish
	}
	suite.connPublish = suite.dialConnection()
	return suite.connPublish
}

func (suite *ChannelSuiteBase) ChannelPublish() *amqp.Channel {
	if suite.channelPublish != nil {
		return suite.channelPublish
	}
	suite.channelPublish = suite.getChannel(suite.ConnConsume())
	return suite.channelPublish
}

// Returns a channel tester for the consume channel using the current test.
func (suite *ChannelSuiteBase) ChannelPublishTester() *amqp.ChannelTesting {
	return suite.channelPublish.Test(suite.T())
}

// replace and close the current channelConsume for a fresh one.
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

// Creates a basic test queue on both test channels, and deletes registers them for
// deletion at the end of the test. The created queue can be optionally bound to an
// exchange (which should have been previously created with CreateTestExchange).
func (suite *ChannelSuiteBase) CreateTestQueue(
	name string, exchange string, exchangeKey string,
) amqp.Queue {
	_, err := suite.ChannelPublish().QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "create publish queue") {
		suite.T().FailNow()
	}

	cleanup := func() {
		_, _ = suite.ChannelPublish().QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	queue, err := suite.ChannelConsume().QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "create consume queue") {
		suite.T().FailNow()
	}

	cleanup = func() {
		_, _ = suite.ChannelConsume().QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	if exchange == "" || exchangeKey == "" {
		return queue
	}

	err = suite.ChannelPublish().QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !suite.NoError(err, "error binding publish queue") {
		suite.T().FailNow()
	}

	err = suite.ChannelConsume().QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !suite.NoError(err, "error binding consume queue") {
		suite.T().FailNow()
	}

	return queue
}

// Creates a basic test exchange on both test channels, and deletes registers them for
// deletion at the end of the test
func (suite *ChannelSuiteBase) CreateTestExchange(name string, kind string) {
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

	cleanup := func() {
		_ = suite.channelPublish.ExchangeDelete(
			name, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

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

	cleanup = func() {
		_ = suite.channelConsume.ExchangeDelete(
			name, false, false,
		)
	}
	suite.T().Cleanup(cleanup)
}

func (suite *ChannelSuiteBase) publishMessagesSend(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	for i := 0; i < count; i++ {
		err := suite.channelPublish.Publish(
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

// Published messages on the given exchange and Key, waits for confirmations, then
// returns. Test is failed immediately if any of these steps fails.
func (suite *ChannelSuiteBase) PublishMessages(
	t *testing.T, exchange string, key string, count int,
) {
	assert := assert.New(t)

	err := suite.channelPublish.Confirm(false)
	if !assert.NoError(err, "put publish channel into confirm mode") {
		t.FailNow()
	}

	confirmationEvents := make(chan dataModels.Confirmation, count)
	suite.channelPublish.NotifyPublish(confirmationEvents)

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

// get a single message, failing the test immediately if there is not a message waiting
// or the get message fails.
func (suite *ChannelSuiteBase) GetMessage(
	queueName string, autoAck bool,
) dataModels.Delivery {
	delivery, ok, err := suite.channelConsume.Get(queueName, autoAck)
	if !suite.NoError(err, "get message") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "message was fetched") {
		suite.T().FailNow()
	}

	return delivery
}

func (suite *ChannelSuiteBase) SetupSuite() {
	if suite.opts == nil {
		suite.opts = &ChannelSuiteOpts{
			dialAddress: TestDialAddress,
			dialConfig:  amqp.DefaultConfig(),
		}
	}
}

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
