package amqp

import (
	"context"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

// The transport interface is a common interface between the *streadway.Connection and
// *streadway.Channel. We want o abstract away identical operations we need to implement
// on both of them
type transport interface {
	NotifyClose(receiver chan *streadway.Error) chan *streadway.Error
	Close() error
}

// Interface we need to implement for a transport capable of being reconnected
type transportReconnect interface {
	transport
	// We need to implement this method for both Connection and Channel. This method
	// attempts to re-establish a connection for the underlying object exactly once.
	tryReconnect(ctx context.Context) error
	// Clean up any resources. Called on final close AFTER the main context is cancelled
	// but before the current underlying connection is closed. NOTE: re-connects will
	// NOT once this method is called, so cleanup implementation must be able to handle
	// a disconnected underlying connection.
	cleanup() error
}

type transportTester struct {
	t *testing.T
	manager *transportManager
}

// Force a disconnect of the channel or connection and waits for a reconnection to
// occur or ctx to cancel.
func (tester *transportTester) ForceReconnect(ctx context.Context) {
	connectionEstablished := make(chan struct{})
	waitingOnReconnect := make(chan struct{})

	// Launch a goroutine to wait on a reconnect
	go func() {
		// Signal that the connection has been re-established
		defer close(connectionEstablished)

		// Grab the cond lock
		tester.manager.reconnectCond.L.Lock()
		defer tester.manager.reconnectCond.L.Unlock()


		// Signal to our main function that we have spun up our wait
		close(waitingOnReconnect)
		tester.manager.reconnectCond.Wait()
	}()

	select {
	case <-waitingOnReconnect:
	case <-ctx.Done():
		tester.t.Error(
			"context cancelled before we could wait on cond: %w", ctx.Err(),
		)
		tester.t.FailNow()
	}

	err := tester.manager.transport.Close()
	if !assert.NoError(tester.t, err, "close underlying transport") {
		tester.t.FailNow()
	}

	select {
	case <-connectionEstablished:
	case <-ctx.Done():
		tester.t.Error(
			"context cancelled before reconnection occured: %w", ctx.Err(),
		)
		tester.t.FailNow()
	}
}

// Handles lifetime of underlying transport method, such as reconnections, closures,
// and connection status subscribers. To be embedded into the Connection and Channel
// types for free implementation of common methods.
type transportManager struct {
	// The master context of the robust connection. Cancellation of this context
	// should keep the connection from re-dialing and close the current connection.
	ctx context.Context
	// Cancel func that cancels our main context.
	cancelFunc context.CancelFunc

	// Our core transport object.
	transport transportReconnect
	// The type of transport this is (CONNECTION or CHANNEL). Used for logging.
	transportType string
	// Lock to control the transport.
	transportLock *sync.RWMutex
	// This value is incremented every time we re-connect to the broker.
	reconnectCount uint64
	// sync.Cond that broadcasts whenever a connection is successfully re-established
	reconnectCond *sync.Cond

	// List of channels to send a connection established notification to.
	notificationSubscribersConnect []chan error
	// List of channels to send a connection lost notification to.
	notificationSubscriberDisconnect []chan error
	// List of channels to send a connection close notification to.
	notificationSubscriberClose []chan *streadway.Error

	// Logger
	logger zerolog.Logger
}

// Errors that should cause an automatic reattempt of an operation. We want to
// automatically re-run any operations that fail
var repeatErrCodes = [3]int{
	streadway.ChannelError,
	streadway.UnexpectedFrame,
	streadway.FrameError,
}

func isRepeatErr(err error) bool {
	// If there was an error, check and see if it is an error we should try again
	// on.
	var streadwayErr *streadway.Error
	if errors.As(err, &streadwayErr) {
		for _, repeatCode := range repeatErrCodes {
			if streadwayErr.Code == repeatCode {
				return true
			}
		}
	}

	return false
}

// Repeats operation until a non-closed error is returned or ctx expires. This is a
// helper method for implementing methods like Channel.QueueBind, in which we want
// to retry the operation if our underlying transport mechanism has connection issues.
func (manager *transportManager) retryOperationOnClosed(
	ctx context.Context,
	operation func() error,
	retry bool,
) error {
	var err error

	for {
		// If the context of our robust transport mechanism is closed, return an
		// ErrClosed.
		if ctx.Err() != nil {
			if manager.logger.Debug().Enabled() {
				log.Debug().Caller(1).Msg(
					"operation attempted after context close",
				)
			}
			return streadway.ErrClosed
		}

		// Run the operation in a closure so we can acquire and release the
		// transportLock using defer.
		func(){
			// Acquire the transportLock for read. This allow multiple operation to
			// occur at the same time, but blocks the connection from being switched
			// out until the operations resolve.
			manager.transportLock.RLock()
			defer manager.transportLock.RUnlock()

			// TODO: to reduce lock contention, we should subscribe to transport
			// 	re-establishment notifications here. But to do that we also need to
			// 	implement unsubscribing to notifications or subscribing to one-time
			//	notifications.
			err = operation()
		}()

		// If there was no error, exit.
		if err == nil || !retry {
			return nil
		}


		if isRepeatErr(err) {
			if manager.logger.Debug().Enabled() {
				log.Debug().Caller(1).Msgf(
					"repeating operation on error: %v", err,
				)
			}
			continue
		}

		// If it's not a retry error, return it.
		return err
	}
}


// Sends results of a dial attempt to all NotifyOnDial subscribers.
func (manager *transportManager) sendConnectNotifications(err error) {
	// Notify all our close subscribers.
	for _, receiver := range manager.notificationSubscribersConnect {
		// We are going to send the close error (if any) and close the channelConsume.
		receiver <- err
	}
}

// Sends the error from NotifyOnClose of the underlying connection when a disconnect
// occurs to all NotifyOnDisconnect subscribers.
func (manager *transportManager) sendDisconnectNotifications(err *streadway.Error) {
	// Notify all our close subscribers.
	for _, receiver := range manager.notificationSubscriberDisconnect {
		// We need to send an explicit nil on a nil pointer as a nil pointer with a
		// concrete type is, weirdly, a non-nil error interface value.
		var event error
		if err != nil {
			event = err
		}
		receiver <- event
	}
}

// Sends notification to all NotifyOnClose subscribers.
func (manager *transportManager) sendCloseNotifications(err *streadway.Error) {
	// Notify all our close subscribers.
	for _, receiver := range manager.notificationSubscriberClose {
		// We are going to send the close error (if any) and close the channelConsume.
		receiver <- err
		// This event is only sent once, so we can
		close(receiver)
	}
}

func (manager *transportManager) reconnect(ctx context.Context, retry bool) error {
	// Lock access to the connection and don't unlock until we have reconnected.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Endlessly redial the broker
	for {
		// Check to see if our context has been cancelled, and exit if so.
		if ctx.Err() != nil {
			if manager.logger.Debug().Enabled() {
				manager.logger.
					Debug().
					Msg("context cancelled before transport reconnected")
			}
			return ctx.Err()
		}

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

			// If this is our retry connection, we want to return the error and allow
			// the user to decide whether or not to retry.
			if !retry {
				return err
			}
			// Otherwise, try again.
			continue
		}

		manager.reconnectCount++

		if manager.logger.Info().Enabled() {
			manager.logger.Info().
				Uint64("RECONNECT_COUNT", manager.reconnectCount).
				Msg("AMQP BROKER CONNECTED")
		}

		// If there was no error, break out of the loop.
		break
	}

	// Broadcast that we have made a successful reconnection to any one-time listeners.
	manager.reconnectCond.Broadcast()

	// Register a notification channelConsume for the new connection's closure.
	closeChan := make(chan *streadway.Error, 1)
	manager.transport.NotifyClose(closeChan)

	// Launch a goroutine to tryReconnect on connection closure.
	go func() {
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
	}()

	return nil
}

// As NotifyClose on streadway Connection/Channel. Subscribers to Close events will not
// be notified when a reconnection occurs under the hood, only when the roger Connection
// or Channel is closed by calling the Close method. This mirrors the streadway
// implementation, where Close events are only sent once when the transport object
// becomes unusable.
//
// For finer grained connection status, see NotifyDial and NotifyDisconnect, which
// will both send individual events when the connection is lost or re-acquired.
func (manager *transportManager) NotifyClose(
	receiver chan *streadway.Error,
) chan *streadway.Error {
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// If the context of the transport manager has been cancelled, close the receiver
	// and exit.
	if manager.ctx.Err() != nil {
		close(receiver)
		return receiver
	}

	manager.notificationSubscriberClose = append(
		manager.notificationSubscriberClose, receiver,
	)
	return receiver
}

// New for robust Roger Transport objects. NotifyDial will send all subscribers an
// event notification every time we try to re-acquire a connection. This will include
// both failure AND successes.
func (manager *transportManager) NotifyDial(
	receiver chan error,
) error {
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// If the context of the transport manager has been cancelled, close the receiver
	// and exit.
	if manager.ctx.Err() != nil {
		close(receiver)
		return streadway.ErrClosed
	}

	manager.notificationSubscribersConnect = append(
		manager.notificationSubscribersConnect, receiver,
	)
	return nil
}

// New for robust Roger Transport objects. NotifyDisconnect will send all subscribers an
// event notification every time a connection is lost.
func (manager *transportManager) NotifyDisconnect(
	receiver chan error,
) error {
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// If the context of the transport manager has been cancelled, close the receiver
	// and exit.
	if manager.ctx.Err() != nil {
		close(receiver)
		return streadway.ErrClosed
	}

	manager.notificationSubscriberDisconnect = append(
		manager.notificationSubscriberDisconnect, receiver,
	)
	return nil
}

// Close the robust connection. This both closes the current connection and keeps it
// from reconnecting.
func (manager *transportManager) Close() error {
	// If the context has already been cancelled, we can exit.
	if manager.ctx.Err() != nil {
		return streadway.ErrClosed
	}

	// Cancel the context the tryReconnect this closure will cause exits.
	manager.cancelFunc()

	// Take control of the connection lock to ensure all outstanding operations have
	// completed.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Close the current connection on exit
	defer manager.transport.Close()

	// Close all disconnect and connect subscribers, then clear them
	for _, subscriber := range manager.notificationSubscribersConnect {
		close(subscriber)
	}
	manager.notificationSubscribersConnect = nil

	for _, subscriber := range manager.notificationSubscriberDisconnect {
		close(subscriber)
	}
	manager.notificationSubscriberDisconnect = nil

	manager.sendCloseNotifications(nil)

	// Execute any cleanup on behalf of the transport implementation.
	return manager.transport.cleanup()
}

// Test methods for the transport
func (manager *transportManager) Test(t *testing.T) *transportTester {
	return &transportTester{
		t: t,
		manager: manager,
	}
}

func newTransportManager(
	ctx context.Context,
	transport transportReconnect,
	transportType string,
) *transportManager {
	ctx, cancelFunc := context.WithCancel(ctx)

	logger := log.Logger.With().Str("TRANSPORT", transportType).Logger()

	manager := &transportManager{
		ctx:                              ctx,
		cancelFunc:                       cancelFunc,
		transport:                        transport,
		transportType:                    transportType,
		transportLock:                    new(sync.RWMutex),
		reconnectCount:                   0,
		reconnectCond:                    nil,
		notificationSubscribersConnect:   nil,
		notificationSubscriberDisconnect: nil,
		notificationSubscriberClose:      nil,
		logger:                           logger,
	}

	manager.reconnectCond = sync.NewCond(manager.transportLock)

	return manager
}
