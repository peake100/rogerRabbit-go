package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// eventRelay is a common interface for relaying events from the underlying channels to
// the client without interruption. The boilerplate of handling all the synchronization
// locks will be handled for any relay passed to Channel.setupAndLaunchEventRelay()
type eventRelay interface {
	// Logger sets up logger for event relay.
	Logger(parent zerolog.Logger) zerolog.Logger

	// SetupForRelayLeg is called once when the relay is first instantiated, and each
	// time there is a reconnection of the underlying channel. The raw
	// streadway/amqp.Channel is passed in for any alterations / inspections that need
	// to be made for the next leg.
	SetupForRelayLeg(newChannel *streadway.Channel) error

	// RunRelayLeg runs the relay until all events from the streadway/amqp.Channel
	// passed to SetupForRelayLeg() are exhausted,
	RunRelayLeg() (done bool, err error)

	// Shutdown is called to exit the relay. This will usually involve closing the
	// client-facing channel.
	Shutdown() error
}

// shutdownRelay handles all the boilerplate of calling eventRelay.Shutdown.
func shutdownRelay(relay eventRelay, relaySync *relaySync, relayLogger zerolog.Logger) {
	// Release any outstanding WaitGroups we are holding on exit
	defer relaySync.Complete()

	if relayLogger.Debug().Enabled() {
		relayLogger.Debug().Msg("shutting down relay")
	}

	// Invoke the shutdown method of the relay.
	err := relay.Shutdown()
	if err != nil {
		relayLogger.Err(err).Msg("error shutting down relay")
	}
}

// runEventRelayCycleSetup handles the boilerplate of setting up a relay for the next
// leg.
func (channel *Channel) runEventRelayCycleSetup(
	relay eventRelay,
	relaySync *relaySync,
	legLogger zerolog.Logger,
) (setupErr error) {
	// Release our hold on the event relays setup complete on exit.
	defer relaySync.SetupComplete()

	// Wait for a new channel to be established.
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("relay waiting for new connection")
	}
	relaySync.WaitToSetup()

	// If the relay is done, return rather than setting up for the next leg.
	if relaySync.IsDone() {
		return nil
	}

	// Set up our next leg.
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("setting up relay leg")
	}
	setupErr = relay.SetupForRelayLeg(channel.transportChannel.BasicChannel)
	if setupErr != nil {
		legLogger.Err(setupErr).Msg("error setting up event relay leg")
	}

	return setupErr
}

// runEventRelayCycleLeg handles the boilerplate of running an event relay leg (relaying
// all events from a single underlying streadway/ampq.Channel)
func (channel *Channel) runEventRelayCycleLeg(
	relay eventRelay, relaySync *relaySync, legLogger zerolog.Logger,
) {
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("starting relay leg")
	}
	done, runErr := relay.RunRelayLeg()
	relaySync.SetDone(done)

	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("exiting relay leg")
	}
	// Log any errors.
	if runErr != nil {
		legLogger.Err(runErr).Msg("error running event relay leg")
	}
}

// runEventRelayCycle runs a single, full cycle of setting up and running a relay leg.
func (channel *Channel) runEventRelayCycle(
	relay eventRelay, relaySync *relaySync, legLogger zerolog.Logger,
) {
	setupErr := channel.runEventRelayCycleSetup(relay, relaySync, legLogger)

	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("waiting for relay start")
	}

	if !relaySync.IsDone() {
		relaySync.WaitToRunLeg()
	}

	// Whether or not we run the leg, reset our sync to mark the run as complete.
	defer relaySync.RunLegComplete()

	// If there was no setup error and our relay is not done, run the leg.
	if setupErr == nil {
		channel.runEventRelayCycleLeg(relay, relaySync, legLogger)
	}
}

// runEventRelay should be launched as goroutine to run an event relay after it's
// initial setup.
func (channel *Channel) runEventRelay(
	relay eventRelay, relaySync *relaySync, relayLogger zerolog.Logger,
) {
	relayLeg := 1
	// Shutdown our relay on exit.
	defer shutdownRelay(relay, relaySync, relayLogger)

	// Start running each leg.
	for {
		legLogger := relayLogger.With().Int("LEG", relayLeg).Logger()
		channel.runEventRelayCycle(relay, relaySync, legLogger)
		if relaySync.IsDone() {
			return
		}
		relayLeg++
	}
}

// setupAndLaunchEventRelay sets up a new relay and launches a goroutine to run it
func (channel *Channel) setupAndLaunchEventRelay(relay eventRelay) error {
	logger := relay.Logger(channel.logger).
		With().
		Str("SUBPROCESS", "EVENT_RELAY").
		Logger()

	// Launch the runner
	thisSync := newRelaySync(channel.transportChannel.relaySync.shared)
	go channel.runEventRelay(relay, thisSync, logger)
	err := thisSync.WaitForInit(channel.ctx)
	if err != nil {
		return err
	}

	// Return
	return nil
}
