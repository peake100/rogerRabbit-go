package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/pkg/amqp/amqpmiddleware"
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
// locks will be handled for any relay passed to Channel.eventRelaySetupAndLaunch()
type eventRelay interface {
	// SetupForRelayLeg runs the setup for a new relay leg.
	SetupForRelayLeg(newChannel *streadway.Channel) error

	// RunRelayLeg runs the relay until all events from the streadway/amqp.Channel
	// passed to SetupForRelayLeg() are exhausted,
	//
	// legNum is the leg number starting from 0 and incrementing each time this relay
	// is called.
	RunRelayLeg(legNum int) (done bool)

	// Shutdown is called to exit the relay. This will usually involve closing the
	// client-facing channel.
	Shutdown() error
}

// shutdownRelay handles all the boilerplate of calling eventRelay.Shutdown.
func shutdownRelay(relay eventRelay, relaySync relaySync) {
	// Release any outstanding WaitGroups we are holding on exit
	defer relaySync.SetDone()

	// Invoke the shutdown method of the relay.
	_ = relay.Shutdown()
}

// eventRelaySetupAndLaunch sets up a new relay and launches a goroutine to run it
func (channel *Channel) eventRelaySetupAndLaunch(relay eventRelay) {
	// Create a signal chan to tell us when the initial setup has been completed. This
	// means it is safe to return to the user as our listener channel has been
	// registered with the underlying channel. If we were to return immediately, the
	// user might start taking actions that SHOULD generate events before the even
	// channel is correctly registered, causing those events to go "missing".
	setupComplete := make(chan struct{})

	// Launch the relay.
	thisSync := newRelaySync(channel.ctx)
	go channel.runEventRelay(relay, thisSync, setupComplete)

	// Wait for the signal that our setup is complete.
	<-setupComplete
}

// runEventRelay should be launched as goroutine to run an event relay after it's
// initial setup.
func (channel *Channel) runEventRelay(relay eventRelay, relaySync relaySync, setupComplete chan struct{}) {
	// Shutdown our relay on exit.
	defer shutdownRelay(relay, relaySync)

	firstLegComplete := make(chan struct{})
	channel.eventRelayInitialSetup(relay, relaySync, setupComplete, firstLegComplete)

	// Wait for ou first leg to complete, then fall into a rhythm with the transport
	// manager
	<-firstLegComplete
	relayLeg := 1

	// Start running each leg.
	for {
		channel.runEventRelayCycle(relay, relaySync, relayLeg)
		if relaySync.IsDone() {
			return
		}
		relayLeg++
	}
}

// eventRelayInitialSetup does the initial setup of an event relay. We need to make sure
// that setup is complete before we return to the user, otherwise they may start taking
// actions that SHOULD trigger events before the event listener has been registered with
// the underlying streadway/amqp.Channel.
func (channel *Channel) eventRelayInitialSetup(
	relay eventRelay,
	relaySync relaySync,
	setupComplete chan struct{},
	firstLegComplete chan struct{},
) {
	// Signal this leg in an op so we can make sure we grab the right channel.
	_ = channel.transportManager.retryOperationOnClosed(
		channel.ctx,
		func(ctx context.Context) error {
			// Register the relay with the channel.
			channel.relaySync.AddRelay(relaySync.shared)

			// Run the fist leg with the current channel. We need to launch it in a
			// routine so we can signal leg complete (the manager needs to grab a write
			// lock to the transport before it checks the relays)
			go func(currentChannel *streadway.Channel) {
				defer close(firstLegComplete)
				defer relaySync.SignalLegComplete()

				// Run the relay setup then signal that initial setup is complete.
				var err error
				func() {
					defer close(setupComplete)
					err = relay.SetupForRelayLeg(currentChannel)
				}()
				if err != nil {
					return
				}

				// Run the first relay leg.
				if done := relay.RunRelayLeg(0); done {
					relaySync.SetDone()
				}
			}(channel.underlyingChannel)

			// Wait for the relay setup to complete before we return or release the read
			// lock to the user. Otherwise the user may think we are receiving events
			// and do something that creates them before we've actually set that up.
			<-setupComplete

			return nil
		},
		true,
	)
}

// runEventRelayCycle runs a single, full cycle of setting up and running a relay leg.
func (channel *Channel) runEventRelayCycle(
	relay eventRelay, relaySync relaySync, legNum int,
) {
	if relaySync.IsDone() {
		return
	}

	newChan := relaySync.WaitForNextLeg()
	if newChan == nil {
		return
	}

	// Whether or not we run the leg, reset our sync to mark the run as complete.
	defer relaySync.SignalLegComplete()

	err := relay.SetupForRelayLeg(newChan)
	relaySync.SignalSetupComplete()
	if err != nil {
		return
	}

	if done := relay.RunRelayLeg(legNum); done {
		relaySync.SetDone()
	}
}
