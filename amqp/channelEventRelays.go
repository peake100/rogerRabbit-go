package amqp

import (
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	streadway "github.com/streadway/amqp"
)

// createEventMetadata creates the amqpmiddleware.EventMetadata for an event
func createEventMetadata(legNum int, eventNum int64) amqpmiddleware.EventMetadata {
	return map[string]interface{}{
		"LegNum":   legNum,
		"EventNum": eventNum,
	}
}

// eventRelay is a common interface for relaying events from the underlying channels to
// the client without interruption. The boilerplate of handling all the synchronization
// locks will be handled for any relay passed to Channel.setupAndLaunchEventRelay()
type eventRelay interface {
	// SetupForRelayLeg is called once when the relay is first instantiated, and each
	// time there is a reconnection of the underlying channel. The raw
	// streadway/amqp.Channel is passed in for any alterations / inspections that need
	// to be made for the next leg.
	SetupForRelayLeg(newChannel *streadway.Channel) error

	// RunRelayLeg runs the relay until all events from the streadway/amqp.Channel
	// passed to SetupForRelayLeg() are exhausted,
	//
	// legNum is the leg number starting from 0 and incrementing each time this relay
	// is called.
	RunRelayLeg(legNum int) (done bool, err error)

	// Shutdown is called to exit the relay. This will usually involve closing the
	// client-facing channel.
	Shutdown() error
}

// shutdownRelay handles all the boilerplate of calling eventRelay.Shutdown.
func shutdownRelay(relay eventRelay, relaySync *relaySync) {
	// Release any outstanding WaitGroups we are holding on exit
	defer relaySync.Complete()

	// Invoke the shutdown method of the relay.
	_ = relay.Shutdown()
}

// runEventRelayCycleSetup handles the boilerplate of setting up a relay for the next
// leg.
func (channel *Channel) runEventRelayCycleSetup(
	relay eventRelay,
	relaySync *relaySync,
) (setupErr error) {
	// Release our hold on the event relays setup complete on exit.
	defer relaySync.SetupComplete()

	// Wait for a new channel to be established.
	relaySync.WaitToSetup()

	// If the relay is done, return rather than setting up for the next leg.
	if relaySync.IsDone() {
		return nil
	}

	// Set up our next leg.
	setupErr = relay.SetupForRelayLeg(channel.underlyingChannel)

	return setupErr
}

// runEventRelayCycleLeg handles the boilerplate of running an event relay leg (relaying
// all events from a single underlying streadway/ampq.Channel)
func (channel *Channel) runEventRelayCycleLeg(
	relay eventRelay, relaySync *relaySync, legNum int,
) {
	done, _ := relay.RunRelayLeg(legNum)
	relaySync.SetDone(done)

	// Log any errors.
}

// runEventRelayCycle runs a single, full cycle of setting up and running a relay leg.
func (channel *Channel) runEventRelayCycle(relay eventRelay, relaySync *relaySync, legNum int) {
	setupErr := channel.runEventRelayCycleSetup(relay, relaySync)

	if !relaySync.IsDone() {
		relaySync.WaitToRunLeg()
	}

	// Whether or not we run the leg, reset our sync to mark the run as complete.
	defer relaySync.RunLegComplete()

	// If there was no setup error and our relay is not done, run the leg.
	if setupErr == nil {
		channel.runEventRelayCycleLeg(relay, relaySync, legNum)
	}
}

// runEventRelay should be launched as goroutine to run an event relay after it's
// initial setup.
func (channel *Channel) runEventRelay(relay eventRelay, relaySync *relaySync) {
	relayLeg := 0
	// Shutdown our relay on exit.
	defer shutdownRelay(relay, relaySync)

	// Start running each leg.
	for {
		channel.runEventRelayCycle(relay, relaySync, relayLeg)
		if relaySync.IsDone() {
			return
		}
		relayLeg++
	}
}

// setupAndLaunchEventRelay sets up a new relay and launches a goroutine to run it
func (channel *Channel) setupAndLaunchEventRelay(relay eventRelay) error {
	// Launch the runner
	thisSync := newRelaySync(channel.relaySync.shared)
	go channel.runEventRelay(relay, thisSync)
	err := thisSync.WaitForInit(channel.ctx)
	if err != nil {
		return err
	}

	// Return
	return nil
}
