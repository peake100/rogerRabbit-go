package amqp

import (
	"context"
	streadway "github.com/streadway/amqp"
	"sync/atomic"
)

// reconnectRedialOnce attempts to reconnect the livesOnce a single time.
func (manager *transportManager) reconnectRedialOnce(ctx context.Context) error {
	// Make the connection.
	err := manager.transport.tryReconnect(
		ctx, atomic.LoadUint64(manager.reconnectCount),
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
	for {
		// Check to see if our context has been cancelled, and exit if so.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := manager.reconnectRedialOnce(ctx)
		// If no error OR there is an error and retry is false return.
		if err == nil || (err != nil && !retry) {
			return err
		}
	}
}

// reconnectListenForClose listens for a close event from the underlying livesOnce, and
// starts the reconnection process.
func (manager *transportManager) reconnectListenForClose(
	closeChan <-chan *streadway.Error,
) {
	// Wait for the current connection to close
	disconnectEvent := <-closeChan
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
	// Lock access to the connection and don't unlock until we have reconnected.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Redial the broker until we reconnectMiddleware
	err := manager.reconnectRedial(ctx, retry)
	if err != nil {
		return err
	}

	// Broadcast that we have made a successful reconnection to any one-time listeners.
	manager.reconnectCond.Broadcast()

	// Register a notification channelConsume for the new connection's closure.
	closeChan := make(chan *streadway.Error, 1)
	manager.transport.underlyingTransport().NotifyClose(closeChan)

	// Launch a goroutine to reconnectMiddleware on connection closure.
	go manager.reconnectListenForClose(closeChan)

	return nil
}
