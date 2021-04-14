package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
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

	// handler is the event handler for the relay wrapped in all middleware.
	handler amqpmiddleware.HandlerNotifyFlowEvents

	// logger is the Logger for this relay.
	logger zerolog.Logger
}

func (relay *notifyFlowRelay) baseHandler() amqpmiddleware.HandlerNotifyFlowEvents {
	return func(_ amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyFlow) {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Bool("FLOW", event.FlowNotification).
				Msg("cancel notification sent")
		}

		relay.CallerFlow <- event.FlowNotification
		relay.lastEvent = event.FlowNotification
	}
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
		relay.handler(
			createEventMetadata(-1, -1),
			amqpmiddleware.EventNotifyFlow{FlowNotification: true},
		)
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
func (relay *notifyFlowRelay) RunRelayLeg(legNum int) (done bool, err error) {
	eventNum := int64(0)
	for thisFlow := range relay.brokerFlow {
		relay.handler(
			createEventMetadata(legNum, eventNum),
			amqpmiddleware.EventNotifyFlow{FlowNotification: thisFlow},
		)
		eventNum++
	}

	// Turn flow to false on broker disconnection if the roger channel has not been
	// closed and the last notification sent was a ``true`` (we don't want to send two
	// false values in a row).
	if relay.ChannelCtx.Err() == nil && relay.lastEvent {
		relay.handler(
			createEventMetadata(-1, -1),
			amqpmiddleware.EventNotifyFlow{FlowNotification: false},
		)
	}

	return false, nil
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyFlowRelay) Shutdown() error {
	defer close(relay.CallerFlow)
	return nil
}

// newNotifyReturnRelay creates a new notifyReturnRelay.
func newNotifyFlowRelay(
	channelCtx context.Context,
	callerFlowNotifications chan<- bool,
	middleware []amqpmiddleware.NotifyFlowEvents,
) *notifyFlowRelay {
	// Create the relay
	relay := &notifyFlowRelay{
		ChannelCtx: channelCtx,
		CallerFlow: callerFlowNotifications,
	}

	// Apply all middleware to the handler
	relay.handler = relay.baseHandler()
	for _, thisMiddleware := range middleware {
		relay.handler = thisMiddleware(relay.handler)
	}

	return relay
}
