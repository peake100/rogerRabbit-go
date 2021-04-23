package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
	"sync/atomic"
	"time"
)

// reconnect establishes a new underlying connection and sets up a listener for it's
// closure.
func (manager *transportManager) reconnect(ctx context.Context, retry bool) error {
	// This may be called directly by Dial methods. It's okay NOT to use the lock here
	// since the caller won't be handed back the Connection or Channel until the initial
	// one is established.
	//
	// Once the first connection is established, reconnectListenForClose will grab
	// the lock immediately on a disconnect.

	// Redial the broker until we reconnectMiddleware
	closeChan, err := manager.reconnectRedial(ctx, retry)
	if err != nil {
		return err
	}

	// Broadcast that we have made a successful reconnection to any one-time listeners.
	manager.reconnectCond.Broadcast()

	// Launch a goroutine to reconnectMiddleware on connection closure.
	go manager.reconnectListenForClose(closeChan)

	return nil
}


// reconnectListenForClose listens for a close event from the underlying livesOnce, and
// starts the reconnection process.
func (manager *transportManager) reconnectListenForClose(closeChan <-chan *streadway.Error) {
	// Wait for a disconnection event.
	disconnectEvent := manager.receiveDisconnectEventAndLockChannel(closeChan)

	// receiveDisconnectEventAndLockChannel already grabbed our transport lock for
	// write, so we just need to release the lock when we exit.
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

func (manager *transportManager) receiveDisconnectEventAndLockChannel(notifyClose <-chan *streadway.Error) error {
	// Wait for the current connection to close
	var disconnectEvent error

	select {
	// Get an event from the current underlying channel. Both of these paths will
	// immediately lock the channel so we don't waste any time in blocking methods
	// and reduce returned errors to the caller.
	case streadwayEvent := <-notifyClose:
		// Lock access to the connection and don't unlock until we have reconnected.
		manager.transportLock.Lock()

		// If the event is not nil, use it as our error. We need to do this because
		// a nil *streadway.Error pointer is still a non-nil error interface value.
		if streadwayEvent != nil {
			disconnectEvent = streadwayEvent
		}
	// Or get a report from an operation that there was an error in the event that
	// streadway fails to signal us.
	case disconnectEvent = <-manager.opErrorEncountered:
		// Lock access to the connection and don't unlock until we have reconnected.
		manager.transportLock.Lock()

		// Give streadway an extra 50ms to signal. We prefer to report it's error.
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()

		select {
		case streadwayEvent := <-notifyClose:
			// If it does, use this as the disconnection event.
			if streadwayEvent != nil {
				disconnectEvent = streadwayEvent
			}
		case <-timer.C:
		}
	}

	return disconnectEvent
}

// reconnectRedial tries to reconnect the livesOnce until successful or ctx is
// cancelled.
func (manager *transportManager) reconnectRedial(ctx context.Context, retry bool) (chan *streadway.Error, error) {
	// Endlessly redial the broker
	attempt := 0

	for {
		// Check to see if our context has been cancelled, and exit if so.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		closeNotifications, err := manager.reconnectRedialOnce(ctx, attempt)
		// If no error OR there is an error and retry is false return.
		if err == nil || (err != nil && !retry) {
			return closeNotifications, err
		}

		// We don't want to saturate the connection with retries if we are having
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

// reconnectRedialOnce attempts to reconnect the livesOnce a single time.
func (manager *transportManager) reconnectRedialOnce(ctx context.Context, attempt int) (chan *streadway.Error, error) {
	opCtx := context.WithValue(ctx, amqpmiddleware.MetadataKey("opAttempt"), attempt)

	// Replace the operation error alert. It might have another error in from the
	// middleware of the last attempt.
	manager.opErrorEncountered = make(chan error, 1)

	// Make the connection.
	closeNotifications, err := manager.transport.tryReconnect(
		opCtx, atomic.LoadUint64(manager.reconnectCount)+uint64(attempt),
	)
	// Send a notification to all listeners subscribed to dial events.
	manager.sendDialNotifications(err)
	if err != nil {
		// Otherwise, return (and possibly try again).
		return nil, err
	}

	// Increment our reconnection count tracker.
	atomic.AddUint64(manager.reconnectCount, 1)

	// If there was no error, break out of the loop.
	return closeNotifications, nil
}
