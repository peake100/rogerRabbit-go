//revive:disable:import-shadowing

package amqptest

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	// TestDialAddress is the default address for a test broker.
	TestDialAddress = "amqp://localhost:57018"
)

// GetTestConnection creates a new connection to amqp://localhost:57018, where our
// test broker will be listening.
//
// t.FailNot() is called on any errors.
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
