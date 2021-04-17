package amqp_test

import (
	"github.com/peake100/rogerRabbit-go/pkg/amqp"
	"github.com/peake100/rogerRabbit-go/pkg/amqptest"
	streadway "github.com/streadway/amqp"
	"sync"
	"testing"
	"time"
)

var msgBody = []byte("some message")

func BenchmarkComparison_QueueInspect_Streadway(b *testing.B) {
	channel := dialStreadway(b)
	queue := setupQueue(b, channel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := channel.QueueInspect(queue.Name)
		if err != nil {
			b.Fatalf("error getting queue info: %v", err)
		}
	}
}

func BenchmarkComparison_QueueInspect_Roger(b *testing.B) {
	channel := dialRoger(b)
	queue := setupQueue(b, channel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := channel.QueueInspect(queue.Name)
		if err != nil {
			b.Fatalf("error getting queue info: %v", err)
		}
	}
}

func BenchmarkComparison_QueuePublish_Streadway(b *testing.B) {
	channel := dialStreadway(b)
	queue := setupQueue(b, channel)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := channel.Publish(
			"",
			queue.Name,
			true,
			false,
			amqp.Publishing{
				Body: msgBody,
			},
		)
		if err != nil {
			b.Fatalf("error publishing message: %v", err)
		}
	}
}

func BenchmarkComparison_QueuePublish_Roger(b *testing.B) {
	channel := dialRoger(b)
	queue := setupQueue(b, channel)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := channel.Publish(
			"",
			queue.Name,
			true,
			false,
			amqp.Publishing{
				Body: msgBody,
			},
		)
		if err != nil {
			b.Fatalf("error publishing message: %v", err)
		}
	}
}

func BenchmarkComparison_QueuePublishConfirm_Streadway(b *testing.B) {
	channel := dialStreadway(b)
	queue := setupQueue(b, channel)

	err := channel.Confirm(false)
	if err != nil {
		b.Fatalf("error putting channel into confrimation mode")
	}

	confirmations := make(chan amqp.BasicConfirmation, 100)
	channel.NotifyPublish(confirmations)

	done := new(sync.WaitGroup)
	done.Add(2)

	errPublish := make(chan error, 1)

	b.ResetTimer()
	go func() {
		defer done.Done()

		for i := 0; i < b.N; i++ {
			err := channel.Publish(
				"",
				queue.Name,
				true,
				false,
				amqp.Publishing{
					Body: msgBody,
				},
			)
			if err != nil {
				b.Errorf("error publishing message: %v", err)
				errPublish <- err
				return
			}
		}
	}()

	go func() {
		defer done.Done()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		for i := 0; i < b.N; i++ {
			timer.Reset(5 * time.Second)
			select {
			case confirm := <-confirmations:
				if !confirm.Ack {
					b.Errorf(
						"publication nacked for tag %v", confirm.DeliveryTag,
					)
					return
				}
			case <-errPublish:
				b.Errorf("error publishing. aborting confirmations")
				return
			case <-timer.C:
				b.Errorf("timeout on confirmation %v", b.N)
				return
			}
		}
	}()

	done.Wait()
}

func BenchmarkComparison_QueuePublishConfirm_Roger(b *testing.B) {
	channel := dialRoger(b)
	queue := setupQueue(b, channel)

	err := channel.Confirm(false)
	if err != nil {
		b.Fatalf("error putting channel into confrimation mode")
	}

	confirmations := make(chan amqp.Confirmation, 100)
	channel.NotifyPublish(confirmations)

	done := new(sync.WaitGroup)
	done.Add(2)

	errPublish := make(chan error, 1)

	b.ResetTimer()
	go func() {
		defer done.Done()

		for i := 0; i < b.N; i++ {
			err := channel.Publish(
				"",
				queue.Name,
				true,
				false,
				amqp.Publishing{
					Body: msgBody,
				},
			)
			if err != nil {
				b.Errorf("error publishing message: %v", err)
				errPublish <- err
				return
			}
		}
	}()

	go func() {
		defer done.Done()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		for i := 0; i < b.N; i++ {
			timer.Reset(5 * time.Second)
			select {
			case confirm := <-confirmations:
				if !confirm.Ack {
					b.Errorf(
						"publication nacked for tag %v", confirm.DeliveryTag,
					)
					return
				}
			case <-errPublish:
				b.Errorf("error publishing. aborting confirmations")
				return
			case <-timer.C:
				b.Errorf("timeout on confirmation %v", b.N)
				return
			}
		}
	}()

	done.Wait()
}

// dialStreadway gets a streadway Connection
func dialStreadway(b *testing.B) *amqp.BasicChannel {
	conn, err := streadway.Dial(amqptest.TestDialAddress)
	if err != nil {
		b.Fatalf("error dialing connection")
	}
	b.Cleanup(func() {
		conn.Close()
	})

	channel, err := conn.Channel()
	if err != nil {
		b.Fatalf("error getting channel: %v", err)
	}
	return channel
}

func dialRoger(b *testing.B) *amqp.Channel {
	conn, err := amqp.Dial(amqptest.TestDialAddress)
	if err != nil {
		b.Fatalf("error dialing connection")
	}
	b.Cleanup(func() {
		conn.Close()
	})

	channel, err := conn.Channel()
	if err != nil {
		b.Fatalf("error getting channel: %v", err)
	}
	return channel
}

func setupQueue(b *testing.B, channel OrganizesQueues) amqp.Queue {
	queue, err := channel.QueueDeclare(
		"benchmark_queue_inspect",
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		b.Fatalf("error getting queue: %v", err)
	}

	_, err = channel.QueuePurge(queue.Name, false)
	if err != nil {
		b.Fatalf("error purging queue: %v", err)
	}

	// Delete the queue on the way out.
	b.Cleanup(func() {
		channel.QueueDelete(queue.Name, false, false, false)
	})

	return queue
}

// publishesAndConfirms is used to run the publish anc confirm test.
type OrganizesQueues interface {
	QueueDeclare(
		name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table,
	) (queue amqp.Queue, err error)
	QueuePurge(name string, noWait bool) (count int, err error)
	QueueDelete(
		name string, ifUnused, ifEmpty, noWait bool,
	) (count int, err error)
}
