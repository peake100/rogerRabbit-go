package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Interface that an event relay should implement to handle continuous relaying of
// events from the underlying channels to the client without interruption. The
// boilerplate of handling all the synchronization locks will be handled for any
// relay passed to Channel.setupAndLaunchEventRelay()
type eventRelay interface {
	// Setup logger for event relay.
	Logger(parent zerolog.Logger) zerolog.Logger
	// Use a streadway.Channel() to set up relaying messages from this channel to the
	// client.
	SetupForRelayLeg(newChannel *streadway.Channel) error
	// Run the relay until all events from the channel passed to SetupRelayLeg() are
	// exhausted,
	RunRelayLeg() (done bool, err error)
	// shutdown the relay. This will usually involve closing the client-facing channel.
	Shutdown() error
}

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

func (channel *Channel) runEventRelayCycleSetup(
	relay eventRelay, relaySync *relaySync, legLogger zerolog.Logger,
) (setupErr error) {
	// Release our hold on the event relays setup complete on exit.
	defer relaySync.SetupComplete()

	// Wait for a new channel to be established.
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("relay waiting for new connection")
	}
	relaySync.WaitForSetup()

	if relaySync.IsDone() {
		return nil
	}

	// Set up our next leg.
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("setting up relay leg")
	}
	setupErr = relay.SetupForRelayLeg(channel.transportChannel.Channel)
	if setupErr != nil {
		legLogger.Err(setupErr).Msg("error setting up event relay leg")
	}

	return setupErr
}

func (channel *Channel) runEventRelayCycleLeg(
	relay eventRelay, flowControl *relaySync, legLogger zerolog.Logger,
) {
	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("starting relay leg")
	}
	done, runErr := relay.RunRelayLeg()
	flowControl.SetDone(done)

	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("exiting relay leg")
	}
	// Log any errors.
	if runErr != nil {
		legLogger.Err(runErr).Msg("error running event relay leg")
	}
}

func (channel *Channel) runEventRelayCycle(
	relay eventRelay, relaySync *relaySync, legLogger zerolog.Logger,
) {
	setupErr := channel.runEventRelayCycleSetup(relay, relaySync, legLogger)

	if legLogger.Debug().Enabled() {
		legLogger.Debug().Msg("waiting for relay start")
	}

	if !relaySync.IsDone() {
		relaySync.WaitForRunLeg()
	}

	// Whether or not we run the leg, reset our sync to mark the run as complete.
	defer relaySync.RunLegComplete()

	// If there was no setup error and our relay is not done, run the leg.
	if setupErr == nil {
		channel.runEventRelayCycleLeg(relay, relaySync, legLogger)
	}
}

// launch as goroutine to run an event relay after it's initial setup.
func (channel *Channel) runEventRelay(relay eventRelay, relayLogger zerolog.Logger) {
	relaySync := &relaySync{
		done:              false,
		shared:            channel.transportChannel.relaySync.shared,
		firstSetupDone:    false,
		setupCompleteHeld: false,
		legCompleteHeld:   false,
	}

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

func (channel *Channel) setupAndLaunchEventRelay(relay eventRelay) error {
	logger := relay.Logger(channel.logger).
		With().
		Str("SUBPROCESS", "EVENT_RELAY").
		Logger()

	// Launch the runner
	go channel.runEventRelay(relay, logger)

	// Return
	return nil
}
