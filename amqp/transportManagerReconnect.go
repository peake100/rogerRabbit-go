package amqp

import (
	"context"
	streadway "github.com/streadway/amqp"
)

func (manager *transportManager) reconnectRedialOnce(ctx context.Context) error {
	// Make the connection.
	if manager.logger.Debug().Enabled() {
		manager.logger.
			Debug().
			Uint64("RECONNECT_COUNT", manager.reconnectCount).
			Msg("attempting connection")
	}
	err := manager.transport.tryReconnect(ctx)
	if err != nil {
		manager.logger.Debug().Err(err).Msg("error re-dialing connection")
	}
	// Send a notification to all listeners subscribed to dial events.
	manager.sendConnectNotifications(err)
	if err != nil {
		manager.logger.
			Error().
			Err(err).
			Uint64("RECONNECT_COUNT", manager.reconnectCount).
			Msg("reconnect error")

		// Otherwise, return (and possibly try again).
		return err
	}

	// Increment our reconnection count tracker.
	manager.reconnectCount++

	if manager.logger.Info().Enabled() {
		manager.logger.Info().
			Uint64("RECONNECT_COUNT", manager.reconnectCount).
			Msg("AMQP BROKER CONNECTED")
	}

	// If there was no error, break out of the loop.
	return nil
}

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

func (manager *transportManager) reconnectListenForClose(
	closeChan <-chan *streadway.Error,
) {
	// Wait for the current connection to close
	disconnectEvent := <-closeChan
	if manager.logger.Info().Enabled() {
		manager.logger.Info().Msgf(
			"AMQP BROKER DISCONNECTED: %v", disconnectEvent,
		)
	}
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

func (manager *transportManager) reconnect(ctx context.Context, retry bool) error {
	// Lock access to the connection and don't unlock until we have reconnected.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Redial the broker until we reconnect
	err := manager.reconnectRedial(ctx, retry)
	if err != nil {
		return err
	}

	// Broadcast that we have made a successful reconnection to any one-time listeners.
	manager.reconnectCond.Broadcast()

	// Register a notification ChannelConsume for the new connection's closure.
	closeChan := make(chan *streadway.Error, 1)
	manager.transport.NotifyClose(closeChan)

	// Launch a goroutine to reconnect on connection closure.
	go manager.reconnectListenForClose(closeChan)

	return nil
}
