package amqp

import (
	"context"
	"sync"
)

// The relay flow need to be carefully controlled so that relays do not miss a channel
// while working on events from a previous channel. The flow must go something like this
//
// 1. A new relay can be setup while a channel is serving requests.
// 2. After a relay sets up, it begins serving requests.
// 3. When a channel disconnects, the transport manager must:
//		- Wait for all currently running relays to finish their work.
//		- Get the new channel
//		- Allow relays to set up on the channel
//      - Once all relay set is complete, allow them to work on the channel.
type sharedSync struct {
	// The context of the channel. Relays should exit when this context is cancelled.
	transportCtx context.Context
	// The transport lock. Relays should acquire for read when setting up (first time
	// only).
	transportLock *sync.RWMutex

	// Event processors should grab this WaitGroup when they spin up and release it
	// when they have finished processing all available events after a channel close.
	// This will block a reconnection from being available for new work until all
	// work from the previous channel has been finished.
	legComplete *sync.WaitGroup
	// This WaitGroup will close when a new channel is opened but not yet serving
	// requests, this allows event processors to grab information about the channel
	// and remain confident that we did not miss a new channel while wrapping up event
	// processing from a previous channel in a situation where we are rapidly connecting
	// and disconnecting
	runSetup *sync.WaitGroup
	// This WaitGroup should be added to before a processor releases
	// legComplete and released after a processor has acquired all the info
	// it needs from a new channel. Once this WaitGroup is fully released, the write
	// lock on the channel will be released and channel methods will become available to
	// callers again.
	setupComplete *sync.WaitGroup
	// This WaitGroup will be released when all relays are ready. Event relays will
	// wait on this group after releasing setupComplete so that they don't
	// error out and re-enter the beginning of their lifecycle before
	// eventRelaysRunSetup has been re-acquired by the transport manager.
	runLeg *sync.WaitGroup
}

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

// Relay sync coordinator with
type channelRelaySync struct {
	shared sharedSync
}

func (chanSync *channelRelaySync) StartSetup() {
	chanSync.shared.runLeg.Add(1)
	chanSync.shared.runSetup.Done()
}

func (chanSync *channelRelaySync) WaitForRelaySetupComplete() {
	defer chanSync.shared.setupComplete.Wait()
}

func (chanSync *channelRelaySync) StartRelays() {
	// Acquire the run setup group to stop relays from running setup on
	chanSync.shared.runSetup.Add(1)
	chanSync.shared.runLeg.Done()
}

func (chanSync *channelRelaySync) WaitForRelayLegComplete() {
	defer chanSync.shared.legComplete.Wait()
}

// Handles acquiring and releasing WaitGroups for relay stages. Not concurrency safe.
type relaySync struct {
	done bool

	shared sharedSync

	firstSetupDone bool

	setupCompleteHeld bool
	legCompleteHeld   bool
}

func (relaySync *relaySync) SetDone(done bool) {
	relaySync.done = done
}

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

// Whether the relay is done, either from a context cancellation or the relay has
// reported it is finished.
func (relaySync *relaySync) IsDone() bool {
	return relaySync.done || relaySync.shared.transportCtx.Err() != nil
}

// Blocks until the relay can begin setup
func (relaySync *relaySync) WaitForSetup() {
	// If this is the first time we are setting up, we are going to grab the transport
	// lock so that we can set up without waiting for the connection to go down.
	if !relaySync.firstSetupDone {
		relaySync.shared.transportLock.RLock()
	} else {
		// Otherwise wait until the reconnect gives us the all clear
		relaySync.shared.runSetup.Wait()
	}
}

// Released and acquires necessary locks to communicate that setup is complete.
func (relaySync *relaySync) SetupComplete() {
	// If this was the first setup, release the transport lock.
	if !relaySync.firstSetupDone {
		defer relaySync.shared.transportLock.RUnlock()
		// Mark that we are through the first setup so we do not try to acquire the
		// transport lock on the next go-around
		relaySync.firstSetupDone = true
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

func (relaySync *relaySync) WaitForRunLeg() {
	// Wait for the channel to communicate that we can begin processing again.
	relaySync.shared.runLeg.Wait()
}

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
