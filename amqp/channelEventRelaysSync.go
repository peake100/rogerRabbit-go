package amqp

import (
	"context"
	streadway "github.com/streadway/amqp"
	"sync"
)

// sharedSync holds all sync objects required for a Channel and it's event relays to
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
//		- Get the new channel
//		- Allow relays to set up on the new underlying streadway/amqp.Channel
//      - Once all relay set is complete, allow them to start relaying events from the
//        new streadway/amqp.Channel.
//      - Release the channel to be used by the caller only once all relays are running
//        the new leg.
type sharedSync struct {
	// transportCtx is the context of the channel. Relays should exit when this context
	// is cancelled.
	transportCtx context.Context

	// transportLock is the transportManager lock. Relays should acquire for read when
	// setting up (first time only) so the sync between the relay and it's livesOnce
	// only has a single entry point (between reconnection events, but not during them).
	transportLock *sync.RWMutex

	// legComplete should be added to by eventRelays when they spin up and released
	// when they have finished processing all available events after an underlying
	// streadway/amqp.Channel close.
	//
	// This will block a reconnection from being available for new work until all
	// events from the previous streadway/amqp.Channel have been relayed.
	legComplete *sync.WaitGroup

	// runSetup WaitGroup will be closed by transportManager when a new underlying
	// streadway/amqp.Channel is established but not yet available to method callers,
	// allowing eventRelays to grab information about the new streadway/amqp.Channel and
	// remain confident that we did not miss a Channel while wrapping up event
	// processing from a previous channel in a situation where we are rapidly connecting
	// and disconnecting
	runSetup *sync.WaitGroup

	// setupComplete should be added to before an eventRelay releases legComplete and
	// released after an eventRelay has acquired all the info it needs from a new
	// streadway/amqp.Channel.
	//
	// Once this WaitGroup is fully released by all relays, the write lock on the
	// channel will be released and channel methods will become available to callers
	// again.
	setupComplete *sync.WaitGroup

	// runLeg will be released by the transportManager when all relays are ready.
	// eventRelays will wait on this group after releasing setupComplete so that they
	// don't error out and re-enter the beginning of their lifecycle before runSetup has
	// been re-acquired by the transportManager.
	runLeg *sync.WaitGroup
}

// newSharedSync returns a new sharedSync to coordinate sync between a Channel and it's
// event relays.
func newSharedSync(channel *Channel) sharedSync {
	shared := sharedSync{
		transportCtx:  channel.ctx,
		transportLock: channel.transportLock,
		legComplete:   new(sync.WaitGroup),
		runSetup:      new(sync.WaitGroup),
		setupComplete: new(sync.WaitGroup),
		runLeg:        new(sync.WaitGroup),
	}

	shared.runSetup.Add(1)
	return shared
}

// channelRelaySync exposes methods for the transportManager to block on / signal
// for it's half of the sharedSync.
type channelRelaySync struct {
	shared sharedSync
}

// AllowSetup signals to all relays to run setup on a new streadway/amqp.Channel. It
// should be called by the transportManager when all relays have completed their last
// leg, and a new channel is available after reconnection.
//
// The livesOnce manager should block on WaitOnSetup after calling this method.
func (chanSync *channelRelaySync) AllowSetup() {
	chanSync.shared.runLeg.Add(1)
	chanSync.shared.runSetup.Done()
}

// WaitOnSetup will block until all eventRelays signal that their setup is complete.
//
// The transportManager should wait on this before calling AllowRelayLegRun.
func (chanSync *channelRelaySync) WaitOnSetup() {
	defer chanSync.shared.setupComplete.Wait()
}

// AllowRelayLegRun signals all relays to being their next leg.
//
// The livesOnce manager should block on
func (chanSync *channelRelaySync) AllowRelayLegRun() {
	// Acquire the run setup group to stop relays from running setup on
	chanSync.shared.runSetup.Add(1)
	chanSync.shared.runLeg.Done()
}

// WaitOnLegComplete will block until all relays have reported that all events from the
// previous underlying streadway/amqp.Channel have been relayed.
//
// The transportManger should block on this before reconnecting to a nwe channel and
// calling AllowSetup.
func (chanSync *channelRelaySync) WaitOnLegComplete() {
	defer chanSync.shared.legComplete.Wait()
}

// relaySync exposes methods for single eventRelay runner to block on / signal for it's
// half of the sharedSync.
//
// relaySync is not concurrency-safe, and each eventRelay runner should get it's own
// relaySync object.
type relaySync struct {
	// done is whether the relay is complete.
	done bool

	// sharedSync object
	shared sharedSync

	// firstSetupDone will be set to true after the first eventRelay.SetupForRelayLeg()
	// call.
	firstSetupDone bool
	firstSetupWait chan struct{}

	// setupCompleteHeld is set to true when sharedSync.SetupComplete.Add(1) is called
	// and set to false when sharedSync.SetupComplete.Done() is called.
	setupCompleteHeld bool

	// legCompleteHeld is set to true when sharedSync.legComplete.Add(1) is called
	// and set to false when sharedSync.legComplete.Done() is called.
	legCompleteHeld bool
}

// SetDone sets whether this relay is done relaying events.
func (relaySync *relaySync) SetDone(done bool) {
	relaySync.done = done
}

// Complete should be called when an eventRelay runner when exiting, and releases any
// held WaitGroups.
func (relaySync *relaySync) Complete() {
	defer func() {
		relaySync.setupCompleteHeld = false
		relaySync.legCompleteHeld = false
	}()

	if relaySync.setupCompleteHeld {
		defer relaySync.shared.setupComplete.Done()
	}

	if relaySync.legCompleteHeld {
		defer relaySync.shared.legComplete.Done()
	}
}

// IsDone returns whether the relay is done, either from a context cancellation or the
// relay has reported it is finished.
func (relaySync *relaySync) IsDone() bool {
	return relaySync.done || relaySync.shared.transportCtx.Err() != nil
}

// WaitToSetup blocks until the transportManager signals that relay setup should begin.
//
// The eventRelay runner should then call eventRelay.SetupForRelayLeg().
func (relaySync *relaySync) WaitToSetup() {
	// If this is the first time we are setting up, we are going to grab the livesOnce
	// lock so that we can set up without waiting for the connection to go down.
	if !relaySync.firstSetupDone {
		relaySync.shared.transportLock.RLock()
	} else {
		// Otherwise wait until the channelReconnect gives us the all clear
		relaySync.shared.runSetup.Wait()
	}
}

// SetupComplete should be called by the eventRelay runner after
// eventRelay.SetupForRelayLeg() returns before blocking on WaitToRunLeg.
func (relaySync *relaySync) SetupComplete() {
	// If this was the first setup, release the livesOnce lock.
	if !relaySync.firstSetupDone {
		defer relaySync.shared.transportLock.RUnlock()
		// Mark that we are through the first setup so we do not try to acquire the
		// livesOnce lock on the next go-around
		relaySync.firstSetupDone = true
		close(relaySync.firstSetupWait)
	} else {
		defer func() {
			relaySync.shared.setupComplete.Done()
			relaySync.setupCompleteHeld = false
		}()
	}

	// Grab the relays running group before we release the setup complete group.
	relaySync.legCompleteHeld = true
	relaySync.shared.legComplete.Add(1)
}

// WaitToRunLeg blocks until the transportManager signals that events should start being
// processed.
//
// Once this method releases, eventRelay runners should call eventRelay.RunRelayLeg()
func (relaySync *relaySync) WaitToRunLeg() {
	// Wait for the channel to communicate that we can begin processing again.
	relaySync.shared.runLeg.Wait()
}

// RunLegComplete should be called by the eventRelay runner after
// eventRelay.RunRelayLeg() returns before blocking on WaitToSetup.
func (relaySync *relaySync) RunLegComplete() {
	defer func() {
		// Release the hold we have on the running group
		relaySync.shared.legComplete.Done()
		relaySync.legCompleteHeld = false
	}()

	// Acquire the setup complete group before releasing the running group.
	relaySync.setupCompleteHeld = true
	relaySync.shared.setupComplete.Add(1)
}

// WaitForInit blocks until EITHER the relay successfully starts relaying events OR
// the passed context cancels.
//
// This allows for methods like Channel.NotifyPublish to block until we are sure the
// relay is properly connected or the roger Channel is closed.
func (relaySync *relaySync) WaitForInit(ctx context.Context) error {
	select {
	case <-relaySync.firstSetupWait:
	case <-ctx.Done():
		return streadway.ErrClosed
	}

	return nil
}

// newRelaySync created a new relaySync for an eventRelay runner from an underlying
// sharedSync object being tracked by the transportManager.
func newRelaySync(shared sharedSync) *relaySync {
	return &relaySync{
		done:              false,
		shared:            shared,
		firstSetupDone:    false,
		firstSetupWait:    make(chan struct{}),
		setupCompleteHeld: false,
		legCompleteHeld:   false,
	}
}
