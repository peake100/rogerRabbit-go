package amqp

import (
	"context"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// Get a new connection for testing.
func getTestConnection(t *testing.T) *Connection {
	assert := assert.New(t)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer dialCancel()

	conn, err := DialCtx(dialCtx, TestAddress)
	if !assert.NoError(err, "dial connection") {
		t.FailNow()
	}

	if !assert.NotNil(conn, "connection is not nil") {
		t.FailNow()
	}

	t.Cleanup(
		func() {
			conn.Close()
		},
	)

	return conn
}

func waitForReconnect(t *testing.T, manager *transportManager, reconnectCount uint64) {
	timeout := time.NewTimer(3 * time.Second)

	// Wait for the tryReconnect count to increment.
	for {
		if manager.reconnectCount >= reconnectCount {
			break
		}

		select {
		case _, closed := <- timeout.C:
			if closed {
				t.Error("tryReconnect took too long")
				t.FailNow()
			}
		default:
			time.Sleep(10)
		}
	}
}

// Test that dial creates a robust connection with a working inner connection, and that
// the robust connection can be closed.
func Test0000_Dial_Succeed(t *testing.T) {
	assert := assert.New(t)

	var conn *Connection
	var err error
	connected := make(chan struct{})

	go func() {
		defer close(connected)
		conn, err = Dial(TestAddress)
	}()

	timeout := time.NewTimer(15 * time.Second)

	select {
	case <- connected:
	case <- timeout.C:
		t.Error("dial timeout")
		t.FailNow()
	}

	if !assert.NoError(err, "dial broker") {
		t.FailNow()
	}

	defer conn.Close()
}

func Test0005_Dial_Fail(t *testing.T) {
	assert := assert.New(t)

	conn, err := Dial("bad address")

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

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	conn, err := DialCtx(ctx, TestAddress)
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

	conn, err := DialCtx(ctx, "bad address")
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

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer dialCancel()

	conn, err := DialCtx(dialCtx, TestAddress)
	if !assert.NoError(err, "dial broker") {
		t.FailNow()
	}

	assert.False(conn.transportConn.IsClosed(), "current connection is open")

	err = conn.Close()
	if !assert.NoError(err, "close connection") {
		t.FailNow()
	}

	assert.ErrorIs(
		conn.ctx.Err(), context.Canceled, "context is cancelled",
	)
	assert.True(
		conn.transportConn.IsClosed(), "current connection is closed",
	)
}

func Test0040_Connection_Reconnect(t *testing.T) {
	assert :=  assert.New(t)

	conn := getTestConnection(t)

	// Reconnect 10 times
	for i := 0 ; i < 10 ; i++ {
		// Check that the internal connection is initially open.
	 	if !assert.False(
			conn.transportConn.IsClosed(),
			"initial connection is open %v",
			i,
		) {
		 	t.FailNow()
	    }

	    currentConn := conn.transportConn.Connection

		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
			defer cancel()
			conn.Test(t).ForceReconnect(ctx)
		}()

		assert.NotSame(
			currentConn,
			conn.transportConn.Connection,
			"current connection was replaces",
		)

		// Check that the internal connection is now open
		if !assert.False(
			conn.transportConn.IsClosed(),
			"connection is open after tryReconnect %v",
			i,
		) {
			t.FailNow()
		}
	}
}


func TestConnection_NotifyOnClose_NoReconnect_NoErr(t *testing.T) {
	assert :=  assert.New(t)

	conn := getTestConnection(t)

	closeEvent := make(chan *streadway.Error, 1)
	conn.NotifyClose(closeEvent)

	err := conn.Close()
	assert.NoError(err, "close connection")

	select {
	case <- closeEvent:
	case <- time.NewTimer(3 * time.Second).C:
		t.Error("close event timed out")
	}

	err = <- closeEvent
	assert.Nil(err, "close event error free")
}

// Tests that NotifyOnClose does not fire on an internal tryReconnect.
func TestConnection_NotifyOnClose_Reconnect_NoErr(t *testing.T) {
	assert :=  assert.New(t)

	conn := getTestConnection(t)

	closeEvent := make(chan *streadway.Error, 1)
	conn.NotifyClose(closeEvent)

	// Force close the internal connection and wait for a reconnection to occur
	conn.transportConn.Close()
	waitForReconnect(t, conn.transportManager, 2)

	select {
	case <- closeEvent:
		t.Error("close event fired before actual close")
	default:
	}

	err := conn.Close()
	assert.NoError(err, "close connection")

	select {
	case <- closeEvent:
	case <- time.NewTimer(3 * time.Second).C:
		t.Error("close event timed out")
	}

	err = <- closeEvent
	assert.Nil(err, "close event error free")
}

func TestConnection_NotifyOnClose_AlreadyClosed(t *testing.T) {
	conn := getTestConnection(t)
	conn.Close()

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

	conn := getTestConnection(t)

	dialEvent := make(chan error, 1)
	conn.NotifyDial(dialEvent)

	conn.transportConn.Close()
	waitForReconnect(t, conn.transportManager, 2)

	timeout := time.NewTimer(1 * time.Second)

	select {
	case err, closed := <-dialEvent:
		assert.True(closed, "dial event channelConsume is open")
		assert.NoError(err, "dial event error is nil")
	case <- timeout.C:
		t.Error("dial channelConsume did not receive event")
	}

	conn.Close()

	select {
	case _, closed := <-dialEvent:
		assert.False(closed, "dial event channelConsume is closed")
	case <- timeout.C:
		t.Error("dial channelConsume did not receive event")
	}
}

func TestConnection_NotifyOnDial_AlreadyClosed(t *testing.T) {
	assert := assert.New(t)

	conn := getTestConnection(t)
	conn.Close()

	eventChan := make(chan error, 1)
	conn.NotifyDial(eventChan)

	timeout := time.NewTimer(1 * time.Second)
	select {
	case _, closed := <-eventChan:
		assert.False(closed, "dial event channelConsume is closed")
	case <- timeout.C:
		t.Error("dial channelConsume did not close")
	}
}

func TestConnection_NotifyOnDisconnect(t *testing.T) {
	assert := assert.New(t)

	conn := getTestConnection(t)

	disconnectEvent := make(chan error, 1)
	err := conn.NotifyDisconnect(disconnectEvent)
	assert.NoError(err, "notification subscribe")

	conn.transportConn.Close()
	waitForReconnect(t, conn.transportManager, 2)

	timeout := time.NewTimer(1 * time.Second)

	select {
	case err, closed := <-disconnectEvent:
		assert.True(closed, "disconnect event channelConsume is open")
		assert.NoError(err, "first disconnect event error is nil")
	case <- timeout.C:
		t.Error("disconnect channelConsume did not receive event")
		t.FailNow()
	}

	conn.transportConn.Close()
	waitForReconnect(t, conn.transportManager, 2)

	timeout.Reset(1 * time.Second)

	select {
	case err, open := <-disconnectEvent:
		assert.True(open, "disconnect event channelConsume is open")
		assert.NoError(err, "second disconnect event error is nil")
	case <- timeout.C:
		t.Error("disconnect channelConsume did not receive event")
	}

	conn.Close()
	timeout.Reset(1 * time.Second)

	select {
	case _, open := <-disconnectEvent:
		assert.False(open, "disconnect event channelConsume is closed")
	case <- timeout.C:
		t.Error("disconnect channelConsume did not close")
	}
}

func TestConnection_NotifyOnDisconnect_AlreadyClosed(t *testing.T) {
	assert := assert.New(t)

	conn := getTestConnection(t)
	conn.Close()

	eventChan := make(chan error, 1)
	err := conn.NotifyDisconnect(eventChan)
	assert.ErrorIs(err, streadway.ErrClosed, "error on register")

	timeout := time.NewTimer(1 * time.Second)
	select {
	case _, closed := <-eventChan:
		assert.False(closed, "disconnect event channelConsume is closed")
	case <- timeout.C:
		t.Error("disconnect channelConsume did not close")
	}
}
