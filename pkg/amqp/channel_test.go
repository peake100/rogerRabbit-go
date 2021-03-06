//revive:disable

package amqp_test

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/defaultmiddlewares"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/internal"
	"github.com/peake100/rogerRabbit-go/pkg/amqptest"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

// This suite tests the basic lifetime of a roger Connection object, including
// reconnects and final closure.
//
// We are going to use a test suite so we can re-use the same connection for multiple
// tests instead of having to re-dial the broker each time.
//
// Some of these tests need to be run in order. Testify runs suite tests in alphabetical
// order, so we are going to number them in the order they need to run.
type ChannelLifetimeSuite struct {
	amqptest.AmqpSuite
}

func (suite *ChannelLifetimeSuite) Test0010_GetChannel() {
	channel, err := suite.ConnConsume().Channel()
	if !suite.NoError(err, "get channelConsume") {
		// Fail the entire suite if we cannot get the channelConsume.
		suite.FailNow("failed to fetch channelConsume.")
	}

	testing := channel.Test(suite.T())

	if !suite.NotNil(channel, "channelConsume is not nil") {
		suite.FailNow("channelConsume was nil")
	}

	if !suite.NotNil(
		testing.UnderlyingChannel(), "underlying channelConsume is not nil",
	) {
		suite.FailNow("underlying channelConsume was nil")
	}

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = testing.UnderlyingChannel().QueueDeclare(
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
	chanTesting := suite.ChannelConsumeTester()
	// Cache the current channelConsume
	currentChan := chanTesting.UnderlyingChannel()

	// Close the channelConsume
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	chanTesting.ForceReconnect(ctx)

	suite.False(
		chanTesting.UnderlyingConnection().IsClosed(),
		"connection is open",
	)

	suite.NotSame(
		currentChan,
		chanTesting.UnderlyingChannel(),
		"channelConsume was replaced",
	)

	// Check that the original channel gives an closed error
	_, err := currentChan.QueueInspect("some_queue")
	suite.ErrorIs(err, streadway.ErrClosed, "original channel is closed")

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = chanTesting.UnderlyingChannel().QueueDeclare(
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
	chanTest := suite.ChannelConsumeTester()

	currentChan := chanTest.UnderlyingChannel()
	chanReconnectSignal := chanTest.SignalOnReconnect()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	chanTest.ConnTest().ForceReconnect(ctx)

	// Wait for the reconnect to flow downstream to the channel.
	chanReconnectSignal.WaitOnReconnect(nil)

	suite.False(
		suite.ChannelConsume().IsClosed(), "connection is open",
	)

	suite.NotSame(
		currentChan,
		chanTest.UnderlyingChannel(),
		"channelConsume was replaced",
	)

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err := chanTest.UnderlyingChannel().QueueDeclare(
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
	chanTest := suite.ChannelConsumeTester()

	err := suite.ChannelConsume().Close()
	suite.NoError(err, "close channelConsume")

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = chanTest.UnderlyingChannel().QueueDeclare(
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
	suite.T().Cleanup(suite.ReplaceChannels)
	err := suite.ChannelConsume().Close()
	suite.ErrorIs(
		err, streadway.ErrClosed, "closed error on closed channelConsume",
	)
}

func (suite *ChannelLifetimeSuite) Test0060_NewChannel() {
	// To test if the channelConsume is open we are going to declare a queue on it.
	chanTester := suite.ChannelConsumeTester()

	_, err := chanTester.UnderlyingChannel().QueueDeclare(
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

// Test0065_OperationOverDisconnect tests that an operation will complete over
// a disconnect
func (suite *ChannelLifetimeSuite) Test0065_OperationOverDisconnect() {
	queueName := "OperationOverDisconnect"

	suite.ReplaceChannels()
	suite.CreateTestQueue(
		queueName,
		"",
		queueName,
		true,
	)

	tester := suite.ChannelPublish().Test(suite.T())

	// Block reconnections from occurring and disconnect the underlying transport so
	// that we attempt to fetch queue info during a disconnection event.
	tester.BlockReconnect()
	tester.DisconnectTransport()

	var queue amqp.Queue
	var err error

	// While the channel is disconnected, and reconnecting is blocked we are going to
	// try to inspect the queue. If it returns without an error, we'll know that we
	// successfully executed a command over a disconnect.

	// This channel will signal once the routine starts executing code
	running := make(chan struct{})
	// This channel will signal once the routine exits.
	complete := make(chan struct{})
	go func() {
		defer close(complete)
		close(running)
		queue, err = suite.ChannelPublish().QueueInspect(queueName)
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-running:
	case <-timer.C:
		tester.UnblockReconnect()
		suite.T().Error("timeout waiting for goroutine start")
		suite.T().FailNow()
	}

	// Wait 50ms for good measure, then unblock the reconnect.
	time.Sleep(50 * time.Millisecond)
	tester.UnblockReconnect()

	select {
	case <-complete:
	case <-timer.C:
		tester.UnblockReconnect()
		suite.T().Error("timeout waiting for goroutine close")
		suite.T().FailNow()
	}

	suite.NoError(err, "get queue info")
	suite.Equal(queueName, queue.Name, "correct queue info fetched")
}

// Test that closing the robust connection also permanently closes the channelConsume.
func (suite *ChannelLifetimeSuite) Test0070_CloseConnection_ClosesChannel() {
	err := suite.ConnConsume().Close()
	suite.NoError(err, "close connection")

	chanTester := suite.ChannelConsumeTester()

	// To test if the channelConsume is open we are going to declare a queue on it.
	_, err = chanTester.UnderlyingChannel().QueueDeclare(
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
	suite.True(
		suite.ChannelConsume().IsClosed(),
		context.Canceled,
		"channelConsume context is cancelled",
	)
}

func TestChannelLifetime(t *testing.T) {
	suite.Run(t, &ChannelLifetimeSuite{
		AmqpSuite: amqptest.NewAmqpSuite(new(suite.Suite), nil),
	})
}

// Suite for testing channelConsume methods.
type ChannelMethodsSuite struct {
	amqptest.AmqpSuite
}

func (suite *ChannelMethodsSuite) Test0010_QueueDeclare() {
	queue, err := suite.ChannelConsume().QueueDeclare(
		"test_channel_methods",
		false,
		true,
		false,
		false,
		nil,
	)

	suite.NoError(err, "declare queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0020_QueueInspect() {
	queue, err := suite.ChannelConsume().QueueInspect("test_channel_methods")

	suite.NoError(err, "inspect queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0030_QueueInspect_Err() {
	_, err := suite.ChannelConsume().QueueInspect("not-a-real-queue")
	suite.Error(err, "error inspecting queue")
	suite.EqualError(err, "Exception (404) Reason: \"NOT_FOUND - no queue"+
		" 'not-a-real-queue' in vhost '/'\"")
}

// Since this test is being done after we got a channel error, we are also implicitly
// testing that we have recovered from the error.
func (suite *ChannelMethodsSuite) Test0040_QueueInspect_AfterErr() {
	queue, err := suite.ChannelConsume().QueueInspect("test_channel_methods")

	suite.NoError(err, "inspect queue")
	suite.Equal(queue.Name, "test_channel_methods")
}

func (suite *ChannelMethodsSuite) Test0050_Publish() {
	err := suite.ChannelConsume().Publish(
		"",
		"test_channel_methods",
		true,
		false,
		amqp.Publishing{
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
	suite.T().Cleanup(suite.ReplaceChannels)

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	var message internal.Delivery
	var ok bool
	var err error

	for {
		message, ok, err = suite.ChannelConsume().Get(
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
	suite.T().Cleanup(suite.ReplaceChannels)

	queue, err := suite.ChannelConsume().QueueDeclare(
		"test_channel_methods",
		false,
		true,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "declare queue") {
		suite.T().FailNow()
	}

	messageChannel, err := suite.ChannelConsume().Consume(
		queue.Name,
		"",
		true,
		false,
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

		err = suite.ChannelPublish().Publish(
			"",
			"test_channel_methods",
			true,
			false,
			amqp.Publishing{
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
	err = suite.ChannelConsume().Close()
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
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "disconnect_consumer_test"
	suite.CreateTestQueue(queueName, "", "", true)

	chanTester := suite.ChannelConsumeTester()
	connTester := chanTester.ConnTest()

	messageChannel, err := suite.ChannelConsume().Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "consume queue") {
		suite.T().FailNow()
	}

	for i := 1; i <= 10; i++ {
		message := fmt.Sprintf("test consumer %v", i)

		err = suite.ChannelPublish().Publish(
			"",
			queueName,
			true,
			false,
			amqp.Publishing{
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
			connTester.DisconnectTransport()
		} else if i%2 == 0 {
			suite.T().Logf("closing channel")
			chanTester.DisconnectTransport()
		}
	}

	// Close the channelConsume and see if it closes our consumer
	err = suite.ChannelConsume().Close()
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
	_, err := suite.ChannelPublish().QueueDeclare(
		queueName, false,
		false,
		false,
		false,
		nil,
	)

	suite.NoError(err, "declare queue")

	deleteCount, err := suite.ChannelPublish().QueueDelete(
		queueName, false, false, true,
	)
	suite.NoError(err, "delete queue")
	suite.Equalf(0, deleteCount, "0 messages deleted")

	_, err = suite.ChannelPublish().QueueInspect(queueName)
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
		suite.ChannelPublish().QueueDelete(
			queueName, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	_, err := suite.ChannelPublish().QueueDeclare(
		queueName, false,
		false,
		false,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	_, err = suite.ChannelPublish().QueueDeclarePassive(
		queueName, false,
		false,
		false,
		false,
		nil,
	)
	suite.NoError(err, "declare queue passive")
}

func (suite *ChannelMethodsSuite) Test0110_QueueDeclarePassive_Err() {
	// Replace channels at the end so the next test has a clean slate.
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "passive_declare_test"
	_, err := suite.ChannelPublish().QueueDeclarePassive(
		queueName, false,
		false,
		false,
		false,
		nil,
	)
	suite.Error(
		err,
		"error should occur on passive declare of a non-existent queue",
	)

	var streadwayErr *amqp.Error
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
		suite.ChannelPublish().QueueDelete(
			queueName, false, false, false,
		)
	}
	suite.T().Cleanup(cleanup)

	// Declare the queue
	_, err := suite.ChannelPublish().QueueDeclare(
		queueName,
		false,
		// Set auto-delete to true for a forced re-connection
		true,
		false,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	chanTester := suite.ChannelPublishTester()

	// grab the channel livesOnce lock so our channel cannot reopen immediately.
	chanTester.BlockReconnect()
	chanTester.DisconnectTransport()

	// Use the other connection to delete the queue
	_, err = suite.ChannelConsume().QueueDelete(
		queueName, false, false, false,
	)
	suite.NoError(err, "delete queue")

	// Use the other connection to delete the queue
	_, err = suite.ChannelConsume().QueueInspect(queueName)
	suite.Error(err, "queue does not exist")

	chanTester.UnblockReconnect()

	// Check and see if the queue was re-declared
	info, err := suite.ChannelPublish().QueueInspect(queueName)
	suite.NoError(err, "inspect queue")
	suite.Equal(info.Name, queueName, "check Name")
}

// Tests that queues are no re-declared if deleted on re-connect
func (suite *ChannelMethodsSuite) Test0130_QueueDeclare_NoRedeclareAfterDelete() {
	queueName := "no_redeclare_test"
	// Cleanup the test by deleting this queue and not checking the error.
	cleanup := func() {
		suite.ChannelPublish().QueueDelete(
			queueName, false, false, false,
		)
		suite.ReplaceChannels()
	}
	suite.T().Cleanup(cleanup)

	chanTesting := suite.ChannelPublishTester()

	// Declare the queue
	_, err := suite.ChannelPublish().QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	suite.NoError(err, "declare queue")

	// Verify it was created
	_, err = suite.ChannelPublish().QueueInspect(queueName)
	suite.NoError(err, "inspect queue")

	// Delete the queue
	_, err = suite.ChannelPublish().QueueDelete(
		queueName, false, false, true,
	)
	suite.NoError(err, "delete queue")

	// close the channel manually, forcing a re-connect
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	chanTesting.ForceReconnect(ctx)

	// Check and see if the queue was re-declared
	_, err = suite.ChannelPublish().QueueInspect(queueName)
	suite.Error(err, "inspect queue")
	var streadwayErr *amqp.Error
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
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "notify_publish_basic"
	suite.CreateTestQueue(queueName, "", "", true)

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	notifyPublish := make(chan internal.Confirmation, publishCount)
	suite.ChannelPublish().NotifyPublish(notifyPublish)

	allReceived := make(chan struct{})

	workersDone := new(sync.WaitGroup)
	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		for i := 0; i < publishCount; i++ {
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				true,
				false,
				amqp.Publishing{
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

	err = suite.ChannelPublish().Close()
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
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "notify_publish_basic"
	suite.CreateTestQueue(queueName, "", "", true)

	chanTesting := suite.ChannelPublishTester()
	connTesting := chanTesting.ConnTest()

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	notifyPublish := make(chan internal.Confirmation, publishCount)
	suite.ChannelPublish().NotifyPublish(notifyPublish)

	allReceived := make(chan struct{})

	publications := make(chan struct{})
	confirmations := make(chan struct{})

	workersDone := new(sync.WaitGroup)
	workersDone.Add(1)
	go func() {
		defer workersDone.Done()
		for i := 0; i < publishCount; i++ {
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				true,
				false,
				amqp.Publishing{
					Body: []byte(fmt.Sprintf("message %v", i+1)),
				},
			)
			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
			// Add 1 to the confirmation WaitGroup
			publications <- struct{}{}
			<-confirmations

			// Every second and third publish, wait for our confirmations to line up
			// and close the channel or connection to force a reconnection
			if i%3 == 0 {
				suite.T().Log("closing connection")
				connTesting.DisconnectTransport()
			} else if i%2 == 0 {
				suite.T().Log("closing channel")
				chanTesting.DisconnectTransport()
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
			<-publications
			confirmations <- struct{}{}
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

	err = suite.ChannelPublish().Close()
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
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "notify_confirms_basic"
	suite.CreateTestQueue(queueName, "", "", true)

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put into confirmation mode") {
		suite.T().FailNow()
	}
	publishCount := 10
	ackEvents, nackEvents := make(chan uint64, publishCount),
		make(chan uint64, publishCount)

	suite.ChannelPublish().NotifyConfirm(ackEvents, nackEvents)

	go func() {
		for i := 0; i < publishCount; i++ {
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				true,
				false,
				amqp.Publishing{
					Body: []byte(fmt.Sprintf("message %v", i+1)),
				},
			)
			if !suite.NoErrorf(err, "publish %v", i) {
				suite.T().FailNow()
			}
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

		if receivedCount >= publishCount {
			break
		}
	}

	err = suite.ChannelPublish().Close()
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
	suite.T().Cleanup(suite.ReplaceChannels)
	// We're going to publish to a queue that does not exist. This will cause a
	// delivery return to occur
	queueName := "test_notify_return"

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	// In order to force returns, we are going to send to a queue that does not exist.

	publishCount := 10
	publishComplete := make(chan struct{})

	go func() {
		defer close(publishComplete)
		for i := 0; i < publishCount; i++ {
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				true,
				false,
				amqp.Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)
			suite.NoError(err, "publish message %v", i)
		}
	}()

	returnEvents := suite.
		ChannelPublish().
		NotifyReturn(make(chan amqp.Return, publishCount))

	returnsComplete := make(chan struct{})
	returns := make([]amqp.Return, 0, publishCount)
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

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case <-publishComplete:
	case <-timeout.C:
		suite.T().Error("publish complete timeout")
		suite.T().FailNow()
	}

	select {
	case <-returnsComplete:
	case <-timeout.C:
		suite.T().Error("returns received timeout")
		suite.T().FailNow()
	}

	// close the channel
	suite.ChannelPublish().Close()

	_, open := <-returnEvents
	suite.False(open, "return event channel should be closed.")

	// make sure the returns we got are all of the expected messages.
mainLoop:
	for i := 0; i < publishCount; i++ {
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
	suite.T().Cleanup(suite.ReplaceChannels)

	ackQueueName := "test_confirm_ack"
	nackQueueName := "test_confirm_nack"

	// Set up the ack queue
	suite.CreateTestQueue(ackQueueName, "", "", true)

	publishCount := 10

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put publish chan into confirm mode") {
		suite.T().FailNow()
	}

	eventsAck := make(chan uint64, publishCount)
	eventsNack := make(chan uint64, publishCount)
	eventsOrphan := make(chan uint64, publishCount)

	suite.ChannelPublish().NotifyConfirmOrOrphaned(
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
		for i := 0; i < publishCount; i++ {
			publishType := "ACK"
			if i%3 == 0 {
				publishType = "NACK"
			} else if i%2 == 0 {
				publishType = "ORPHAN"
			}

			queueName := ackQueueName
			if i%3 != 0 && i%2 != 0 {
				queueName = nackQueueName
			}

			published.Wait()
			published.Add(1)
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				true,
				// RabbitMQ does not support immediate, this will result in a channel
				// crash that will cause this message to be orphaned.
				publishType == "ORPHAN",
				amqp.Publishing{
					Body: []byte(fmt.Sprintf("message %v", i)),
				},
			)

			suite.T().Logf("PUBLISHED MESSAGE %v", i)
			suite.NoErrorf(err, "publish message %v", i)
		}
	}()

	go func() {
		for range eventsAck {
			ackCount++
			receivedCount++
			published.Done()
		}
	}()

	go func() {
		for range eventsNack {
			nackCount++
			receivedCount++
			published.Done()
		}
	}()

	go func() {
		for range eventsOrphan {
			orphanCount++
			receivedCount++
			published.Done()
		}
	}()

	select {
	case <-allPublished:
	case <-time.NewTimer(5 * time.Second).C:
		suite.T().Error("ERROR WAITING FOR MESSAGES")
		suite.T().FailNow()
	}

	published.Wait()

	suite.Equal(
		publishCount, receivedCount, "received 10 confirmations",
	)
	suite.Equal(
		7, ackCount, "received 7 acks",
	)
	suite.Equal(
		3, orphanCount, "received 3 orphans",
	)
}

func (suite *ChannelMethodsSuite) Test0190_QueuePurge() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_queue_purge"
	suite.CreateTestQueue(queueName, "", "", true)

	err := suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	publishCount := 2
	notifyPublish := make(chan internal.Confirmation, publishCount)

	suite.ChannelPublish().NotifyPublish(notifyPublish)

	// publish 2 messages and wait for acks
	go func() {
		for i := 0; i < publishCount; i++ {
			err := suite.ChannelPublish().Publish(
				"",
				queueName,
				false,
				false,
				amqp.Publishing{
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
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()

		for i := 0; i < publishCount; i++ {
			timer.Reset(3 * time.Second)

			select {
			case confirmation := <-notifyPublish:
				suite.Truef(confirmation.Ack, "message %v acked", i)
			case <-timer.C:
				suite.T().Error("confirmation timeout")
				suite.T().FailNow()
			}
		}
	}()

	select {
	case <-confirmationsReceived:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("confirmations timeout")
		suite.T().FailNow()
	}

	// Check that 2 messages get purged
	count, err := suite.ChannelPublish().QueuePurge(queueName, false)
	if !suite.NoError(err, "purge queue") {
		suite.T().FailNow()
	}

	suite.Equal(2, count, "2 messages purged")
}

func (suite *ChannelMethodsSuite) Test0200_ExchangeDeclare() {
	err := suite.ChannelPublish().ExchangeDeclare(
		"test_exchange_basic",
		amqp.ExchangeDirect,
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
	err := suite.ChannelPublish().ExchangeDeclarePassive(
		"test_exchange_basic",
		amqp.ExchangeDirect,
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
	err := suite.ChannelPublish().ExchangeDelete(
		"test_exchange_basic",
		false,
		false,
	)

	if !suite.NoError(err, "delete exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0230_ExchangeDeclarePassive_Err() {
	suite.T().Cleanup(suite.ReplaceChannels)

	err := suite.ChannelPublish().ExchangeDeclarePassive(
		"test_exchange_basic",
		amqp.ExchangeDirect,
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
	suite.CreateTestExchange(exchangeName, amqp.ExchangeDirect, true)
	suite.CreateTestQueue(queueName, "", "", true)

	err := suite.ChannelPublish().QueueBind(
		queueName, queueName, exchangeName, false, nil,
	)
	if !suite.NoError(err, "bind queue") {
		suite.T().FailNow()
	}

	err = suite.ChannelPublish().Confirm(false)
	if !suite.NoError(err, "put channel into confirm mode") {
		suite.T().FailNow()
	}

	confirmations := make(chan internal.Confirmation, 5)
	suite.ChannelPublish().NotifyPublish(confirmations)

	// lets test publishing and getting a message on the exchange
	err = suite.ChannelPublish().Publish(
		exchangeName,
		queueName,
		false,
		false,
		amqp.Publishing{
			Body: []byte("some message"),
		},
	)
	if !suite.NoError(err, "error publishing message") {
		suite.T().FailNow()
	}

	select {
	case confirmation := <-confirmations:
		if !suite.True(confirmation.Ack, "publishing confirmed") {
			suite.T().FailNow()
		}
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("publish confirmation timed out")
		suite.T().FailNow()
	}

	_, ok, err := suite.ChannelPublish().Get(queueName, false)
	if !suite.NoError(err, "error getting message") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "message was fetched") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0250_QueueUnbind() {
	exchangeName := "test_queue_unbind_exchange"
	suite.CreateTestExchange(exchangeName, amqp.ExchangeDirect, true)

	queueName := "test_queue_unbind_queue"
	suite.CreateTestQueue(queueName, exchangeName, queueName, true)

	suite.T().Log("GOT HERE")

	// unbind the queue
	err := suite.ChannelPublish().QueueUnbind(
		queueName, queueName, exchangeName, nil,
	)

	if !suite.NoError(err, "unbind queue") {
		suite.T().FailNow()
	}
}

// TODO: write better tests for exchange bind and unbind
func (suite *ChannelMethodsSuite) Test0260_ExchangeBindUnbind() {
	suite.T().Cleanup(suite.ReplaceChannels)

	exchangeName1 := "test_exchange_bind1"
	suite.CreateTestExchange(exchangeName1, amqp.ExchangeDirect, true)

	exchangeName2 := "test_exchange_bind2"
	suite.CreateTestExchange(exchangeName2, amqp.ExchangeDirect, true)

	// unbind the queue
	err := suite.ChannelPublish().ExchangeBind(
		exchangeName2, "test_key", exchangeName1, false, nil,
	)

	if !suite.NoError(err, "bind exchange") {
		suite.T().FailNow()
	}

	// unbind the queue
	err = suite.ChannelPublish().ExchangeUnbind(
		exchangeName2, "test_key", exchangeName1, false, nil,
	)

	if !suite.NoError(err, "unbind exchange") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0265_ExchangeRedeclareAfterDisconnect() {
	suite.T().Cleanup(suite.ReplaceChannels)

	exchangeName := "exchange_test_redeclare"
	cleanup := func() {
		suite.ChannelPublish().ExchangeDelete(
			exchangeName,
			false,
			false,
		)
	}
	suite.T().Cleanup(cleanup)

	// Declare an exchange that gets auto-deleted
	err := suite.ChannelPublish().ExchangeDeclare(
		exchangeName,
		amqp.ExchangeDirect,
		false,
		// Use AutoDelete to force a full re-declare after disconnect
		true,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "declare exchange") {
		suite.T().FailNow()
	}

	// Delete the exchange on the other connection
	err = suite.ChannelConsume().ExchangeDelete(
		exchangeName, false, false,
	)
	if !suite.NoError(err, "delete exchange") {
		suite.T().FailNow()
	}

	// Force a reconnectMiddleware
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	suite.ChannelPublish().Test(suite.T()).ForceReconnect(ctx)

	// Check that the exchange exists again
	err = suite.ChannelConsume().ExchangeDeclarePassive(
		exchangeName,
		amqp.ExchangeDirect,
		false,
		true,
		false,
		false,
		nil,
	)
	suite.NoError(err, "check for exchange after reconnectMiddleware")
}

func (suite *ChannelMethodsSuite) Test0270_AckMessage() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "queue_test_consume_ack"
	suite.CreateTestQueue(queueName, "", "", true)

	// Publish 1 message
	suite.PublishMessages(suite.T(), "", queueName, 1)

	delivery := suite.GetMessage(queueName, false)

	err := delivery.Ack(false)
	if !suite.NoError(err, "ack message") {
		suite.T().FailNow()
	}

	// If the message was acked then the queue should be empty.
	_, ok, err := suite.ChannelConsume().Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0280_NackMessage() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "queue_test_consume_nack"
	suite.CreateTestQueue(queueName, "", "", true)

	// Publish 1 message
	suite.PublishMessages(suite.T(), "", queueName, 1)

	delivery := suite.GetMessage(queueName, false)

	err := delivery.Nack(false, false)
	if !suite.NoError(err, "ack message") {
		suite.T().FailNow()
	}

	// If the message was nacked then the queue should be empty.
	_, ok, err := suite.ChannelConsume().Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0290_NackMessage_Requeue() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "queue_test_consume_nack_requeue"
	suite.CreateTestQueue(queueName, "", "", true)

	// Publish 1 message
	suite.PublishMessages(suite.T(), "", queueName, 1)

	delivery := suite.GetMessage(queueName, false)

	err := delivery.Nack(false, true)
	if !suite.NoError(err, "nack and requeue message") {
		suite.T().FailNow()
	}

	// If the message was nacked then we should get the same message
	redelivery, ok, err := suite.ChannelConsume().Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "got message") {
		suite.T().FailNow()
	}

	suite.Equal(delivery.Body, redelivery.Body, "message is redelivery")
}

func (suite *ChannelMethodsSuite) Test0300_RejectMessage() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "queue_test_consume_reject"
	suite.CreateTestQueue(queueName, "", "", true)

	// Publish 1 message
	suite.PublishMessages(suite.T(), "", queueName, 1)

	delivery := suite.GetMessage(queueName, false)

	err := delivery.Reject(false)
	if !suite.NoError(err, "reject message") {
		suite.T().FailNow()
	}

	// If the message was acked then the queue should be empty.
	_, ok, err := suite.ChannelConsume().Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.False(ok, "queue is empty") {
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0310_RejectMessage_Requeue() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "queue_test_consume_reject_requeue"
	suite.CreateTestQueue(queueName, "", "", true)

	// Publish 1 message
	suite.PublishMessages(suite.T(), "", queueName, 1)

	delivery := suite.GetMessage(queueName, false)

	err := delivery.Reject(true)
	if !suite.NoError(err, "reject and requeue message") {
		suite.T().FailNow()
	}

	// If the message was nacked then we should get the same message
	redelivery, ok, err := suite.ChannelConsume().Get(queueName, false)
	if !suite.NoError(err, "get empty queue") {
		suite.T().FailNow()
	}

	if !suite.True(ok, "got message") {
		suite.T().FailNow()
	}

	suite.Equal(delivery.Body, redelivery.Body, "message is redelivery")
}

func (suite *ChannelMethodsSuite) Test0320_Acknowledge_OrphanErr() {
	type testCase struct {
		method       string
		publishCount int
	}

	// We're going to use a table test to test all three acknowledgement methods
	testCases := []testCase{
		{
			method:       "ack",
			publishCount: 1,
		},
		{
			method:       "nack",
			publishCount: 1,
		},
		{
			method:       "reject",
			publishCount: 1,
		},
	}

	queueName := "queue_test_consume_ack_orphan"
	suite.CreateTestQueue(queueName, "", "", true)

	var thisCase testCase

	test := func(t *testing.T) {
		assert := assert.New(t)

		t.Cleanup(suite.ReplaceChannels)

		// Publish 1 message. This one message will keep getting redelivered on the force
		// reconnectMiddleware, so we only need to publish it once
		suite.PublishMessages(t, "", queueName, thisCase.publishCount)

		var delivery internal.Delivery
		for i := 0; i < thisCase.publishCount; i++ {
			delivery = suite.GetMessage(queueName, false)
		}

		suite.ChannelConsume().Test(t).ForceReconnect(nil)

		var err error

		switch thisCase.method {
		case "ack":
			err = delivery.Ack(false)
		case "nack":
			err = delivery.Nack(false, false)
		case "reject":
			err = delivery.Reject(false)
		default:
			t.Errorf("incorrect method arg: %v", thisCase.method)
			t.FailNow()
		}
		if !assert.Error(err, "got acknowledgement error") {
			t.FailNow()
		}

		var orphanErr amqp.ErrCantAcknowledgeOrphans
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

func (suite *ChannelMethodsSuite) Test0330_QoS_PrefetchCount() {
	suite.T().Cleanup(suite.ReplaceChannels)

	err := suite.ChannelConsume().Qos(10, 0, false)
	suite.NoError(err, "QoS")

	provider := suite.ChannelConsume().
		Test(suite.T()).
		GetMiddlewareProvider(defaultmiddlewares.QoSMiddlewareID)

	qosMiddleware, ok := provider.(*defaultmiddlewares.QoSMiddleware)
	if !suite.True(ok, "type assert QoSMiddleware") {
		suite.T().FailNow()
	}

	suite.Equal(
		10,
		qosMiddleware.QosArgs().PrefetchCount,
		"prefetch count",
	)
	suite.Equal(
		0, qosMiddleware.QosArgs().PrefetchSize, "prefetch size",
	)

	suite.True(qosMiddleware.IsSet(), "QoS was set")
}

func (suite *ChannelMethodsSuite) Test0340_QoS_OverReconnect() {
	suite.T().Cleanup(suite.ReplaceChannels)

	err := suite.ChannelConsume().Qos(10, 0, false)
	suite.NoError(err, "QoS")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Force a reconnectMiddleware
	suite.ChannelConsume().Test(suite.T()).ForceReconnect(ctx)

	// TODO: Add some way to make sure the correct QoS was sent to the server, probably
	//   by adding some mock functionality to the Tester object or monkey-patching.
}

func (suite *ChannelMethodsSuite) Test0350_QoS_PrefetchSize_Err() {
	suite.T().Cleanup(suite.ReplaceChannels)

	err := suite.ChannelConsume().Qos(0, 1024, false)
	if !suite.Error(err, "QoS err") {
		suite.T().FailNow()
	}

	suite.EqualError(
		err,
		"Exception (540) Reason: \"NOT_IMPLEMENTED - prefetch_size!=0 (1024)\"",
	)
}

// TODO: Implement mocked tests for flow settings over reconnectMiddleware. Flow is not supported
//   by RabbitMQ
func (suite *ChannelMethodsSuite) Test0360_Flow() {
	suite.T().Cleanup(suite.ReplaceChannels)

	err := suite.ChannelConsume().Flow(false)
	suite.Error(err, "error deactivating flow")
	suite.EqualError(
		err, "Exception (540) Reason: \"NOT_IMPLEMENTED - active=false\"",
	)
}

func (suite *ChannelMethodsSuite) Test0370_NotifyFlow() {
	suite.T().Cleanup(suite.ReplaceChannels)

	flowEvents := make(chan bool, 2)
	suite.ChannelConsume().NotifyFlow(flowEvents)

	// Check that we don'tb get flow notifications right off the bat
	select {
	case <-flowEvents:
		suite.T().Error("got flow event")
		suite.T().FailNow()
	default:
	}

	// Force a reconnectMiddleware
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	suite.ChannelConsume().Test(suite.T()).ForceReconnect(ctx)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	// Check that we have a flow = false followed by a flow = true notification on
	// reconnectMiddleware
	select {
	case flow, open := <-flowEvents:
		suite.False(flow, "flow false notification")
		suite.True(open, "event channel is open")
	case <-timer.C:
		suite.T().Error("no flow event")
		suite.T().FailNow()
	}

	select {
	case flow, open := <-flowEvents:
		suite.True(flow, "flow false notification")
		suite.True(open, "event channel is open")
	case <-timer.C:
		suite.T().Error("no flow event")
		suite.T().FailNow()
	}

	// close the channel
	err := suite.ChannelConsume().Close()
	suite.NoError(err, "close channel")

	// Check that no further notifications are sent, and the event channel is closed
	select {
	case _, open := <-flowEvents:
		suite.False(open, "event channel is closed")
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("flow event close timeout")
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0380_NotifyCancel() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_notify_cancel"
	suite.CreateTestQueue(queueName, "", "", true)

	cancelEvents := make(chan string, 10)
	suite.ChannelConsume().NotifyCancel(cancelEvents)

	consumerName := "test_consumer"
	messages, err := suite.ChannelConsume().Consume(
		queueName,
		consumerName,
		true,
		false,
		false,
		false,
		nil,
	)
	if !suite.NoError(err, "consume") {
		suite.T().FailNow()
	}

	// Publish 1 message then consume it to make sure the consumer is running before
	// we cancel it.
	suite.PublishMessages(suite.T(), "", queueName, 1)

	select {
	case <-messages:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("consume timeout")
		suite.T().FailNow()
	}

	_, err = suite.ChannelConsume().QueueDelete(
		queueName, false, false, false,
	)

	if !suite.NoError(err, "delete queue") {
		suite.T().FailNow()
	}

	select {
	case cancelled := <-cancelEvents:
		suite.Equal(consumerName, cancelled)
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("event timeout")
		suite.T().FailNow()
	}

	err = suite.ChannelConsume().Close()
	if !suite.NoError(err, "close channel") {
		suite.T().FailNow()
	}

	select {
	case _, open := <-cancelEvents:
		suite.False(open, "event channel closed")
	case <-time.NewTimer(1 * time.Second).C:
		suite.T().Error("event close timeout")
		suite.T().FailNow()
	}
}

func (suite *ChannelMethodsSuite) Test0390_TxMethodsPanic() {
	type testCase struct {
		name        string
		method      func() error
		expectedErr error
	}

	testCases := []testCase{
		{
			name:        "Tx",
			method:      suite.ChannelConsume().Tx,
			expectedErr: nil,
		},
		{
			name:        "TxCommit",
			method:      suite.ChannelConsume().TxCommit,
			expectedErr: nil,
		},
		{
			name:        "TxRollback",
			method:      suite.ChannelConsume().TxRollback,
			expectedErr: nil,
		},
	}

	var thisCase testCase

	testFunc := func(t *testing.T) {
		assert := assert.New(t)

		errString := fmt.Sprintf(
			"%v and other transaction methods not implemented, pull requests"+
				" are welcome for this functionality",
			thisCase.name,
		)

		assert.PanicsWithError(
			errString,
			func() {
				thisCase.method()
			},
		)
	}

	for _, thisCase = range testCases {
		suite.T().Run(thisCase.name, testFunc)
	}
}

func TestChannelMethods(t *testing.T) {
	suite.Run(t, &ChannelMethodsSuite{
		AmqpSuite: amqptest.NewAmqpSuite(new(suite.Suite), nil),
	})
}
