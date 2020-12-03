package roger

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type BasicTestConsumer struct {
	QueueName            string
	ExpectedMessageCount int

	// We'll keep track of how many messages we have received here.
	ReceivedMessageCount int
	ReceivedLock         *sync.Mutex

	// This channel will be closed when ReceivedMessageCount equals
	// ExpectedMessageCount.
	AllReceived chan struct{}

	// Will be closed when the setup of the processor is complete.
	SetupComplete chan struct{}
}

func (consumer *BasicTestConsumer) ConsumeArgs() *ConsumeArgs {
	return &ConsumeArgs{
		ConsumerName: consumer.QueueName,
		AutoAck:      false,
		Exclusive:    false,
		Args:         nil,
	}
}

func (consumer *BasicTestConsumer) SetupChannel(
	ctx context.Context, amqpChannel AmqpRouteManager,
) error {
	defer close(consumer.SetupComplete)

	_, err := amqpChannel.QueueDeclare(
		consumer.QueueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error declaring Queue: %w", err)
	}

	return nil
}

func (consumer *BasicTestConsumer) HandleDelivery(
	ctx context.Context, delivery amqp.Delivery, logger zerolog.Logger,
) (err error, requeue bool) {

	// Check whether all messages have been received, then signal receipt.
	consumer.ReceivedLock.Lock()
	defer consumer.ReceivedLock.Unlock()
	consumer.ReceivedMessageCount++
	if consumer.ReceivedMessageCount == consumer.ExpectedMessageCount {
		close(consumer.AllReceived)
	}

	logger.Info().
		Bytes("BODY", delivery.Body).
		Int("RECEIVED_COUNT", consumer.ReceivedMessageCount).
		Msg("message received")

	return nil, false
}

func (consumer *BasicTestConsumer) Cleanup(amqpChannel AmqpRouteManager) error {
	_, err := amqpChannel.QueueDelete(
		consumer.QueueName, false, false, false,
	)
	if err != nil {
		return fmt.Errorf("error deleting Queue: %w", err)
	}

	return nil
}

type ConsumerSuite struct {
	amqp.ChannelSuiteBase
}

func (suite *ConsumerSuite) TestConsumeBasicLifecycle() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_consume_basic_lifecycle"

	_, err := suite.ChannelPublish.QueueInspect(queueName)
	suite.EqualError(
		err,
		"Exception (404) Reason: \"NOT_FOUND - no queue"+
			" 'test_consume_basic_lifecycle' in vhost '/'\"",
		"queue not created",
	)

	processor := &BasicTestConsumer{
		QueueName:            queueName,
		ExpectedMessageCount: 0,
		ReceivedMessageCount: 0,
		ReceivedLock:         new(sync.Mutex),
		AllReceived:          make(chan struct{}),
		SetupComplete:        make(chan struct{}),
	}

	consumer := NewConsumer(suite.ChannelConsume, nil)
	suite.T().Cleanup(consumer.StartShutdown)

	consumer.RegisterProcessor(processor)

	runComplete := make(chan struct{})

	go func() {
		defer close(runComplete)
		defer consumer.StartShutdown()
		err := consumer.Run()
		suite.NoError(err, "run consumer")
	}()

	select {
	case <-processor.SetupComplete:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("channel setup timeout")
		suite.T().FailNow()
	}

	_, err = suite.ChannelPublish.QueueInspect(queueName)
	suite.NoError(err, "queue created")

	consumer.StartShutdown()

	select {
	case <-runComplete:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}

	// Check that shutting down the consumer triggered the processor cleanup and
	// deleting the queue.
	_, err = suite.ChannelPublish.QueueInspect(queueName)
	suite.EqualError(
		err,
		"Exception (404) Reason: \"NOT_FOUND - no queue"+
			" 'test_consume_basic_lifecycle' in vhost '/'\"",
		"queue deleted",
	)
}

func (suite *ConsumerSuite) TestConsumeBasicMessages() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_consume_basic"

	suite.CreateTestQueue(queueName, "", "")

	processor := &BasicTestConsumer{
		QueueName:            queueName,
		ExpectedMessageCount: 10,
		ReceivedMessageCount: 0,
		ReceivedLock:         new(sync.Mutex),
		AllReceived:          make(chan struct{}),
		SetupComplete:        make(chan struct{}),
	}

	consumer := NewConsumer(suite.ChannelConsume, nil)
	suite.T().Cleanup(consumer.StartShutdown)

	consumer.RegisterProcessor(processor)

	runComplete := make(chan struct{})

	go func() {
		defer close(runComplete)
		defer consumer.StartShutdown()
		err := consumer.Run()
		suite.NoError(err, "run consumer")
	}()

	// Publish 10 messages
	suite.PublishMessages(suite.T(), "", queueName, 10)

	select {
	case <-processor.AllReceived:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("messages processed timeout")
		suite.T().FailNow()
	}

	consumer.StartShutdown()

	select {
	case <-runComplete:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}
}

func TestNewConsumer(t *testing.T) {
	suite.Run(t, new(ConsumerSuite))
}
