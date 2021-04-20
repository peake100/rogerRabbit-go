//revive:disable

package rconsumer

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/amqptest"
	"github.com/peake100/rogerRabbit-go/pkg/roger/rconsumer/middleware"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type BasicTestProcessor struct {
	QueueName string

	// We'll keep track of how many messages we have received here.
	ReceivedMessageCount int
	ReceiveCond          *sync.Cond

	// Panic will cause the HandleDelivery to panic when called.
	Panic bool

	// Will be closed when the setup of the processor is complete.
	SetupComplete chan struct{}
}

func (consumer *BasicTestProcessor) AmqpArgs() AmqpArgs {
	return AmqpArgs{
		ConsumerName: consumer.QueueName,
		AutoAck:      false,
		Exclusive:    false,
		Args:         nil,
	}
}

func (consumer *BasicTestProcessor) SetupChannel(
	ctx context.Context, amqpChannel middleware.AmqpRouteManager,
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

func (consumer *BasicTestProcessor) HandleDelivery(
	ctx context.Context, delivery amqp.Delivery,
) (requeue bool, err error) {

	// Check whether all messages have been received, then signal receipt.
	consumer.ReceiveCond.L.Lock()
	defer consumer.ReceiveCond.L.Unlock()
	defer func() {
		consumer.ReceivedMessageCount++
		consumer.ReceiveCond.Broadcast()
	}()

	if consumer.Panic {
		panic("mock panic")
	}

	return false, nil
}

func (consumer *BasicTestProcessor) CleanupChannel(
	_ context.Context, amqpChannel middleware.AmqpRouteManager,
) error {
	_, err := amqpChannel.QueueDelete(
		consumer.QueueName, false, false, false,
	)
	if err != nil {
		return fmt.Errorf("error deleting Queue: %w", err)
	}

	return nil
}

type ConsumerSuite struct {
	amqptest.AmqpSuite
}

func (suite *ConsumerSuite) TestConsumeBasicLifecycle() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_consume_basic_lifecycle"

	processor := &BasicTestProcessor{
		QueueName:            queueName,
		ReceivedMessageCount: 0,
		ReceiveCond:          sync.NewCond(new(sync.Mutex)),
		SetupComplete:        make(chan struct{}),
	}

	consumer := New(suite.ChannelConsume(), DefaultOpts().WithLoggingLevel(zerolog.DebugLevel))
	suite.T().Cleanup(consumer.StartShutdown)

	err := consumer.RegisterProcessor(processor)
	if !suite.NoError(err, "register processor") {
		suite.T().FailNow()
	}

	runComplete := make(chan struct{})

	go func() {
		defer close(runComplete)
		defer consumer.StartShutdown()
		err := consumer.Run()
		suite.NoError(err, "run consumer")
	}()

	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()

	select {
	case <-processor.SetupComplete:
	case <-timeout.C:
		suite.T().Error("channel setup timeout")
		suite.T().FailNow()
	}

	_, err = suite.ChannelPublish().QueueInspect(queueName)
	suite.NoError(err, "queue created")

	consumer.StartShutdown()

	select {
	case <-runComplete:
	case <-timeout.C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}

	// Check that shutting down the consumer triggered the processor cleanup and
	// deleting the queue.
	_, err = suite.ChannelPublish().QueueInspect(queueName)
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

	suite.CreateTestQueue(queueName, "", "", true)

	messageCount := 500

	processor := &BasicTestProcessor{
		QueueName:            queueName,
		ReceivedMessageCount: 0,
		ReceiveCond:          sync.NewCond(new(sync.Mutex)),
		SetupComplete:        make(chan struct{}),
	}

	consumer := New(suite.ChannelConsume(), DefaultOpts().WithLoggingLevel(zerolog.DebugLevel))
	suite.T().Cleanup(consumer.StartShutdown)

	err := consumer.RegisterProcessor(processor)
	if !suite.NoError(err, "register processor") {
		suite.T().FailNow()
	}

	runComplete := make(chan struct{})

	go func() {
		defer close(runComplete)
		defer consumer.StartShutdown()
		err := consumer.Run()
		suite.NoError(err, "run consumer")
	}()

	allReceived := make(chan struct{})
	go func() {
		defer close(allReceived)
		processor.ReceiveCond.L.Lock()
		defer processor.ReceiveCond.L.Unlock()
		for processor.ReceivedMessageCount < messageCount {
			processor.ReceiveCond.Wait()
		}
	}()

	// Publish 10 messages
	suite.PublishMessages(suite.T(), "", queueName, messageCount)

	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()

	select {
	case <-allReceived:
	case <-timeout.C:
		suite.T().Error("messages processed timeout")
		suite.T().FailNow()
	}

	consumer.StartShutdown()

	select {
	case <-runComplete:
	case <-timeout.C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}
}

// TestConsumePanic forces the DeliveryProcessor to panic.
func (suite *ConsumerSuite) TestConsumePanic() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_consume_panic"

	suite.CreateTestQueue(queueName, "", "", true)

	processor := &BasicTestProcessor{
		QueueName:            queueName,
		ReceivedMessageCount: 0,
		ReceiveCond:          sync.NewCond(new(sync.Mutex)),
		SetupComplete:        make(chan struct{}),
		Panic:                true,
	}

	consumer := New(suite.ChannelConsume(), DefaultOpts().WithLoggingLevel(zerolog.DebugLevel))
	suite.T().Cleanup(consumer.StartShutdown)

	err := consumer.RegisterProcessor(processor)
	if !suite.NoError(err, "register processor") {
		suite.T().FailNow()
	}

	runComplete := make(chan struct{})

	go func() {
		defer close(runComplete)
		defer consumer.StartShutdown()
		err := consumer.Run()
		suite.NoError(err, "run consumer")
	}()

	allReceived := make(chan struct{})
	go func() {
		defer close(allReceived)
		processor.ReceiveCond.L.Lock()
		defer processor.ReceiveCond.L.Unlock()
		for processor.ReceivedMessageCount == 0 {
			processor.ReceiveCond.Wait()
		}
	}()

	suite.PublishMessages(suite.T(), "", queueName, 1)

	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()

	select {
	case <-allReceived:
	case <-timeout.C:
		suite.T().Error("messages processed timeout")
		suite.T().FailNow()
	}

	consumer.StartShutdown()

	select {
	case <-runComplete:
	case <-timeout.C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}
}

func TestNewConsumer(t *testing.T) {
	suite.Run(t, &ConsumerSuite{
		AmqpSuite: amqptest.NewAmqpSuite(new(suite.Suite), nil),
	})
}
