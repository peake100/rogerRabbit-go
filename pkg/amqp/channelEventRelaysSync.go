package amqp

import (
	"context"
	streadway "github.com/streadway/amqp"
	"sync"
	"time"
)

// sharedRelaySync holds all sync objects required for a Channel and it's event relays to
// stay in sync.
//
// Relays need to be carefully controlled so that they do not miss a short-lived channel
// while working on events from a previous channel. The flow must go:
//
// 1. A new relay can be setup while a channel is serving requests.
// 2. After a relay sets up, it begins serving events from the current underlying
//    channel.
// 3. When a channel disconnects, the transportManager must:
//		- Wait for all currently running relays to finish their work.
//		- Get the new channel.
//		- Allow the relays to setup on the channel BEFORE callers can access channel methods.
//		- Once setup is Restart the relays.
type sharedRelaySync struct {
	// ctx should be cancelled by the relay on an abort, or by the channel on
	// an unresponsive relay.
	ctx    context.Context
	cancel context.CancelFunc

	// beginLeg signals to the eventRelay that it should setup for this leg on Channel.
	// Relays can once on to running once they are done setting up and te manager has
	// received this signal.
	beginLeg chan *streadway.Channel

	// setup complete signals the transport manager that the relay is done setting up
	// and ready for the channel to be released back to the method callers.
	setupComplete chan struct{}

	// legComplete signals to the transportManager that this relay has finished running.
	legComplete chan struct{}
}

// newSharedSync creates a new shared sync object.
func newSharedSync(transportCtx context.Context) sharedRelaySync {
	ctx, cancel := context.WithCancel(transportCtx)

	return sharedRelaySync{
		ctx:           ctx,
		cancel:        cancel,
		beginLeg:      make(chan *streadway.Channel),
		setupComplete: make(chan struct{}),
		legComplete:   make(chan struct{}),
	}
}

// managerRelaySync exposes methods for the transportManager to block on / signal
// for it's half of the sharedRelaySync.
type managerRelaySync struct {
	shared     []sharedRelaySync
	sharedLock *sync.Mutex

	// legCompleteWaited tracks whether we have waited for the leg to complete already.
	// WaitOnLegComplete is called every time a reconnect *attempt* occurs, so we need
	// to track whether we have to wait again, or if we've waited once already since
	// the last disconnect.
	legCompleteWaited bool
}

// AddRelay adds a relay to the manager.
func (managerSync *managerRelaySync) AddRelay(shared sharedRelaySync) {
	managerSync.sharedLock.Lock()
	defer managerSync.sharedLock.Unlock()
	managerSync.shared = append(managerSync.shared, shared)
}

// ReleaseRelaysForLeg signals all relays to begin their next leg.
func (managerSync *managerRelaySync) ReleaseRelaysForLeg(newChan *streadway.Channel) {
	managerSync.sharedLock.Lock()
	defer managerSync.sharedLock.Unlock()
	managerSync.legCompleteWaited = false

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	// We want to remove any relays that o not signal.
	signaled := make([]sharedRelaySync, 0, len(managerSync.shared))

	for _, thisSync := range managerSync.shared {
		select {
		// Signal to a relay that it can start relaying events.
		case thisSync.beginLeg <- newChan:
		case <-thisSync.ctx.Done():
		case <-timer.C:
			// On a timeout, cancel the relay.
			thisSync.cancel()
			continue
		}
		signaled = append(signaled, thisSync)
	}

	// replace our list of shared signals with only the active relays.
	managerSync.shared = signaled
}

// WaitOnSetupComplete will block until all relays have reported that they are done
// setting up.
//
// The transportManger should block on this before reconnecting to a nwe channel.
func (managerSync *managerRelaySync) WaitOnSetupComplete() {
	managerSync.sharedLock.Lock()
	defer managerSync.sharedLock.Unlock()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	signaled := make([]sharedRelaySync, 0, len(managerSync.shared))

	for _, thisSync := range managerSync.shared {
		select {
		// Wait fot the ree.
		case <-thisSync.setupComplete:
		case <-thisSync.ctx.Done():
		case <-timer.C:
			// On a timeout, cancel the relay.
			thisSync.cancel()
			continue
		}
		signaled = append(signaled, thisSync)
	}

	managerSync.shared = signaled
}

// WaitOnLegComplete will block until all relays have reported that all events from the
// previous underlying streadway/amqp.Channel have been relayed.
//
// The transportManger should block on this before reconnecting to a nwe channel.
func (managerSync *managerRelaySync) WaitOnLegComplete() {
	managerSync.sharedLock.Lock()
	defer managerSync.sharedLock.Unlock()

	// If we've already waited on this and have not yet started the setup of the relays,
	// return right away. We don't need to wait again.
	if managerSync.legCompleteWaited {
		return
	}

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	signaled := make([]sharedRelaySync, 0, len(managerSync.shared))

	for _, thisSync := range managerSync.shared {
		select {
		// Wait fot the ree.
		case <-thisSync.legComplete:
		case <-thisSync.ctx.Done():
		case <-timer.C:
			// On a timeout, cancel the relay.
			thisSync.cancel()
			continue
		}
		signaled = append(signaled, thisSync)
	}

	managerSync.shared = signaled
	managerSync.legCompleteWaited = true
}

// relaySync exposes methods for single eventRelay runner to block on / signal for it's
// half of the sharedRelaySync.
//
// relaySync is not concurrency-safe, and each eventRelay runner should get it's own
// relaySync object.
type relaySync struct {
	// sharedRelaySync object
	shared sharedRelaySync
}

// SetDone signals that this relay is done serving messages.
func (relaySync relaySync) SetDone() {
	relaySync.shared.cancel()
}

// IsDone returns whether the relay is done, either from a context cancellation or the
// relay has reported it is finished.
func (relaySync relaySync) IsDone() bool {
	return relaySync.shared.ctx.Err() != nil
}

// SignalSetupComplete should be called by the eventRelay runner after
// setup has been completed for the relay but before the relay moves on to processing
// messages.
func (relaySync relaySync) SignalSetupComplete() {
	select {
	// Wait fot the ree.
	case relaySync.shared.setupComplete <- struct{}{}:
	case <-relaySync.shared.ctx.Done():
	}
}

// SignalLegComplete should be called by the eventRelay runner after
// eventRelay.RunRelayLeg() returns before blocking on WaitToSetup.
func (relaySync relaySync) SignalLegComplete() {
	select {
	// Wait fot the ree.
	case relaySync.shared.legComplete <- struct{}{}:
	case <-relaySync.shared.ctx.Done():
	}
}

// WaitForNextLeg blocks until the transportManager signals that relay setup should
// begin.
//
// The eventRelay runner should then call eventRelay.SetupForRelayLeg().
func (relaySync relaySync) WaitForNextLeg() *streadway.Channel {
	select {
	// Wait fot the manager to signal the run of the next leg or to be cancelled.
	case newChan := <-relaySync.shared.beginLeg:
		return newChan
	case <-relaySync.shared.ctx.Done():
		return nil
	}
}

// newRelaySync created a new relaySync for an eventRelay runner from an underlying
// sharedRelaySync object being tracked by the transportManager.
func newRelaySync(transportCtx context.Context) relaySync {
	return relaySync{shared: newSharedSync(transportCtx)}
}
