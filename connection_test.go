//revive:disable

package rogerRabbit_test

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp"
	"github.com/peake100/rogerRabbit-go/amqpTest"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// Test that dial creates a robust connection with a working inner connection, and that
// the robust connection can be closed.
func Test0000_Dial_Succeed(t *testing.T) {
	assert := assert.New(t)

	var conn *amqp.Connection
	var err error
	connected := make(chan struct{})

	go func() {
		defer close(connected)
		conn, err = amqp.Dial(amqpTest.TestAddress)
	}()

	timeout := time.NewTimer(15 * time.Second)

	select {
	case <-connected:
	case <-timeout.C:
		t.Error("dial timeout")
		t.FailNow()
	}

	defer conn.Close()
	if !assert.NoError(err, "dial broker") {
		t.FailNow()
	}
}

func Test0005_Dial_Fail(t *testing.T) {
	assert := assert.New(t)

	conn, err := amqp.Dial("bad address")

	if !assert.Error(err, "error occurred") {
		t.FailNow()
	}
	assert.Nil(conn, "connection is nil")

	assert.EqualError(
		err, "URI must not contain whitespace", "error text",
	)
}

func Test0010_DialCtx_Succeed(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := amqp.DialCtx(ctx, amqpTest.TestAddress)
	if err != nil {
		return
	}

	if !assert.NoError(err, "dial connection") {
		t.FailNow()
	}

	defer conn.Close()
}

func Test0020_DialCtx_Fail_Cancelled(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	// cancel the error.
	cancel()

	conn, err := amqp.DialCtx(ctx, "bad address")
	if !assert.Error(err, "dialing error") {
		return
	}

	if !assert.ErrorIs(err, context.Canceled) {
		t.FailNow()
	}

	assert.Nil(conn)
}

func Test0030_Close(t *testing.T) {
	assert := assert.New(t)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()

	conn, err := amqp.DialCtx(dialCtx, amqpTest.TestAddress)
	if !assert.NoError(err, "dial broker") {
		t.FailNow()
	}

	connTester := conn.Test(t)

	assert.False(
		connTester.UnderlyingConn().IsClosed(),
		"current connection is open",
	)

	err = conn.Close()
	if !assert.NoError(err, "close connection") {
		t.FailNow()
	}

	assert.True(
		conn.IsClosed(), context.Canceled, "robust connection is closed",
	)
	assert.True(
		connTester.UnderlyingConn().IsClosed(),
		"underlying connection is closed",
	)
}

func Test0040_Connection_Reconnect(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	connTester := conn.Test(t)

	// Reconnect 10 times
	for i := 0; i < 10; i++ {
		// Check that the internal connection is initially open.
		if !assert.False(
			connTester.UnderlyingConn().IsClosed(),
			"initial connection is open %v",
			i,
		) {
			t.FailNow()
		}

		// Make sre we are reporting the user-facing connection is open
		assert.False(conn.IsClosed(), "connection is not closed")

		currentConn := connTester.UnderlyingConn()

		// Force a reconnection
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			conn.Test(t).ForceReconnect(ctx)
		}()

		assert.NotSame(
			currentConn,
			connTester.UnderlyingConn(),
			"current connection was replaces",
		)

		// Check that the internal connection is now open
		if !assert.False(
			connTester.UnderlyingConn().IsClosed(),
			"connection is open after tryReconnect %v",
			i,
		) {
			t.FailNow()
		}
		assert.False(conn.IsClosed(), "connection is not closed")
	}
}

func TestConnection_NotifyOnClose_NoReconnect_NoErr(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)

	closeEvent := make(chan *streadway.Error, 1)
	conn.NotifyClose(closeEvent)

	err := conn.Close()
	assert.NoError(err, "close connection")

	select {
	case <-closeEvent:
	case <-time.NewTimer(3 * time.Second).C:
		t.Error("close event timed out")
	}

	err = <-closeEvent
	assert.Nil(err, "close event error free")
}

// Tests that NotifyOnClose does not fire on an internal tryReconnect.
func TestConnection_NotifyOnClose_Reconnect_NoErr(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	connTester := conn.Test(t)

	closeEvent := make(chan *streadway.Error, 1)
	conn.NotifyClose(closeEvent)

	// Force close the internal connection and wait for a reconnection to occur.
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()
	connTester.ForceReconnect(ctx)

	// Make sure the internal disconnect did not fire a close event
	select {
	case <-closeEvent:
		t.Error("close event fired before actual close")
	default:
	}

	err := conn.Close()
	assert.NoError(err, "close connection")

	select {
	case <-closeEvent:
	case <-time.NewTimer(3 * time.Second).C:
		t.Error("close event timed out")
	}

	err = <-closeEvent
	assert.Nil(err, "close event error free")
}

func TestConnection_NotifyOnClose_AlreadyClosed(t *testing.T) {
	conn := amqpTest.GetTestConnection(t)
	err := conn.Close()
	if !assert.NoError(t, err, "close connection") {
		t.FailNow()
	}

	closeEvent := make(chan *streadway.Error, 1)
	conn.NotifyClose(closeEvent)

	select {
	case <-closeEvent:
	default:
		t.Error("notification channelConsume not immediately closed")
	}
}

func TestConnection_NotifyOnDial(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	connTester := conn.Test(t)

	dialEvent := make(chan error, 1)
	conn.NotifyDial(dialEvent)

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()
	connTester.ForceReconnect(ctx)

	timeout := time.NewTimer(1 * time.Second)

	select {
	case err, closed := <-dialEvent:
		assert.True(closed, "dial event channelConsume is open")
		assert.NoError(err, "dial event error is nil")
	case <-timeout.C:
		t.Error("dial channelConsume did not receive event")
	}

	conn.Close()

	select {
	case _, closed := <-dialEvent:
		assert.False(closed, "dial event channelConsume is closed")
	case <-timeout.C:
		t.Error("dial channelConsume did not receive event")
	}
}

func TestConnection_NotifyOnDial_AlreadyClosed(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	conn.Close()

	eventChan := make(chan error, 1)
	_ = conn.NotifyDial(eventChan)

	timeout := time.NewTimer(1 * time.Second)
	select {
	case _, closed := <-eventChan:
		assert.False(closed, "dial event channelConsume is closed")
	case <-timeout.C:
		t.Error("dial channelConsume did not close")
	}
}

func TestConnection_NotifyOnDisconnect(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)

	disconnectEvent := make(chan error, 2)
	err := conn.NotifyDisconnect(disconnectEvent)
	assert.NoError(err, "notification subscribe")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn.Test(t).ForceReconnect(ctx)

	timeout := time.NewTimer(1 * time.Second)

	select {
	case err, closed := <-disconnectEvent:
		assert.True(closed, "disconnect event channelConsume is open")
		assert.NoError(err, "first disconnect event error is nil")
	case <-timeout.C:
		t.Error("disconnect channelConsume did not receive event")
		t.FailNow()
	}

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn.Test(t).ForceReconnect(ctx)

	timeout.Reset(1 * time.Second)

	select {
	case err, open := <-disconnectEvent:
		assert.True(open, "disconnect event channelConsume is open")
		assert.NoError(err, "second disconnect event error is nil")
	case <-timeout.C:
		t.Error("disconnect channelConsume did not receive event")
	}

	err = conn.Close()
	if !assert.NoError(err, "close connection") {
		t.FailNow()
	}

	timeout.Reset(1 * time.Second)

	// See if we can drain this channel
	eventsClosed := make(chan struct{})
	go func() {
		defer close(eventsClosed)
		for range disconnectEvent {
		}
	}()

	select {
	case <-eventsClosed:
	case <-timeout.C:
		t.Error("disconnect channelConsume did not close")
		t.FailNow()
	}
}

func TestConnection_NotifyOnDisconnect_AlreadyClosed(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	conn.Close()

	eventChan := make(chan error, 1)
	err := conn.NotifyDisconnect(eventChan)
	assert.ErrorIs(err, streadway.ErrClosed, "error on register")

	timeout := time.NewTimer(1 * time.Second)
	select {
	case _, closed := <-eventChan:
		assert.False(closed, "disconnect event channelConsume is closed")
	case <-timeout.C:
		t.Error("disconnect channelConsume did not close")
	}
}

func TestConnection_IsClosed(t *testing.T) {
	assert := assert.New(t)

	conn := amqpTest.GetTestConnection(t)
	connTester := conn.Test(t)

	// Check that we report the connection as open on creation
	assert.False(conn.IsClosed(), "connection is open")

	t.Log("register notification")
	// register a notification for when we reconnectMiddleware
	dialEvents := make(chan error, 10)
	err := conn.NotifyDial(dialEvents)
	if !assert.NoError(err, "register dial notifier") {
		t.FailNow()
	}

	func(){
		// grab a lock on the transport so we don't auto-reconnectMiddleware
		connTester.BlockReconnect()

		// release the lock to let the connection reconnectMiddleware
		defer connTester.UnblockReconnect()

		// force close the internal connection
		connTester.DisconnectTransport()

		// make sure we are still open on the user-facing side, even though the internal
		// connection is down
		assert.False(conn.IsClosed(), "connection is open")
	}()


	timeout := time.NewTimer(3 * time.Second)

	select {
	case dialErr := <-dialEvents:
		if !assert.NoError(dialErr, "dial error") {
			t.FailNow()
		}
	case <-timeout.C:
		t.Error("redial timeout")
		t.FailNow()
	}

	// make sure we are still open on the user-facing side after a reconnectMiddleware
	assert.False(conn.IsClosed(), "connection is open")

	// Fully close the connection
	err = conn.Close()
	if !assert.NoError(err, "close roger connection") {
		t.FailNow()
	}

	// Make sure we are now telling the user the connection is closed.
	// make sure we are still open on the user-facing side after a reconnectMiddleware
	assert.True(conn.IsClosed(), "connection is closed")
}
