package amqp

import (
	"context"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

type reconnectSignaler struct {
	t               *testing.T
	reconnectSignal chan struct{}
}

// Blocks until a reconnection of the underlying transport occurs. Once the first
// reconnection event occurs, this object will no longer block and a new signaler will
// need to be created for the next re-connection.
func (signaler *reconnectSignaler) WaitOnReconnect(ctx context.Context) {
	select {
	case <-signaler.reconnectSignal:
	case <-ctx.Done():
		signaler.t.Error(
			"context cancelled before reconnection occurred: %w", ctx.Err(),
		)
		signaler.t.FailNow()
	}
}

// Provides testing methods for testing channels and connections.
type transportTesting struct {
	t       *testing.T
	manager *transportManager
	// The number of times a connection has been blocked from being acquired.
	blocks *int32
}

// The lock which controls access to the channel / connection
func (tester *transportTesting) TransportLock() *sync.RWMutex {
	return tester.manager.transportLock
}

// Block a transport from reconnecting. If too few calls to UnblockReconnect() are
// made, the block will be removed at the end of the test.
func (tester *transportTesting) BlockReconnect() {
	atomic.AddInt32(tester.blocks, 1)
	tester.manager.transportLock.RLock()
}

// Unblock the transport from reconnecting after calling BlockReconnect()
func (tester *transportTesting) UnblockReconnect() {
	defer tester.manager.transportLock.RUnlock()
	atomic.AddInt32(tester.blocks, -1)
}

func (tester *transportTesting) cleanup() {
	blocks := *tester.blocks
	for i := int32(0); i < blocks; i++ {
		tester.manager.transportLock.RUnlock()
	}
}

// Returns a signaler that can be used to wait on the next reconnection event of the
// transport.
func (tester *transportTesting) SignalOnReconnect() *reconnectSignaler {
	// Signal that the connection has been re-established
	reconnected := make(chan struct{})

	// Grab the cond lock
	tester.manager.reconnectCond.L.Lock()

	// Launch a routine to close the channel after the wait.
	go func() {
		defer tester.manager.reconnectCond.L.Unlock()
		defer close(reconnected)
		tester.manager.reconnectCond.Wait()
	}()

	signaler := &reconnectSignaler{
		t:               tester.t,
		reconnectSignal: reconnected,
	}

	return signaler
}

// Closes the underlying transport to signal a disconnect.
func (tester *transportTesting) DisconnectTransport() {
	err := tester.manager.transport.Close()
	if !assert.NoError(tester.t, err, "close underlying transport") {
		tester.t.FailNow()
	}
}

// Force a disconnect of the channel or connection and waits for a reconnection to
// occur or ctx to cancel. If a nil context is passed, a context with a 3-second timeout
// will be used instead
func (tester *transportTesting) ForceReconnect(ctx context.Context) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	// Register a channel to be closed when a reconnection occurs
	reconnected := tester.SignalOnReconnect()

	// Disconnect the transport
	tester.DisconnectTransport()

	// Wait for the connection to be re-connected.
	reconnected.WaitOnReconnect(ctx)
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
	// Mutex for notification subscriber lists. Allows subscribers to be added during
	// an active redial loop.
	notificationSubscriberLock *sync.Mutex

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

func (manager *transportManager) retryOperationOnClosedSingle(
	ctx context.Context,
	operation func() error,
) error {
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
	var err error
	func() {
		// Acquire the transportLock for read. This allow multiple operation to
		// occur at the same time, but blocks the connection from being switched
		// out until the operations resolve.
		//
		// We don't need to worry about lock contention, as once the transport
		// reconnection routine requests the lock, and new read acquisitions will
		// be blocked until the lock is acquired and released for write.
		manager.transportLock.RLock()
		defer manager.transportLock.RUnlock()

		err = operation()
	}()
	return err
}

// Repeats operation until a non-closed error is returned or ctx expires. This is a
// helper method for implementing methods like Channel.QueueBind, in which we want
// to retry the operation if our underlying transport mechanism has connection issues.
func (manager *transportManager) retryOperationOnClosed(
	ctx context.Context,
	operation func() error,
	retry bool,
) error {
	for {
		err := manager.retryOperationOnClosedSingle(ctx, operation)

		// If there was no error, we are not trying, or this is not a repeat error,
		// or our context has cancelled: return.
		if err == nil || !retry || !isRepeatErr(err) || ctx.Err() != nil {
			return err
		}

		// Otherwise retry the operation once the connection has been established.
		if manager.logger.Debug().Enabled() {
			log.Debug().Caller(1).Msgf(
				"repeating operation on error: %v", err,
			)
		}
	}
}

// Sends results of a dial attempt to all NotifyOnDial subscribers.
func (manager *transportManager) sendConnectNotifications(err error) {
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()
	// Notify all our close subscribers.
	for _, receiver := range manager.notificationSubscribersConnect {
		// We are going to send the close error (if any) and close the channelConsume.
		receiver <- err
	}
}

// Sends the error from NotifyOnClose of the underlying connection when a disconnect
// occurs to all NotifyOnDisconnect subscribers.
func (manager *transportManager) sendDisconnectNotifications(err *streadway.Error) {
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()
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
	if manager.logger.Debug().Enabled() {
		manager.logger.Debug().Msg("sending close notifications")
	}

	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()
	// Notify all our close subscribers.
	for _, receiver := range manager.notificationSubscriberClose {
		// We are going to send the close error (if any) and close the channelConsume.
		receiver <- err
		// This event is only sent once, so we can
		close(receiver)
	}

	if manager.logger.Debug().Enabled() {
		manager.logger.Debug().Msg("close notifications sent")
	}
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
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

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
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

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
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

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

func (manager *transportManager) cancelCtxCloseTransport() {
	// Grab the notification subscriber lock so new subscribers will not get added
	// without seeing the context cancel.
	manager.notificationSubscriberLock.Lock()

	// Cancel the context the tryReconnect this closure will cause exits.
	manager.cancelFunc()

	// Release the notification lock. Not doing so before we grab the transport lock
	// can result in a deadlock if a redial is in process (since the redial needs to
	// grab the subscribers lock to notify them).
	manager.notificationSubscriberLock.Unlock()

	// Take control of the connection lock to ensure all in-process operations have
	// completed.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Close the current connection on exit
	defer manager.transport.Close()
}

// Close the robust connection. This both closes the current connection and keeps it
// from reconnecting.
func (manager *transportManager) Close() error {
	// If the context has already been cancelled, we can exit.
	if manager.ctx.Err() != nil {
		return streadway.ErrClosed
	}

	manager.cancelCtxCloseTransport()

	// Close all disconnect and connect subscribers, then clear them. We don't need to
	// grab the lock for this since the cancelled context will keep any new subscribers
	// from being added.
	for _, subscriber := range manager.notificationSubscribersConnect {
		close(subscriber)
	}
	manager.notificationSubscribersConnect = nil

	for _, subscriber := range manager.notificationSubscriberDisconnect {
		close(subscriber)
	}
	manager.notificationSubscriberDisconnect = nil

	// Send closure notifications to all subscribers.
	manager.sendCloseNotifications(nil)

	// Execute any cleanup on behalf of the transport implementation.
	return manager.transport.cleanup()
}

// ROGER NOTE: unlike streadway/amqp, which only implements IsClosed() on connection
// objects, rogerRabbit makes IsClosed() available on both connections and channels.
// IsClosed() will return false until the connection / channel is shut down, even if the
// underlying connection is currently disconnected and waiting to reconnectMiddleware.
//
// --
//
// IsClosed returns true if the connection is marked as closed, otherwise false
// is returned.
func (manager *transportManager) IsClosed() bool {
	// If the context is cancelled, the transport is closed.
	return manager.ctx.Err() != nil
}

// Test methods for the transport
func (manager *transportManager) Test(t *testing.T) *transportTesting {
	return &transportTesting{
		t:       t,
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
		notificationSubscriberLock:       new(sync.Mutex),
		logger:                           logger,
	}

	manager.reconnectCond = sync.NewCond(manager.transportLock)

	return manager
}
