package amqp

import (
	"context"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// notifyFlowRelay implements eventRelay for Channel.NotifyFlow.
type notifyFlowRelay struct {
	// ChannelCtx is the context of the current channel
	ChannelCtx context.Context

	// CallerFlow is the channel we are relaying returns to from the broker
	CallerFlow chan<- bool

	// brokerFlow is the current broker channel we are pulling from.
	brokerFlow <-chan bool

	// setup is whether this relay has been setup before.
	setup bool

	// lastEvent is the value of the last event sent from the broker.
	lastEvent bool

	// logger is the Logger for this relay.
	logger zerolog.Logger
}

// Logger implements eventRelay and sets up our logger.
func (relay *notifyFlowRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_CANCEL").Logger()
	relay.logger = logger
	return relay.logger
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyFlow.
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
	relay.lastEvent = true

	brokerChannel := make(chan bool, cap(relay.CallerFlow))
	relay.brokerFlow = brokerChannel
	newChannel.NotifyFlow(brokerChannel)

	return nil
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyFlow
// events to the original caller.
func (relay *notifyFlowRelay) RunRelayLeg() (done bool, err error) {
	for thisFlow := range relay.brokerFlow {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Bool("FLOW", thisFlow).
				Msg("cancel notification sent")
		}

		relay.CallerFlow <- thisFlow
		relay.lastEvent = thisFlow
	}

	// Turn flow to false on broker disconnection if the roger channel has not been
	// closed and the last notification sent was a ``true`` (we don't want to send two
	// falses in a row).
	if relay.ChannelCtx.Err() == nil && relay.lastEvent {
		relay.CallerFlow <- false
		relay.lastEvent = false
	}

	return false, nil
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyFlowRelay) Shutdown() error {
	defer close(relay.CallerFlow)
	return nil
}
