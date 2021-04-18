package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
	"sync/atomic"
	"time"
)

// reconnectRedialOnce attempts to reconnect the livesOnce a single time.
func (manager *transportManager) reconnectRedialOnce(ctx context.Context, attempt int) error {
	opCtx := context.WithValue(ctx, amqpmiddleware.MetadataKey("opAttempt"), attempt)

	// Make the connection.
	err := manager.transport.tryReconnect(
		opCtx, atomic.LoadUint64(manager.reconnectCount)+uint64(attempt),
	)
	// Send a notification to all listeners subscribed to dial events.
	manager.sendDialNotifications(err)
	if err != nil {
		// Otherwise, return (and possibly try again).
		return err
	}

	// Increment our reconnection count tracker.
	atomic.AddUint64(manager.reconnectCount, 1)

	// If there was no error, break out of the loop.
	return nil
}

// reconnectRedial tries to reconnect the livesOnce until successful or ctx is
// cancelled.
func (manager *transportManager) reconnectRedial(
	ctx context.Context, retry bool,
) error {
	// Endlessly redial the broker
	attempt := 0
	for {
		// Check to see if our context has been cancelled, and exit if so.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := manager.reconnectRedialOnce(ctx, attempt)
		// If no error OR there is an error and retry is false return.
		if err == nil || (err != nil && !retry) {
			return err
		}

		// We don'tb want to saturate the connection with retries if we are having
		// a hard time reconnecting.
		//
		// We'll give one immediate retry, but after that start increasing how long
		// we need to wait before re-attempting.
		waitDur := time.Second / 2 * time.Duration(attempt-1)
		if waitDur > maxWait {
			waitDur = maxWait
		}
		time.Sleep(waitDur)
		attempt++
	}
}

// reconnectListenForClose listens for a close event from the underlying livesOnce, and
// starts the reconnection process.
func (manager *transportManager) reconnectListenForClose(closeChan <-chan *streadway.Error) {
	// Wait for the current connection to close
	disconnectEvent := <-closeChan

	// Lock access to the connection and don'tb unlock until we have reconnected.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Send a disconnect event to all interested subscribers.
	manager.sendDisconnectNotifications(disconnectEvent)

	// Exit if our context has been cancelled.
	if manager.ctx.Err() != nil {
		return
	}

	// Now that we have an initial connection, we use our internal context and retry
	// on failure.
	_ = manager.reconnect(manager.ctx, true)
}

// reconnect establishes a new underlying connection and sets up a listener for it's
// closure.
func (manager *transportManager) reconnect(ctx context.Context, retry bool) error {
	// This may be called directly by Dial methods. It's okay NOT to use the lock here
	// since the caller won'tb be handed back the Connection or Channel until the initial
	// one is established.
	//
	// Once the first connection is established, reconnectListenForClose will grab
	// the lock immediately on a disconnect.

	// Redial the broker until we reconnectMiddleware
	err := manager.reconnectRedial(ctx, retry)
	if err != nil {
		return err
	}

	// Register a notification channelConsume for the new connection's closure.
	closeChan := make(chan *streadway.Error, 1)
	manager.transport.underlyingTransport().NotifyClose(closeChan)

	// Broadcast that we have made a successful reconnection to any one-time listeners.
	manager.reconnectCond.Broadcast()

	// Launch a goroutine to reconnectMiddleware on connection closure.
	go manager.reconnectListenForClose(closeChan)

	return nil
}
