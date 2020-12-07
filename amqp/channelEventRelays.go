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

// launch as goroutine to run an event relay after it's initial setup.
func (channel *Channel) runEventRelay(relay eventRelay, relayLogger zerolog.Logger) {
	var setupErr error

	relayLeg := 1
	legLogger := relayLogger.With().Int("LEG", relayLeg).Logger()

	// Shutdown our relay on exit.
	defer func() {
		if relayLogger.Debug().Enabled() {
			relayLogger.Debug().Msg("shutting down relay")
		}
		err := relay.Shutdown()
		if err != nil {
			relayLogger.Err(err).Msg("error shutting down relay")
		}
	}()

	// Start running each leg.
	for {
		var done bool

		// We're going to use closure here to control the flow of the WaitGroups.
		func() {
			// Release our hold on the running WaitGroup.
			defer channel.transportChannel.eventRelaysRunning.Done()
			// Grab a hold on the relays ready WaitGroup.
			defer channel.transportChannel.eventRelaysSetupComplete.Add(1)

			// If there was no error the last time we ran the setup for this relay,
			// run the leg.
			if setupErr != nil {
				return
			}

			var runErr error
			if legLogger.Debug().Enabled() {
				legLogger.Debug().Msg("starting relay leg")
			}
			done, runErr = relay.RunRelayLeg()
			if legLogger.Debug().Enabled() {
				legLogger.Debug().Msg("exiting relay leg")
			}
			// Log any errors.
			if runErr != nil {
				legLogger.Err(runErr).Msg("error running event relay leg")
			}
		}()

		relayLeg++
		legLogger = relayLogger.With().Int("LEG", relayLeg).Logger()

		func() {
			// Release our hold on the event relays ready on exit.
			defer channel.transportChannel.eventRelaysSetupComplete.Done()

			// Exit if done.
			if done {
				return
			}

			// Wait for a new channel to be established.
			if legLogger.Debug().Enabled() {
				legLogger.Debug().Msg("relay waiting for new connection")
			}
			channel.transportChannel.eventRelaysRunSetup.Wait()

			// If the new channel WaitGroup was released due to a context cancellation,
			// exit.
			if channel.ctx.Err() != nil {
				done = true
				return
			}

			// Set up our next leg.
			if legLogger.Debug().Enabled() {
				legLogger.Debug().Msg("setting up relay leg")
			}
			setupErr = relay.SetupForRelayLeg(channel.transportChannel.Channel)
			if setupErr != nil {
				legLogger.Err(setupErr).Msg("error setting up event relay leg")
			}

			// Grab the event processor running WaitGroup.
			channel.transportChannel.eventRelaysRunning.Add(1)
		}()

		// Stop loop if done or if our channel context has been cancelled.
		if done || channel.ctx.Err() != nil {
			break
		}

		// Wait for the final go-ahead to start our relay again
		channel.transportChannel.eventRelaysGo.Wait()
	}
}

func (channel *Channel) setupAndLaunchEventRelay(relay eventRelay) error {
	// Run the initial setup as an op so we can use the transport lock to safely enter
	// into our loop.
	var waitGroupGrabbed bool

	logger := relay.Logger(channel.logger).
		With().
		Str("SUBPROCESS", "EVENT_RELAY").
		Logger()

	op := func() error {
		err := relay.SetupForRelayLeg(channel.transportChannel.Channel)
		if err != nil {
			return err
		}
		// Grab a spot on the event processor WaitGroup
		waitGroupGrabbed = true
		channel.transportChannel.eventRelaysRunning.Add(1)
		return nil
	}

	// Run the initial setup, exit if we hit an error.
	if logger.Debug().Enabled() {
		logger.Debug().Msg("setting up initial relay leg")
	}
	setupErr := channel.retryOperationOnClosed(channel.ctx, op, true)
	if setupErr != nil {
		// Release the running WaitGroup if we grabbed it during setup.
		// Run the initial setup, exit if we hit an error.
		if waitGroupGrabbed {
			channel.transportChannel.eventRelaysRunning.Done()
		}
		return setupErr
	}

	// Launch the runner
	go channel.runEventRelay(relay, logger)

	// Return
	return nil
}
