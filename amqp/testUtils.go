//revive:disable:import-shadowing

package amqp

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

const (
	TestAddress = "amqp://localhost:57018"
)

// Get a new connection for testing.
func GetTestConnection(t *testing.T) *Connection {
	assert := assert.New(t)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()

	conn, err := DialCtx(dialCtx, TestAddress)
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

// Embed into other suite types to have a connection and channel automatically set
// up for testing on suite start, and closed on suite shutdown.
type ChannelSuiteBase struct {
	suite.Suite

	ConnConsume    *Connection
	ChannelConsume *Channel

	ConnPublish    *Connection
	ChannelPublish *Channel
}

// replace and close the current ChannelConsume for a fresh one.
func (suite *ChannelSuiteBase) ReplaceChannels() {
	if suite.ChannelConsume != nil {
		_ = suite.ChannelConsume.Close()
	}

	channel, err := suite.ConnConsume.Channel()
	if !suite.NoError(err, "open new ChannelConsume for suite") {
		suite.FailNow("could not open ChannelConsume for test suite")
	}

	suite.ChannelConsume = channel

	if suite.ChannelPublish != nil {
		_ = suite.ChannelPublish.Close()
	}
	channel, err = suite.ConnConsume.Channel()
	if !suite.NoError(err, "open new ChannelPublish for suite") {
		suite.FailNow("could not open ChannelPublish for test suite")
	}

	suite.ChannelPublish = channel
}

// Creates a basic test queue on both test channels, and deletes registers them for
// deletion at the end of the test. The created queue can be optionally bound to an
// exchange (which should have been previously created with CreateTestExchange).
func (suite *ChannelSuiteBase) CreateTestQueue(
	name string, exchange string, exchangeKey string,
) Queue {
	_, err := suite.ChannelPublish.QueueDeclare(
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
		_, _ = suite.ChannelPublish.QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	queue, err := suite.ChannelConsume.QueueDeclare(
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
		_, _ = suite.ChannelConsume.QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	if exchange == "" || exchangeKey == "" {
		return queue
	}

	err = suite.ChannelPublish.QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !suite.NoError(err, "error binding publish queue") {
		suite.T().FailNow()
	}

	err = suite.ChannelConsume.QueueBind(
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
	err := suite.ChannelPublish.ExchangeDeclare(
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
		_ = suite.ChannelPublish.ExchangeDelete(
			name, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	err = suite.ChannelConsume.ExchangeDeclare(
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
		_ = suite.ChannelConsume.ExchangeDelete(
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
		err := suite.ChannelPublish.Publish(
			exchange,
			key,
			true,
			false,
			Publishing{
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
	confirmationEvents <-chan data.Confirmation,
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

	err := suite.ChannelPublish.Confirm(false)
	if !assert.NoError(err, "put publish channel into confirm mode") {
		t.FailNow()
	}

	confirmationEvents := make(chan data.Confirmation, count)
	suite.ChannelPublish.NotifyPublish(confirmationEvents)

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
func (suite *ChannelSuiteBase) GetMessage(queueName string, autoAck bool) data.Delivery {
	delivery, ok, err := suite.ChannelConsume.Get(queueName, autoAck)
	if !suite.NoError(err, "get message") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "message was fetched") {
		suite.T().FailNow()
	}

	return delivery
}

func (suite *ChannelSuiteBase) SetupSuite() {
	// Get the test connection we are going to use for all of our tests
	suite.ConnConsume = GetTestConnection(suite.T())
	if suite.T().Failed() {
		suite.FailNow("could not get consumer connection")
	}

	suite.ConnPublish = GetTestConnection(suite.T())
	if suite.T().Failed() {
		suite.FailNow("could not get publisher connection")
	}

	channel, err := suite.ConnConsume.Channel()
	if !suite.NoError(err, "open ChannelConsume for testing") {
		suite.FailNow("failed to open test ChannelConsume")
	}

	suite.ChannelConsume = channel

	channel, err = suite.ConnConsume.Channel()
	if !suite.NoError(err, "open ChannelPublish for testing") {
		suite.FailNow("failed to open test ChannelPublish")
	}

	suite.ChannelPublish = channel
}

func (suite *ChannelSuiteBase) TearDownSuite() {
	if suite.ConnConsume != nil {
		defer suite.ConnConsume.Close()
	}

	if suite.ConnPublish != nil {
		defer suite.ConnPublish.Close()
	}

	if suite.ChannelConsume != nil {
		defer suite.ChannelConsume.Close()
	}
	if suite.ChannelPublish != nil {
		defer suite.ChannelPublish.Close()
	}
}
