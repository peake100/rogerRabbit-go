package amqp

import (
	"context"
	"fmt"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

// Embed into other suite types to have a connection and channel automatically set
// up for testing on suite start, and closed on suite shutdown.
type ChannelSuiteBase struct {
	suite.Suite

	connConsume    *Connection
	channelConsume *Channel

	connPublish    *Connection
	channelPublish *Channel
}

// replace and close the current channelConsume for a fresh one.
func (suite *ChannelSuiteBase) replaceChannels() {
	if suite.channelConsume != nil {
		suite.channelConsume.Close()
	}
	channel, err := suite.connConsume.Channel()
	if !suite.NoError(err, "open new channelConsume for suite") {
		suite.FailNow("could not open channelConsume for test suite")
	}

	suite.channelConsume = channel

	if suite.channelPublish != nil {
		suite.channelPublish.Close()
	}
	channel, err = suite.connConsume.Channel()
	if !suite.NoError(err, "open new channelPublish for suite") {
		suite.FailNow("could not open channelPublish for test suite")
	}

	suite.channelPublish = channel
}

// Creates a basic test queue on both test channels, and deletes registers them for
// deletion at the end of the test. The created queue can be optionally bound to an
// exchange (which should have been previously created with createTestExchange).
func (suite *ChannelSuiteBase) createTestQueue(
	name string, exchange string, exchangeKey string,
) Queue {
	_, err := suite.channelPublish.QueueDeclare(
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
		suite.channelPublish.QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	queue, err := suite.channelConsume.QueueDeclare(
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
		suite.channelConsume.QueueDelete(
			name, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	if exchange == "" || exchangeKey == "" {
		return queue
	}

	err = suite.channelPublish.QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !suite.NoError(err, "error binding publish queue") {
		suite.T().FailNow()
	}

	err = suite.channelConsume.QueueBind(
		queue.Name, exchangeKey, exchange, false, nil,
	)
	if !suite.NoError(err, "error binding consume queue") {
		suite.T().FailNow()
	}

	return queue
}

// Creates a basic test exchange on both test channels, and deletes registers them for
// deletion at the end of the test
func (suite *ChannelSuiteBase) createTestExchange(name string, kind string) {
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
		suite.channelPublish.ExchangeDelete(
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
		suite.channelConsume.ExchangeDelete(
			name, false, false,
		)
	}
	suite.T().Cleanup(cleanup)
}

// Published messages on the given exchange and key, waits for confirmations, then
// returns. Test is failed immediately if any of these steps fails.
func (suite *ChannelSuiteBase) publishMessages(exchange string, key string, count int) {
	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put publish channel into confirm mode") {
		suite.T().FailNow()
	}

	confirmationEvents := make(chan Confirmation, count)
	suite.channelPublish.NotifyPublish(confirmationEvents)

	go func() {
		for i := 0 ; i < count ; i++ {
			err := suite.channelPublish.Publish(
				exchange,
				key,
				true,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("%v", i)),
				},
			)

			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
		}
	}()

	allConfirmed := make(chan struct{})
	go func() {
		confirmCount := 0
		for confirmation := range confirmationEvents {
			if !suite.Truef(confirmation.Ack, "message %v acked", confirmCount) {
				suite.T().FailNow()
			}
			confirmCount++
			if confirmCount == count {
				close(allConfirmed)
			}
		}
	}()

	select {
	case <- allConfirmed:
	case <- time.NewTimer(100 * time.Millisecond * time.Duration(count)).C:
		suite.T().Error("publish confirmations timed out")
		suite.T().FailNow()
	}
}

// get a single message, failing the test immediately if there is not a message waiting
// or the get message fails.
func (suite *ChannelSuiteBase) getMessage(queueName string, autoAck bool) Delivery {
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
	fmt.Println("SETTING UP SUITE")
	// Get the test connection we are going to use for all of our tests
	suite.connConsume = getTestConnection(suite.T())
	if suite.T().Failed() {
		suite.FailNow("could not get consumer connection")
	}

	suite.connPublish = getTestConnection(suite.T())
	if suite.T().Failed() {
		suite.FailNow("could not get publisher connection")
	}

	channel, err := suite.connConsume.Channel()
	if !suite.NoError(err, "open channelConsume for testing") {
		suite.FailNow("failed to open test channelConsume")
	}

	suite.channelConsume = channel

	channel, err = suite.connConsume.Channel()
	if !suite.NoError(err, "open channelPublish for testing") {
		suite.FailNow("failed to open test channelPublish")
	}

	suite.channelPublish = channel
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

// This suite tests the basic lifetime of a roger Connection object, including
// reconnects and final closure.
//
// We are going to use a test suite so we can re-use the same connection for multiple
// tests instead of having to re-dial the broker each time.
//
// Some of these tests need to be run in order. Testify runs suite tests in alphabetical
// order, so we are going to number them in the order they need to run.
type ChannelLifetimeSuite struct {
	ChannelSuiteBase
}

func (suite *ChannelLifetimeSuite) Test0010_GetChannel() {
	channel, err := suite.connConsume.Channel()
	if !suite.NoError(err, "get channelConsume") {
		// Fail the entire suite if we cannot get the channelConsume.
		suite.FailNow("failed to fetch channelConsume.")
	}

	// Stash this channelConsume for future tests.
	suite.channelConsume = channel

	if !suite.NotNil(channel, "channelConsume is not nil") {
		suite.FailNow("channelConsume was nil")
	}

	if !suite.NotNil(
		channel.transportChannel, "underlying channelConsume is not nil",
	) {
		suite.FailNow("underlying channelConsume was nil")
	}

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = channel.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.NoError(err, "queue created.")
}

func (suite *ChannelLifetimeSuite) Test0020_Reestablish_ChannelClose() {
	// Cache the current channelConsume
	currentChan := suite.channelConsume.transportChannel.Channel

	// Close the channelConsume
	suite.channelConsume.transportChannel.Close()

	// Wait for reestablish
	waitForReconnect(suite.T(), suite.channelConsume.transportManager, 2)

	suite.channelConsume.transportLock.Lock()
	suite.channelConsume.transportLock.Unlock()

	suite.False(
		suite.connConsume.transportConn.IsClosed(), "connection is open",
	)

	suite.NotSame(
		currentChan,
		suite.channelConsume.transportChannel.Channel,
		"channelConsume was replaced",
	)

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err := suite.channelConsume.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.NoError(err, "create queue")
}

func (suite *ChannelLifetimeSuite) Test0030_Reestablish_ConnectionClose() {
	// Cache the current channelConsume
	currentChan := suite.channelConsume.transportChannel.Channel
	connCount := suite.connConsume.reconnectCount

	// Close the connection
	suite.connConsume.transportConn.Close()

	// Wait for reestablish
	waitForReconnect(suite.T(), suite.channelConsume.transportManager, connCount+1)

	// try and see if the connection is open 10 times
	wasOpen := false
	for i := 0; i < 10; i++ {
		suite.channelConsume.transportLock.RLock()
		wasOpen = !suite.connConsume.transportConn.IsClosed()
		suite.channelConsume.transportLock.RUnlock()
		if wasOpen {
			break
		}
		time.Sleep(1 * time.Second / 5)
	}
	suite.True(
		wasOpen, "connection is open",
	)

	suite.NotSame(
		currentChan,
		suite.channelConsume.transportChannel.Channel,
		"channelConsume was replaced",
	)

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err := suite.channelConsume.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.NoError(err, "create queue")
}

func (suite *ChannelLifetimeSuite) Test0040_Close() {
	err := suite.channelConsume.Close()
	suite.NoError(err, "close channelConsume")

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = suite.channelConsume.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.ErrorIs(
		err, streadway.ErrClosed, "closed error on closed channelConsume",
	)
}

func (suite *ChannelLifetimeSuite) Test0050_CloseAgain_Err() {
	err := suite.channelConsume.Close()
	suite.ErrorIs(
		err, streadway.ErrClosed, "closed error on closed channelConsume",
	)
}

func (suite *ChannelLifetimeSuite) Test0060_NewChannel() {
	channel, err := suite.connConsume.Channel()
	suite.NoError(err, "get channelConsume")
	suite.channelConsume = channel

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = suite.channelConsume.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.NoError(
		err, "declare queue",
	)
}

// Test that closing the robust connection also permanently closes the channelConsume.
func (suite *ChannelLifetimeSuite) Test0070_CloseConnection_ClosesChannel() {
	err := suite.connConsume.Close()
	suite.NoError(err, "close connection")

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = suite.channelConsume.transportChannel.QueueDeclare(
		"test queue",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.ErrorIs(
		err, streadway.ErrClosed, "declare queue returns closed err",
	)
	suite.ErrorIs(
		suite.connConsume.ctx.Err(),
		context.Canceled,
		"channelConsume context is cancelled",
	)
}

func TestChannelLifetime(t *testing.T) {
	suite.Run(t, new(ChannelLifetimeSuite))
}

// Suite for testing channelConsume methods.
type ChannelMethodsSuite struct {
	ChannelSuiteBase
}

func (suite *ChannelMethodsSuite) Test0010_QueueDeclare() {
	queue, err := suite.channelConsume.QueueDeclare(
		"test_channel_methods",
		false,
		true,
		true,
		false,
		nil,
	)

	suite.NoError(err, "declare queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0020_QueueInspect() {
	queue, err := suite.channelConsume.QueueInspect("test_channel_methods")

	suite.NoError(err, "inspect queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0030_QueueInspect_Err() {
	_, err := suite.channelConsume.QueueInspect("not-a-real-queue")
	suite.Error(err, "error inspecting queue")
	suite.EqualError(err, "Exception (404) Reason: \"NOT_FOUND - no queue"+
		" 'not-a-real-queue' in vhost '/'\"")
}

// Since this test is being done after we got a channel error, we are also implicitly
// testing that we have recovered from the error.
func (suite *ChannelMethodsSuite) Test0040_QueueInspect_AfterErr() {
	queue, err := suite.channelConsume.QueueInspect("test_channel_methods")

	suite.NoError(err, "inspect queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0050_Publish() {
	err := suite.channelConsume.Publish(
		"",
		"test_channel_methods",
		true,
		false,
		Publishing{
			Headers:         nil,
			ContentType:     "plain/text",
			ContentEncoding: "",
			DeliveryMode:    0,
			Priority:        0,
			Body:            []byte("test message 1"),
		},
	)

	suite.NoError(err, "publish message")
}

func (suite *ChannelMethodsSuite) Test0060_Get() {
	suite.T().Cleanup(suite.replaceChannels)

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	var message Delivery
	var ok bool
	var err error

	for {
		message, ok, err = suite.channelConsume.Get(
			"test_channel_methods", true,
		)
		if !suite.NoError(err, "get message") {
			suite.T().FailNow()
		}

		if ok {
			break
		}

		select {
		case <-timer.C:
			suite.T().Error("get message timed out")
			suite.T().FailNow()
		default:
		}
	}

	messageBody := string(message.Body)
	suite.T().Log("MESSAGE BODY:", messageBody)
	suite.Equal("test message 1", messageBody)
}

func (suite *ChannelMethodsSuite) Test0070_Consume_Basic() {
	suite.T().Cleanup(suite.replaceChannels)

	queue, err := suite.channelConsume.QueueDeclare(
		"test_channel_methods",
		false,
		true,
		true,
		false,
		nil,
	)
	if !suite.NoError(err, "declare queue") {
		suite.T().FailNow()
	}

	messageChannel, err := suite.channelConsume.Consume(
		queue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "consume queue") {
		suite.T().FailNow()
	}

	for i := 1; i <= 10; i++ {
		suite.NoError(err, "set up consumer")

		message := fmt.Sprintf("test consumer %v", i)

		err = suite.channelPublish.Publish(
			"",
			"test_channel_methods",
			true,
			false,
			Publishing{
				Headers:         nil,
				ContentType:     "plain/text",
				ContentEncoding: "",
				DeliveryMode:    0,
				Priority:        0,
				Body:            []byte(message),
			},
		)
		suite.T().Logf("published message %v", i)
		suite.NoErrorf(err, "publish message %v", i)

		timeout := time.NewTimer(3 * time.Second)
		defer timeout.Stop()

		select {
		case thisMessage := <-messageChannel:
			suite.T().Logf(
				"RECEIVED MESSAGE '%v' with deivery tag %v",
				string(thisMessage.Body),
				thisMessage.DeliveryTag,
			)
			suite.Equalf(
				string(thisMessage.Body), message, "message %v correct", i,
			)
			suite.Equalf(uint64(i), thisMessage.DeliveryTag, "delivery tag")
		case <-timeout.C:
			suite.T().Errorf("timeout on consumer receive message %v", i)
			suite.T().FailNow()
		}
	}

	suite.T().Log("closing channelConsume")

	// Close the channelConsume and see if it closes our consumer
	err = suite.channelConsume.Close()
	if !suite.NoError(err, "close channelConsume") {
		suite.T().FailNow()
	}

	timeout := time.NewTimer(3 * time.Second)

	select {
	case _, ok := <-messageChannel:
		suite.False(ok, "consumer channelConsume is closed")
	case <-timeout.C:
		suite.T().Errorf("timeout on consumer channeel close")
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0080_Consume_OverDisconnect_Channel() {
	suite.T().Cleanup(suite.replaceChannels)
	defer func() {
		_, err := suite.channelPublish.transportChannel.QueueDelete(
			"disconnect_consumer_test",
			false,
			false,
			true,
		)
		suite.NoError(err, "remove test queue")
	}()

	queue, err := suite.channelConsume.QueueDeclare(
		"disconnect_consumer_test",
		false,
		false,
		true,
		false,
		nil,
	)
	if !suite.NoError(err, "declare queue") {
		suite.T().FailNow()
	}

	queue, err = suite.channelPublish.QueueDeclare(
		"disconnect_consumer_test",
		false,
		false,
		true,
		false,
		nil,
	)
	if !suite.NoError(err, "declare queue") {
		suite.T().FailNow()
	}

	messageChannel, err := suite.channelConsume.Consume(
		queue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "consume queue") {
		suite.T().FailNow()
	}

	for i := 1; i <= 10; i++ {
		suite.NoError(err, "set up consumer")

		message := fmt.Sprintf("test consumer %v", i)

		err = suite.channelPublish.Publish(
			"",
			queue.Name,
			true,
			false,
			Publishing{
				Headers:         nil,
				ContentType:     "plain/text",
				ContentEncoding: "",
				DeliveryMode:    0,
				Priority:        0,
				Body:            []byte(message),
			},
		)

		suite.T().Logf("published message %v", i)
		if !suite.NoErrorf(err, "publish message %v", i) {
			suite.T().FailNow()
		}

		timeout := time.NewTimer(3 * time.Second)
		defer timeout.Stop()

		select {
		case thisMessage, ok := <-messageChannel:
			if !suite.True(ok, "message channel open") {
				suite.T().FailNow()
			}
			suite.T().Logf(
				"RECEIVED MESSAGE '%v' with deivery tag %v",
				string(thisMessage.Body),
				thisMessage.DeliveryTag,
			)
			suite.Equalf(
				string(thisMessage.Body), message, "message %v correct", i,
			)
			suite.Equalf(uint64(i), thisMessage.DeliveryTag, "delivery tag")
		case <-timeout.C:
			suite.T().Errorf("timeout on consumer receive message %v", i)
			suite.T().FailNow()
		}

		// Force close either the channelConsume or the connection
		if i%3 == 0 {
			suite.T().Log("closing connection")
			suite.connConsume.transportConn.Close()
		} else if i%2 == 0 {
			suite.T().Logf("closing channel")
			suite.channelConsume.transportChannel.Close()
		}
	}

	// Close the channelConsume and see if it closes our consumer
	err = suite.channelConsume.Close()
	if !suite.NoError(err, "close channelConsume") {
		suite.T().FailNow()
	}

	timeout := time.NewTimer(3 * time.Second)

	select {
	case _, ok := <-messageChannel:
		suite.False(ok, "consumer channel is closed")
	case <-timeout.C:
		suite.T().Errorf("timeout on consumer channel close")
		suite.T().FailNow()
	}
}

// The last test set up a queue that is not a
func (suite *ChannelMethodsSuite) Test0090_QueueDelete() {
	queueName := "queue_delete_test"
	_, err := suite.channelPublish.QueueDeclare(
		queueName, false,
		false,
		true,
		false,
		nil,
	)

	suite.NoError(err, "declare queue")

	deleteCount, err := suite.channelPublish.QueueDelete(
		queueName, false, false, true,
	)
	suite.NoError(err, "delete queue")
	suite.Equalf(0, deleteCount, "0 messages deleted")

	_, err = suite.channelPublish.QueueInspect(queueName)
	suite.Error(err, "inspect error")

	var streadwayErr *streadway.Error
	suite.ErrorAs(err, &streadwayErr, "err is streadway")
	suite.Equal(404, streadwayErr.Code, "error code is not found")
	suite.Equal(
		"NOT_FOUND - no queue 'queue_delete_test' in vhost '/'",
		streadwayErr.Reason,
		"error message",
	)
}

func (suite *ChannelMethodsSuite) Test0100_QueueDeclarePassive() {
	queueName := "passive_declare_test"
	// Cleanup the test by deleting this queue and not checking the error.
	cleanup := func() {
		suite.channelPublish.QueueDelete(
			queueName, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	_, err := suite.channelPublish.QueueDeclare(
		queueName, false,
		false,
		true,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	_, err = suite.channelPublish.QueueDeclarePassive(
		queueName, false,
		false,
		true,
		false,
		nil,
	)
	suite.NoError(err, "declare queue passive")
}

func (suite *ChannelMethodsSuite) Test0110_QueueDeclarePassive_Err() {
	// Replace channels at the end so the next test has a clean slate.
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "passive_declare_test"
	_, err := suite.channelPublish.QueueDeclarePassive(
		queueName, false,
		false,
		true,
		false,
		nil,
	)
	suite.Error(
		err,
		"error should occur on passive declare of a non-existent queue",
	)

	var streadwayErr *Error
	suite.ErrorAs(err, &streadwayErr, "error is streadway error type")
	suite.Equal(404, streadwayErr.Code, "error code")
	suite.Equal(
		"NOT_FOUND - no queue 'passive_declare_test' in vhost '/'",
		streadwayErr.Reason,
	)
}

// Tests that queues are automatically re-declared on re-connect
func (suite *ChannelMethodsSuite) Test0120_QueueDeclare_RedeclareAfterDisconnect() {
	queueName := "auto_redeclare_declare_test"
	// Cleanup the test by deleting this queue and not checking the error.
	cleanup := func() {
		suite.channelPublish.QueueDelete(
			queueName, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	// Declare the queue
	_, err := suite.channelPublish.QueueDeclare(
		queueName,
		false,
		false,
		true,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	// grab the channel transport lock so our channel cannot reopen immediately.
	suite.channelPublish.transportLock.Lock()
	// close the channel manually
	suite.channelPublish.transportChannel.Channel.Close()

	// Use the other connection to delete the queue
	_, err = suite.channelConsume.QueueDelete(
		queueName, false, false, false,
	)
	suite.NoError(err, "delete queue")

	// release our lock on the original channel so it reconnects
	suite.channelPublish.transportLock.Unlock()

	// Check and see if the queue was re-declared
	info, err := suite.channelPublish.QueueInspect(queueName)
	suite.NoError(err, "inspect queue")
	suite.Equal(info.Name, queueName, "check name")
}

// Tests that queues are no re-declared if deleted on re-connect
func (suite *ChannelMethodsSuite) Test0130_QueueDeclare_NoRedeclareAfterDelete() {
	queueName := "no_redeclare_test"
	// Cleanup the test by deleting this queue and not checking the error.
	cleanup := func() {
		suite.channelPublish.QueueDelete(
			queueName, false, false, false,
		)
		suite.replaceChannels()
	}
	suite.T().Cleanup(cleanup)

	// Declare the queue
	_, err := suite.channelPublish.QueueDeclare(
		queueName,
		false,
		false,
		true,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	// Verify it was created
	_, err = suite.channelPublish.QueueInspect(queueName)
	suite.NoError(err, "inspect queue")

	// Delete the queue
	_, err = suite.channelPublish.QueueDelete(
		queueName, false, false, true,
	)
	suite.NoError(err, "delete queue")

	// close the channel manually, forcing a re-connect
	suite.channelPublish.transportChannel.Channel.Close()

	// Check and see if the queue was re-declared
	_, err = suite.channelPublish.QueueInspect(queueName)
	suite.Error(err, "inspect queue")
	var streadwayErr *Error
	suite.ErrorAs(err, &streadwayErr, "is streadway err")
	suite.Equal(404, streadwayErr.Code, "error code is not found")
	suite.Equal(
		"NOT_FOUND - no queue 'no_redeclare_test' in vhost '/'",
		streadwayErr.Reason,
		"error message is correct",
	)

}

func (suite *ChannelMethodsSuite) Test0140_NotifyPublish_Basic() {
	// Replace channels at the end since we are enabling confirmation mode
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "notify_publish_basic"
	suite.createTestQueue(queueName, "", "")

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	notifyPublish := make(chan Confirmation, publishCount)
	suite.channelPublish.NotifyPublish(notifyPublish)

	allReceived := make(chan struct{})

	workersDone := new(sync.WaitGroup)
	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		for i := 0; i < publishCount; i++ {
			err := suite.channelPublish.Publish(
				"",
				queueName,
				true,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)
			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
		}
	}()

	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		defer close(allReceived)
		i := 0
		for confirmation := range notifyPublish {
			suite.Equal(
				uint64(i)+1, confirmation.DeliveryTag, "delivery tag",
			)
			suite.True(confirmation.Ack, "server acked")
			suite.False(confirmation.DisconnectOrphan)
			i++
			if i >= 10 {
				break
			}
		}
	}()

	select {
	case <-allReceived:
	case <-time.NewTimer(5 * time.Second).C:
		suite.T().Error("confirmations timeout")
	}

	err = suite.channelPublish.Close()
	suite.NoError(err, "close channel")

	select {
	case _, ok := <-notifyPublish:
		suite.False(ok, "confirmation channel closed")
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("ack timeout")
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0150_NotifyPublish_Reconnections() {
	// Replace channels at the end since we are enabling confirmation mode
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "notify_publish_basic"
	suite.createTestQueue(queueName, "", "")

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	notifyPublish := make(chan Confirmation, publishCount)
	suite.channelPublish.NotifyPublish(notifyPublish)

	allReceived := make(chan struct{})

	confirmations := new(sync.WaitGroup)

	workersDone := new(sync.WaitGroup)
	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		for i := 0; i < publishCount; i++ {
			err := suite.channelPublish.Publish(
				"",
				queueName,
				true,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i+1)),
				},
			)
			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
			// Add 1 to the confirmation WaitGroup
			confirmations.Add(1)

			// Every second and third publish, wait for our confirmations to line up
			// and close the channel or connection to force a reconnection
			if i%3 == 0 {
				suite.T().Log("closing connection")
				confirmations.Wait()
				suite.connPublish.transportConn.Close()
			} else if i%2 == 0 {
				suite.T().Log("closing channel")
				confirmations.Wait()
				suite.channelPublish.transportChannel.Close()
			}
		}
	}()

	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		defer close(allReceived)
		i := 0
		for confirmation := range notifyPublish {
			// Subtract 1 to the confirmation WaitGroup
			confirmations.Done()
			suite.Equal(
				uint64(i)+1, confirmation.DeliveryTag, "delivery tag",
			)
			suite.True(confirmation.Ack, "server acked")
			suite.False(confirmation.DisconnectOrphan)
			i++
			if i >= 10 {
				break
			}
		}
	}()

	select {
	case <-allReceived:
	case <-time.NewTimer(5 * time.Second).C:
		suite.T().Error("confirmations timeout")
	}

	err = suite.channelPublish.Close()
	suite.NoError(err, "close channel")

	select {
	case _, ok := <-notifyPublish:
		suite.False(ok, "confirmation channel closed")
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("ack timeout")
		suite.T().FailNow()
	}
}

// We're only going to test the basic notify confirms flow since it's built on top of
// NotifyPublish, and we test that with reconnections anyway.
func (suite *ChannelMethodsSuite) Test0160_NotifyConfirm() {
	// Replace channels at the end since we are enabling confirmation mode
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "notify_confirms_basic"
	suite.createTestQueue(queueName, "", "")

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	ackEvents, nackEvents := make(chan uint64, publishCount),
		make(chan uint64, publishCount)

	suite.channelPublish.NotifyConfirm(ackEvents, nackEvents)

	confirmations := new(sync.WaitGroup)

	go func() {
		for i := 0; i < publishCount; i++ {
			err := suite.channelPublish.Publish(
				"",
				queueName,
				true,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i+1)),
				},
			)
			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
			// Add 1 to the confirmation WaitGroup
			confirmations.Add(1)
		}
	}()

	received := make([]int, publishCount)
	receivedConfirm := make(chan struct{}, publishCount)

	go func() {
		for tag := range ackEvents {
			received[tag-1] = publishCount
			receivedConfirm <- struct{}{}
		}
	}()

	go func() {
		for tag := range nackEvents {
			received[tag-1] = publishCount
			receivedConfirm <- struct{}{}
		}
	}()

	timer := time.NewTimer(5 * time.Second)
	receivedCount := 0
	for {
		select {
		case <-receivedConfirm:
			receivedCount++
		case <-timer.C:
			suite.T().Error("timeout receiving confirmations")
			suite.T().FailNow()
		}

		if receivedCount >= 10 {
			break
		}
	}

	err = suite.channelPublish.Close()
	suite.NoError(err, "close channel")

	select {
	case _, ok := <-ackEvents:
		suite.False(ok, "ack channel closed")
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("ack close timeout")
		suite.T().FailNow()
	}

	select {
	case _, ok := <-nackEvents:
		suite.False(ok, "nack channel closed")
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("nack close timeout")
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0170_NotifyReturn() {
	suite.T().Cleanup(suite.replaceChannels)
	// We're going to publish to a queue that does not exist. This will cause a
	// delivery return to occur
	queueName := "test_notify_return"

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	// In order to force returns, we are going to send to a queue that does not exist.

	publishCount := 10

	go func() {
		for i := 0 ; i < publishCount ; i++	{
			err := suite.channelPublish.Publish(
				"",
				queueName,
				true,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)
			suite.NoError(err, "publish message %v", i)
		}
	}()

	returnEvents := suite.channelPublish.NotifyReturn(make(chan Return, publishCount))
	returnsComplete := make(chan struct{})
	returns := make([]Return, 0, publishCount)
	go func() {
		defer close(returnsComplete)

		returnCount := 0
		for thisReturn := range returnEvents {
			returns = append(returns, thisReturn)
			returnCount++
			if returnCount >= 10 {
				return
			}
		}
	}()

	select {
	case <- returnsComplete:
	case <-time.NewTimer(5 * time.Second).C:
		suite.T().Error("returns received timeout")
		suite.T().FailNow()
	}

	// close the channel
	suite.channelPublish.Close()

	_, open := <- returnEvents
	suite.False(open, "return event channel should be closed.")

	// make sure the returns we got are all of the expected messages.
mainLoop:
	for i := 0 ; i < publishCount ; i++ {
		expectedMessage := fmt.Sprintf("message %v", i)
		for _, returned := range returns {
			if string(returned.Body) == expectedMessage {
				continue mainLoop
			}
		}
		suite.T().Errorf(
			"did not receive returned message '%v'", expectedMessage,
		)
	}
}

func (suite *ChannelMethodsSuite) Test0180_NotifyConfirmOrOrphaned() {
	suite.T().Cleanup(suite.replaceChannels)

	ackQueueName := "test_confirm_ack"
	nackQueueName := "test_confirm_nack"

	// Set up the ack queue
	suite.createTestQueue(ackQueueName, "", "")

	publishCount := 10

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put publish chan into confirm mode") {
		suite.T().FailNow()
	}

	eventsAck := make(chan uint64, publishCount)
	eventsNack := make(chan uint64, publishCount)
	eventsOrphan := make(chan uint64, publishCount)

	suite.channelPublish.NotifyConfirmOrOrphaned(
		eventsAck, eventsNack, eventsOrphan,
	)

	published := new(sync.WaitGroup)
	allPublished := make(chan struct{})
	receivedCount := 0
	ackCount := 0
	nackCount := 0
	orphanCount := 0

	go func() {
		defer close(allPublished)
		for i := 0 ; i < publishCount ; i++ {
			publishType := "ACK"
			if i % 3 == 0 {
				publishType = "NACK"
			} else if i % 2 == 0 {
				publishType = "ORPHAN"
			}

			queueName := ackQueueName
			if i % 3 != 0 && i % 2 != 0 {
				queueName = nackQueueName
			}

			published.Wait()
			published.Add(1)
			err := suite.channelPublish.Publish(
				"",
				queueName,
				true,
				// RabbitMQ does not support immediate, this will result in a channel
				// crash that will cause this message to be orphaned.
				publishType == "ORPHAN",
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)

			suite.NoErrorf(err, "publish message %v", i)
		}
	}()

	go func() {
		for _ = range eventsAck {
			ackCount++
			receivedCount++
			published.Done()
		}
	}()

	go func() {
		for _ = range eventsNack {
			nackCount++
			receivedCount++
			published.Done()
		}
	}()

	go func() {
		for _ = range eventsOrphan {
			orphanCount++
			receivedCount++
			published.Done()
		}
	}()

	select {
	case <- allPublished:
	case <- time.NewTimer(5 * time.Second).C:
	}

	published.Wait()

	suite.Equal(
		publishCount, receivedCount, "received 10 confirmations",
	)
	suite.Equal(
		7, ackCount, "received 5 acks",
	)
	suite.Equal(
		3, orphanCount, "received 5 orphans",
	)
}

func (suite *ChannelMethodsSuite) Test0190_QueuePurge() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "test_queue_purge"
	suite.createTestQueue(queueName, "", "")

	err := suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	publishCount := 2
	notifyPublish := make(chan Confirmation, publishCount)

	suite.channelPublish.NotifyPublish(notifyPublish)

	// publish 2 messages and wait for acks
	go func() {
		for i := 0 ; i < publishCount ; i++ {
			err := suite.channelPublish.Publish(
				"",
				queueName,
				false,
				false,
				Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)
			if !suite.NoErrorf(err, "publish message %v", i) {
				suite.T().FailNow()
			}
		}
	}()

	confirmationsReceived := make(chan struct{})

	go func() {
		defer close(confirmationsReceived)
		timer := time.NewTimer(0)

		for i := 0 ; i < publishCount ; i++ {
			timer.Reset(3 * time.Second)

			select {
			case confirmation := <- notifyPublish:
				suite.Truef(confirmation.Ack, "message %v acked", i)
			case <- timer.C:
				suite.T().Error("confirmation timeout")
				suite.T().FailNow()
			}
		}
	}()

	select {
	case <- confirmationsReceived:
	case <- time.NewTimer(3 * time.Second).C:
		suite.T().Error("confirmations timeout")
		suite.T().FailNow()
	}

	// Check that 2 messages get purged
	count, err := suite.channelPublish.QueuePurge(queueName, false)
	if !suite.NoError(err, "purge queue") {
		suite.T().FailNow()
	}

	suite.Equal(2, count, "2 messages purged")
}

func (suite *ChannelMethodsSuite) Test0200_ExchangeDeclare() {
	err := suite.channelPublish.ExchangeDeclare(
		"test_exchange_basic",
		ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "declare exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0210_ExchangeDeclarePassive() {
	err := suite.channelPublish.ExchangeDeclarePassive(
		"test_exchange_basic",
		ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.NoError(err, "declare passive exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0220_ExchangeDelete() {
	err := suite.channelPublish.ExchangeDelete(
		"test_exchange_basic",
		false,
		false,
	)

	if !suite.NoError(err, "delete exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0230_ExchangeDeclarePassive_Err() {
	suite.T().Cleanup(suite.replaceChannels)

	err := suite.channelPublish.ExchangeDeclarePassive(
		"test_exchange_basic",
		ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)

	if !suite.Error(err, "declare passive non-existent exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0240_QueueBind() {
	exchangeName := "test_exchange_bind"
	queueName := "test_queue_name"
	suite.createTestExchange(exchangeName, ExchangeDirect)
	suite.createTestQueue(queueName, "", "")

	err := suite.channelPublish.QueueBind(
		queueName, queueName, exchangeName, false, nil,
	)
	if !suite.NoError(err, "bind queue") {
		suite.T().FailNow()
	}

	err = suite.channelPublish.Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	confirmations := make(chan Confirmation, 5)
	suite.channelPublish.NotifyPublish(confirmations)

	// lets test publishing and getting a message on the exchange
	err = suite.channelPublish.Publish(
		exchangeName,
		queueName,
		false,
		false,
		Publishing{
			Body: []byte("some message"),
		},
	)
	if !suite.NoError(err, "error publishing message") {
		suite.T().FailNow()
	}

	select {
	case confirmation := <- confirmations:
		if !suite.True(confirmation.Ack, "publishing confirmed") {
			suite.T().FailNow()
		}
	case <- time.NewTimer(3 * time.Second).C:
		suite.T().Error("publish confirmation timed out")
		suite.T().FailNow()
	}

	_, ok, err := suite.channelPublish.Get(queueName, false)
	if !suite.NoError(err, "error getting message") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "message was fetched") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0250_QueueUnbind() {
	exchangeName := "test_queue_unbind_exchange"
	suite.createTestExchange(exchangeName, ExchangeDirect)

	queueName := "test_queue_unbind_queue"
	suite.createTestQueue(queueName, exchangeName, queueName)

	suite.T().Log("GOT HERE")

	// unbind the queue
	err := suite.channelPublish.QueueUnbind(
		queueName, queueName, exchangeName, nil,
	)

	if !suite.NoError(err, "unbind queue") {
		suite.T().FailNow()
	}
}

// TODO: write better tests for exchange bind and unbind
func (suite *ChannelMethodsSuite) Test0260_ExchangeBindUnbind() {
	suite.T().Cleanup(suite.replaceChannels)

	exchangeName1 := "test_exchange_bind1"
	suite.createTestExchange(exchangeName1, ExchangeDirect)

	exchangeName2 := "test_exchange_bind2"
	suite.createTestExchange(exchangeName2, ExchangeDirect)

	// unbind the queue
	err := suite.channelPublish.ExchangeBind(
		exchangeName2, "test_key", exchangeName1, false, nil,
	)

	if !suite.NoError(err, "bind exchange") {
		suite.T().FailNow()
	}

	// unbind the queue
	err = suite.channelPublish.ExchangeUnbind(
		exchangeName2, "test_key", exchangeName1, false, nil,
	)

	if !suite.NoError(err, "unbind exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0270_AckMessage() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message
	suite.publishMessages("", queueName, 1)

	delivery := suite.getMessage(queueName, false)

	err := delivery.Ack(false)
	if !suite.NoError(err, "ack message") {
		suite.T().FailNow()
	}

	// If the message was acked then the queue should be emtpy
	_, ok, err := suite.channelConsume.Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0280_NackMessage() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message
	suite.publishMessages("", queueName, 1)

	delivery := suite.getMessage(queueName, false)

	err := delivery.Nack(false, false)
	if !suite.NoError(err, "ack message") {
		suite.T().FailNow()
	}

	// If the message was nacked then the queue should be emtpy
	_, ok, err := suite.channelConsume.Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0290_NackMessage_Requeue() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message
	suite.publishMessages("", queueName, 1)

	delivery := suite.getMessage(queueName, false)

	err := delivery.Nack(false, true)
	if !suite.NoError(err, "nack and requeue message") {
		suite.T().FailNow()
	}

	// If the message was nacked then we should get the same message
	redelivery, ok, err := suite.channelConsume.Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "got message") {
		suite.T().FailNow()
	}

	suite.Equal(delivery.Body, redelivery.Body, "message is redelivery")
}

func (suite *ChannelMethodsSuite) Test0300_RejectMessage() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message
	suite.publishMessages("", queueName, 1)

	delivery := suite.getMessage(queueName, false)

	err := delivery.Reject(false)
	if !suite.NoError(err, "reject message") {
		suite.T().FailNow()
	}

	// If the message was acked then the queue should be emtpy
	_, ok, err := suite.channelConsume.Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0310_RejectMessage_Requeue() {
	suite.T().Cleanup(suite.replaceChannels)

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message
	suite.publishMessages("", queueName, 1)

	delivery := suite.getMessage(queueName, false)

	err := delivery.Reject(true)
	if !suite.NoError(err, "reject and requeue message") {
		suite.T().FailNow()
	}

	// If the message was nacked then we should get the same message
	redelivery, ok, err := suite.channelConsume.Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "got message") {
		suite.T().FailNow()
	}

	suite.Equal(delivery.Body, redelivery.Body, "message is redelivery")
}

func (suite *ChannelMethodsSuite) Test0320_Acknowledge_OrphanErr_Single() {
	type testCase struct {
		method string
	}

	// We're going to use a table test to test all three acknowledgement methods
	testCases := []testCase{
		{
			method: "ack",
		},
		{
			method: "nack",
		},
		{
			method: "reject",
		},
	}

	queueName := "queue_test_consume_ack"
	suite.createTestQueue(queueName, "", "")

	// Publish 1 message. This one message will keep getting redelivered on the force
	// reconnect, so we only need to publish it once
	suite.publishMessages("", queueName, 1)

	var thisCase testCase

	test := func(t *testing.T) {
		assert := assert.New(t)

		t.Cleanup(suite.replaceChannels)

		delivery := suite.getMessage(queueName, false)

		suite.channelConsume.Test(t).ForceReconnect(nil)

		var err error

		switch thisCase.method {
		case "ack":
			err = delivery.Ack(false)
		case "nack":
			err = delivery.Nack(false, false)
		case "reject":
			err = delivery.Reject( false)
		default:
			t.Errorf("incorrect method arg: %v", thisCase.method)
			t.FailNow()
		}
		if !assert.Error(err, "got acknowledgement error") {
			t.FailNow()
		}

		var orphanErr *ErrCantAcknowledgeOrphans
		if !assert.ErrorAs(
			err, &orphanErr, "error is ErrCantAcknowledgeOrphans",
		) {
			t.FailNow()
		}

		assert.Equal(
			uint64(1), orphanErr.OrphanTagFirst, "first orphan is 1",
		)
		assert.Equal(
			uint64(1), orphanErr.OrphanTagLast, "last orphan is 1",
		)

		assert.Equal(
			uint64(0), orphanErr.SuccessTagFirst, "first success is 0",
		)
		assert.Equal(
			uint64(0), orphanErr.SuccessTagLast, "last success is 0",
		)
	}

	for _, thisCase = range testCases {
		suite.T().Run(fmt.Sprintf("%v_method", thisCase.method), test)
	}
}

func TestChannelMethods(t *testing.T) {
	suite.Run(t, new(ChannelMethodsSuite))
}
