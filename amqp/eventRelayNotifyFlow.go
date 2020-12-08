package amqp

import (
	"context"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

type notifyFlowRelay struct {
	// Context of the current channel
	ChannelCtx context.Context

	// The channel we are relaying returns to from the broker
	CallerFlow chan<- bool

	// The current broker channel we are pulling from.
	brokerFlow <-chan bool

	// Whether this relay has been setup before.
	setup bool
	// The last notification from the broker.
	lastNotification bool

	// Logger
	logger zerolog.Logger
}

func (relay *notifyFlowRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_CANCEL").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyFlowRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	// Check if this is our initial setup
	if relay.setup {
		// If we have already setup the relay once, that means we are opening a new
		// channel, and should send a flow -> true to the caller as a fresh channel
		// will not have flow turned off yet.
		relay.CallerFlow <- true
	} else {
		relay.setup = true
	}

	// Set the last notification to true.
	relay.lastNotification = true

	brokerChannel := make(chan bool, cap(relay.CallerFlow))
	relay.brokerFlow = brokerChannel
	newChannel.NotifyFlow(brokerChannel)

	return nil
}

func (relay *notifyFlowRelay) RunRelayLeg() (done bool, err error) {
	for thisFlow := range relay.brokerFlow {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Bool("FLOW", thisFlow).
				Msg("cancel notification sent")
		}

		relay.CallerFlow <- thisFlow
		relay.lastNotification = thisFlow
	}

	// Turn flow to false on broker disconnection if the roger channel has not been
	// closed and the last notification sent was a ``true`` (we don't want to send two
	// falses in a row).
	if relay.ChannelCtx.Err() == nil && relay.lastNotification {
		relay.CallerFlow <- false
		relay.lastNotification = false
	}

	return false, nil
}

func (relay *notifyFlowRelay) Shutdown() error {
	defer close(relay.CallerFlow)
	return nil
}
