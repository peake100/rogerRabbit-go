//revive:disable
package roger_test

import (
	"context"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqptest"
	"github.com/peake100/rogerRabbit-go/roger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type ProducerSuite struct {
	amqptest.AmqpSuite
}

func (suite *ProducerSuite) TestProducerBasicLifetime() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_queue_producer_lifetime"
	suite.CreateTestQueue(queueName, "", "", true)

	producer := roger.NewProducer(suite.ChannelPublish(), nil)
	complete := make(chan struct{})

	go func() {
		defer close(complete)

		err := producer.Run()
		suite.NoError(err, "run producer")
	}()

	published := make(chan struct{})
	go func() {
		defer close(published)

		err := producer.Publish(
			context.Background(),
			"",
			"",
			false,
			false,
			amqp.Publishing{
				Body: []byte("Some Message"),
			},
		)
		suite.NoError(err, "publish message")
	}()

	select {
	case <-published:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("publish timeout")
		suite.T().FailNow()
	}

	producer.StartShutdown()

	select {
	case <-complete:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("shutdown timeout")
		suite.T().FailNow()
	}
}

func (suite *ProducerSuite) TestProducerPublish() {
	suite.T().Cleanup(suite.ReplaceChannels)

	publishCount := 10

	queueName := "test_queue_producer_publish"
	suite.CreateTestQueue(queueName, "", "", true)

	producer := roger.NewProducer(suite.ChannelPublish(), nil)

	go func() {
		err := producer.Run()
		suite.NoError(err, "run producer")
	}()

	suite.T().Cleanup(producer.StartShutdown)

	publishWork := new(sync.WaitGroup)
	publishWork.Add(publishCount)

	publishComplete := make(chan struct{})

	go func() {
		defer close(publishComplete)
		publishWork.Wait()
	}()

	for i := 0; i < publishCount; i++ {
		go func(index int) {
			defer publishWork.Done()
			err := producer.Publish(
				context.Background(),
				"",
				queueName,
				false,
				false,
				amqp.Publishing{
					Body: []byte(fmt.Sprint(index)),
				},
			)
			suite.NoError(err, "publish message", index)
		}(i)
	}

	select {
	case <-publishComplete:
	case <-time.NewTimer(3 * time.Second).C:
		suite.T().Error("publish timeout")
		suite.T().FailNow()
	}

	messages := [10]bool{}
	for i := 0; i < publishCount; i++ {
		msg, ok, err := suite.ChannelPublish().Get(queueName, true)
		suite.NoError(err, "get message")
		suite.True(ok, "message existed")

		index, err := strconv.Atoi(string(msg.Body))
		suite.NoError(err, "parse message", i)

		messages[index] = true
	}

	for i, val := range messages {
		suite.Truef(val, "message %i found", i)
	}
}

func (suite *ProducerSuite) TestProducerQueuePublication() {
	suite.T().Cleanup(suite.ReplaceChannels)

	publishCount := 10

	queueName := "test_queue_producer_queue_publication"
	suite.CreateTestQueue(queueName, "", "", true)

	producer := roger.NewProducer(suite.ChannelPublish(), nil)

	go func() {
		err := producer.Run()
		suite.NoError(err, "run producer")
	}()

	suite.T().Cleanup(producer.StartShutdown)

	publications := make([]*roger.Publication, publishCount)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < publishCount; i++ {
		publication, err := producer.QueueForPublication(
			ctx,
			"",
			queueName,
			false,
			false,
			amqp.Publishing{
				Body: []byte(fmt.Sprint(i)),
			},
		)
		if !suite.NoErrorf(err, "queue order %v", i) {
			suite.T().FailNow()
		}
		publications[i] = publication
	}

	publishWork := new(sync.WaitGroup)
	publishWork.Add(publishCount)
	publishComplete := make(chan struct{})
	go func() {
		defer close(publishComplete)
		publishWork.Wait()
	}()

	for i, thisPublication := range publications {
		err := thisPublication.WaitOnConfirmation()
		if !suite.NoError(err, "wait on publication %v", i) {
			suite.T().FailNow()
		}
	}

	for i := 0; i < publishCount; i++ {
		msg := suite.GetMessage(queueName, true)
		suite.Equal(fmt.Sprint(i), string(msg.Body), "message expected")
	}
}

func TestProducer(t *testing.T) {
	suite.Run(t, &ProducerSuite{
		AmqpSuite: amqptest.NewAmqpSuite(new(suite.Suite), nil),
	})
}
