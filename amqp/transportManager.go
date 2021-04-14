package amqp

import (
	"context"
	"errors"
	"fmt"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// livesOnce is a common interface between the *streadway.Connection and
// *streadway.Channel. We want o abstract away identical operations we need to implement
// call on these for reconnecting to a new underlying underlyingTransport.
type livesOnce interface {
	NotifyClose(receiver chan *streadway.Error) chan *streadway.Error
	Close() error
}

// reconnects is the interface we need to implement for a livesOnce capable of
// reconnecting itself.
type reconnects interface {
	transportType() amqpmiddleware.TransportType

	// livesOnce returns  the underlying livesOnce (streadway.Connection or
	// streadway.Channel)
	underlyingTransport() livesOnce

	// tryReconnect is needed for both Connection and Channel. This method
	// attempts to re-establish a connection for the underlying object exactly once.
	tryReconnect(ctx context.Context, attempt uint64) error

	// cleanup releases any resources. Called on final close AFTER the main context is
	// cancelled but before the current underlying connection is closed. NOTE:
	// re-connects will NOT occur once this method is called, so cleanup implementation
	// must be able to handle a disconnected underlying connection.
	cleanup() error
}

// TestReconnectSignaler allows us to block until a reconnection occurs during a test.
type TestReconnectSignaler struct {
	// The test we are using.
	t *testing.T

	// reconnectSignal will close when a reconnection occurs.
	reconnectSignal chan struct{}

	original livesOnce
	manager  *transportManager
}

// WaitOnReconnect blocks until a reconnection of the underlying livesOnce occurs. Once
// the first reconnection event occurs, this object will no longer block and a new
// signaler will need to be created for the next re-connection.
//
// If no context is passed a context with 3-second timeout will be used.
func (signaler *TestReconnectSignaler) WaitOnReconnect(ctx context.Context) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	for signaler.original == signaler.manager.transport.underlyingTransport() {
		select {
		case <-signaler.reconnectSignal:
		case <-ctx.Done():
			signaler.t.Error(
				"context cancelled before reconnection occurred: %w", ctx.Err(),
			)
			signaler.t.FailNow()
		}
	}
}

// TransportTesting provides testing methods for testing Channel and Connection.
type TransportTesting struct {
	t       *testing.T
	manager *transportManager
	// The number of times a connection has been blocked from being acquired.
	blocks *int32
}

// TransportLock is the underlying lock which controls access to the channel /
// connection. When held for read or write, a reconnection of the underlying livesOnce
// cannot occur.
func (tester *TransportTesting) TransportLock() *sync.RWMutex {
	return tester.manager.transportLock
}

// BlockReconnect blocks a livesOnce from reconnecting. If too few calls to
// UnblockReconnect() are made, the block will be removed at the end of the test.
func (tester *TransportTesting) BlockReconnect() {
	atomic.AddInt32(tester.blocks, 1)
	tester.manager.transportLock.RLock()
}

// UnblockReconnect unblocks the livesOnce from reconnecting after calling
// BlockReconnect()
func (tester *TransportTesting) UnblockReconnect() {
	defer tester.manager.transportLock.RUnlock()
	atomic.AddInt32(tester.blocks, -1)
}

// cleanup calls UnblockReconnect the required number of times to unblock the channel
// from reconnecting during test cleanup.
func (tester *TransportTesting) cleanup() {
	blocks := *tester.blocks
	for i := int32(0); i < blocks; i++ {
		tester.manager.transportLock.RUnlock()
	}
}

// SignalOnReconnect returns a signaler that can be used to wait on the next
// reconnection event of the livesOnce.
func (tester *TransportTesting) SignalOnReconnect() *TestReconnectSignaler {
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

	signaler := &TestReconnectSignaler{
		t:               tester.t,
		reconnectSignal: reconnected,
		original:        tester.manager.transport.underlyingTransport(),
		manager:         tester.manager,
	}

	return signaler
}

// DisconnectTransport closes the underlying livesOnce to force a reconnection.
func (tester *TransportTesting) DisconnectTransport() {
	err := tester.manager.transport.underlyingTransport().Close()
	if !assert.NoError(tester.t, err, "close underlying livesOnce") {
		tester.t.FailNow()
	}
}

// ForceReconnect forces a disconnect of the channel or connection and waits for a
// reconnection to occur or ctx to cancel. If a nil context is passed, a context with
// a 3-second timeout will be used instead.
func (tester *TransportTesting) ForceReconnect(ctx context.Context) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	// Get the current transport
	tester.TransportLock().RLock()
	current := tester.manager.transport.underlyingTransport()
	// Wait until the underlying transport pointer has changed. The first time we signal
	// might be from the previous reconnect.
	tester.TransportLock().RUnlock()

	for current == tester.manager.transport.underlyingTransport() {
		// Register a channel to be closed when a reconnection occurs
		reconnected := tester.SignalOnReconnect()

		// Disconnect the livesOnce
		tester.DisconnectTransport()

		// Wait for the connection to be re-connected.
		reconnected.WaitOnReconnect(ctx)
		// If the context has cancelled, return.
		if ctx.Err() != nil {
			return
		}
	}
}

// transportManager handles lifetime of underlying livesOnce method, such as
// reconnections, closures, and connection status subscribers. To be embedded into the
// Connection and Channel types for free implementation of common methods.
type transportManager struct {
	// The master context of the robust connection. Cancellation of this context
	// should keep the connection from re-dialing and close the current connection.
	ctx context.Context
	// Cancel func that cancels our main context.
	cancelFunc context.CancelFunc

	// Our core transport object.
	transport reconnects
	// Lock to control the transport.
	transportLock *sync.RWMutex
	// This value is incremented every time we re-connect to the broker.
	reconnectCount *uint64
	// sync.Cond that broadcasts whenever a connection is successfully re-established
	reconnectCond *sync.Cond

	// handlers are the underlying method and event handlers for this transport's
	// methods, in order to allow user-defined middleware on both channels and
	// connections.
	handlers transportManagerHandlers

	// notificationSubscribersDial is the list of channels to send a connection
	// established notification to.
	notificationSubscribersDial []chan error
	// notifyDialEventHandlers is the list of event handles used to send NotifyDial
	// events.
	notifyDialEventHandlers []amqpmiddleware.HandlerNotifyDialEvents

	// List of channels to send a connection lost notification to.
	notificationSubscriberDisconnect []chan error
	// notifyDisconnectEventHandlers is the list of event handles used to send
	// NotifyDisconnect events.
	notifyDisconnectEventHandlers []amqpmiddleware.HandlerNotifyDisconnectEvents

	// List of channels to send a connection close notification to.
	notificationSubscriberClose []chan *streadway.Error
	// notifyCloseEventHandlers is the list of event handles used to send NotifyClose
	// events.
	notifyCloseEventHandlers []amqpmiddleware.HandlerNotifyCloseEvents
	// Mutex for notification subscriber lists. Allows subscribers to be added during
	// an active redial loop.
	notificationSubscriberLock *sync.Mutex
}

// repeatErrCodes are errors that should cause an automatic reattempt of an operation.
// We want to automatically re-run any operations that fail
var repeatErrCodes = [3]int{
	streadway.ChannelError,
	streadway.UnexpectedFrame,
	streadway.FrameError,
}

// isRepeatErr returns true if err is an error we should reattempt an operation on.
func isRepeatErr(err error) bool {
	// If there was an error, check and see if it is an error we should try again
	// on.

	// If this is an io.EOF error, there was probably an aborted connection during
	// broker startup. We can try again with a new channel.
	if errors.Is(err, io.EOF) {
		return true
	}

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

// revive:disable:context-as-argument - we have two contexts here, they can't both be first.

// retryOperationOnClosedSingle attempts a Connection or Channel channel method a single
// time.
func (manager *transportManager) retryOperationOnClosedSingle(
	transportCtx context.Context,
	opCtx context.Context,
	operation func(ctx context.Context) error,
) error {
	// If the context of our robust livesOnce mechanism is closed, return an
	// ErrClosed.
	if transportCtx.Err() != nil {
		return streadway.ErrClosed
	}

	if opCtx.Err() != nil {
		return fmt.Errorf("operation context closed: %w", opCtx.Err())
	}

	// Run the operation in a closure so we can acquire and release the
	// transportLock using defer.
	var err error
	func() {
		// Acquire the transportLock for read. This allow multiple operation to
		// occur at the same time, but blocks the connection from being switched
		// out until the operations resolve.
		//
		// We don't need to worry about lock contention, as once the livesOnce
		// reconnection routine requests the lock, and new read acquisitions will
		// be blocked until the lock is acquired and released for write.
		manager.transportLock.RLock()
		defer manager.transportLock.RUnlock()

		err = operation(opCtx)
	}()
	return err
}

// revive:enable:context-as-argument

// retryOperationOnClosed repeats operation / method call until a non-closed error is
// returned or ctx expires. This is a helper method for implementing methods like
// Channel.QueueBind, in which we want to retry the operation if our underlying
// livesOnce mechanism has connection issues.
func (manager *transportManager) retryOperationOnClosed(
	transportCtx context.Context,
	operation func(ctx context.Context) error,
	retry bool,
) error {
	attempt := 0
	for {
		opCtx := context.WithValue(transportCtx, "opAttempt", attempt)
		err := manager.retryOperationOnClosedSingle(transportCtx, opCtx, operation)

		// If there was no error, we are not trying, or this is not a repeat error,
		// or our context has cancelled: return.
		if err == nil || !retry || !isRepeatErr(err) || transportCtx.Err() != nil {
			return err
		}

		// Otherwise retry the operation once the connection has been established.
		attempt++
	}
}

// sendDialNotifications sends results of a dial attempt to all
// transportManager.NotifyDial subscribers.
func (manager *transportManager) sendDialNotifications(err error) {
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

	// We are going to send the close error (if any) through the event handlers.
	event := amqpmiddleware.EventNotifyDial{
		TransportType: manager.transport.transportType(),
		Err:           err,
	}

	// Notify all our close subscribers.
	for _, receiver := range manager.notifyDialEventHandlers {
		receiver(make(amqpmiddleware.EventMetadata), event)
	}
}

// sendDisconnectNotifications sends the error from NotifyClose of the underlying
// connection when a disconnect occurs to all NotifyOnDisconnect subscribers.
func (manager *transportManager) sendDisconnectNotifications(
	streadwayErr *streadway.Error,
) {
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

	// Even if the pointer is nil, setting an error field to a concrete nil pointer
	// results in a non-nil error interface, so we only want to set this if it's NOT
	// nil.
	var err error
	if streadwayErr != nil {
		err = streadwayErr
	}

	// We need to send an explicit nil on a nil pointer as a nil pointer with a
	// concrete type is, weirdly, a non-nil error interface value.
	event := amqpmiddleware.EventNotifyDisconnect{
		TransportType: manager.transport.transportType(),
		Err:           err,
	}

	// Notify all our close subscribers.
	for _, handler := range manager.notifyDisconnectEventHandlers {
		handler(make(amqpmiddleware.EventMetadata), event)
	}
}

// sendCloseNotifications sends notification to all NotifyOnClose subscribers.
func (manager *transportManager) sendCloseNotifications(err *streadway.Error) {
	manager.notificationSubscriberLock.Lock()
	defer manager.notificationSubscriberLock.Unlock()

	// Notify all our close subscribers.
	event := amqpmiddleware.EventNotifyClose{
		TransportType: manager.transport.transportType(),
		Err:           err,
	}
	for _, receiver := range manager.notifyCloseEventHandlers {
		receiver(make(amqpmiddleware.EventMetadata), event)
	}
}

// we can get away with making methods like NotifyClose a non-pinter type because the
// method handlers have already captured a pointer to the manager.

// NotifyClose is as NotifyClose on streadway Connection/Channel.NotifyClose.
// Subscribers to Close events will not be notified when a reconnection occurs under
// the hood, only when the roger Connection or Channel is closed by calling the Close
// method. This mirrors the streadway implementation, where Close events are only sent
// once when the livesOnce object becomes unusable.
//
// For finer-grained connection status, see NotifyDial and NotifyDisconnect, which
// will both send individual events when the connection is lost or re-acquired.
func (manager *transportManager) NotifyClose(
	receiver chan *streadway.Error,
) chan *streadway.Error {
	args := amqpmiddleware.ArgsNotifyClose{
		Receiver:      receiver,
		TransportType: manager.transport.transportType(),
	}
	return manager.handlers.notifyClose(manager.ctx, args).CallerChan
}

// NotifyDial is new for robust Roger transportType objects. NotifyDial will send all
// subscribers an event notification every time we try to re-acquire a connection. This
// will include both failure AND successes.
func (manager *transportManager) NotifyDial(
	receiver chan error,
) error {
	args := amqpmiddleware.ArgsNotifyDial{
		TransportType: manager.transport.transportType(),
		Receiver:      receiver,
	}
	return manager.handlers.notifyDial(manager.ctx, args)
}

// NotifyDisconnect is new for robust Roger transportType objects. NotifyDisconnect will
// send all subscribers an event notification every time the underlying connection is
// lost.
func (manager *transportManager) NotifyDisconnect(
	receiver chan error,
) error {
	args := amqpmiddleware.ArgsNotifyDisconnect{
		TransportType: manager.transport.transportType(),
		Receiver:      receiver,
	}
	return manager.handlers.notifyDisconnect(manager.ctx, args)
}

// cancelCtxCloseTransport cancels the main context and closes the underlying connection
// during shutdown.
func (manager *transportManager) cancelCtxCloseTransport() {
	// Grab the notification subscriber lock so new subscribers will not get added
	// without seeing the context cancel.
	manager.notificationSubscriberLock.Lock()

	// Cancel the context the tryReconnect this closure will cause exits.
	manager.cancelFunc()

	// Release the notification lock. Not doing so before we grab the livesOnce lock
	// can result in a deadlock if a redial is in process (since the redial needs to
	// grab the subscribers lock to notify them).
	manager.notificationSubscriberLock.Unlock()

	// Take control of the connection lock to ensure all in-process operations have
	// completed.
	manager.transportLock.Lock()
	defer manager.transportLock.Unlock()

	// Close the current connection on exit
	defer manager.transport.underlyingTransport().Close()
}

// Close the robust connection. This both closes the current connection and keeps it
// from reconnecting.
func (manager *transportManager) Close() error {
	args := amqpmiddleware.ArgsClose{TransportType: manager.transport.transportType()}
	return manager.handlers.transportClose(manager.ctx, args)
}

// IsClosed returns true if the connection is marked as closed, otherwise false
// is returned.
//
// --
//
// ROGER NOTE: unlike streadway/amqp, which only implements IsClosed() on connection
// objects, rogerRabbit makes IsClosed() available on both connections and channels.
// IsClosed() will return false until the connection / channel is shut down, even if the
// underlying connection is currently disconnected and waiting to reconnectMiddleware.
func (manager *transportManager) IsClosed() bool {
	// If the context is cancelled, the livesOnce is closed.
	return manager.ctx.Err() != nil
}

// Test methods for the livesOnce
func (manager *transportManager) Test(t *testing.T) *TransportTesting {
	return &TransportTesting{
		t:       t,
		manager: manager,
	}
}

// newTransportManager returns a new transportManager for a given livesOnce.
func (manager *transportManager) setup(
	ctx context.Context,
	transport reconnects,
	middleware transportManagerMiddleware,
) {
	ctx, cancelFunc := context.WithCancel(ctx)

	reconnectCount := uint64(0)

	manager.ctx = ctx
	manager.cancelFunc = cancelFunc
	manager.transport = transport
	manager.transportLock = new(sync.RWMutex)
	manager.notificationSubscriberLock = new(sync.Mutex)
	manager.reconnectCond = sync.NewCond(manager.transportLock)
	manager.reconnectCount = &reconnectCount

	// Create the base method handlers.
	manager.handlers = newTransportManagerHandlers(manager, middleware)
}
