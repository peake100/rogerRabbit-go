//revive:disable
package roger_test

import (
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp"
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
	amqp.ChannelSuiteBase
}

func (suite *ProducerSuite) TestProducerBasicLifetime() {
	suite.T().Cleanup(suite.ReplaceChannels)

	queueName := "test_queue_producer_lifetime"
	suite.CreateTestQueue(queueName, "", "")

	producer := roger.NewProducer(suite.ChannelPublish, nil)
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
	suite.CreateTestQueue(queueName, "", "")

	producer := roger.NewProducer(suite.ChannelPublish, nil)

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
		msg, ok, err := suite.ChannelPublish.Get(queueName, true)
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

func TestProducer(t *testing.T) {
	suite.Run(t, new(ProducerSuite))
}
