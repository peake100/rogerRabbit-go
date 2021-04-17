//revive:disable:import-shadowing

package amqptest

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp"
	"testing"
)

const (
	// TestDialAddress is the default address for a test broker.
	TestDialAddress = "amqp://localhost:57018"
)

// GetTestConnection creates a new connection to amqp://localhost:57018, where our
// test broker will be listening.
//
// t.FailNow() is called on any errors.
func GetTestConnection(tb testing.TB) *amqp.Connection {

	conn, err := amqp.DialCtx(context.Background(), TestDialAddress)
	if err != nil {
		tb.Errorf("error dialing broker: %v", err)
		tb.FailNow()
	}

	if conn == nil {
		tb.Errorf("connection is nil: %v", err)
		tb.FailNow()
	}

	tb.Cleanup(
		func() {
			_ = conn.Close()
		},
	)

	return conn
}
