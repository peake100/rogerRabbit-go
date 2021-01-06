package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// notifyPublishRelay implements eventRelay for Channel.NotifyReturn.
type notifyReturnRelay struct {
	// The channel we are relaying returns to from the broker
	CallerReturns chan<- Return

	// The current broker channel we are pulling from.
	brokerReturns <-chan Return

	// Logger
	logger zerolog.Logger
}

// Logger implements eventRelay and sets up our logger.
func (relay *notifyReturnRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_RETURN").Logger()
	relay.logger = logger
	return relay.logger
}

// SetupForRelayLeg implements eventRelay, and sets up a new source event channel from
// streadway/amqp.Channel.NotifyReturn.
func (relay *notifyReturnRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerChannel := make(chan Return, cap(relay.CallerReturns))
	relay.brokerReturns = brokerChannel
	newChannel.NotifyReturn(brokerChannel)
	return nil
}

// RunRelayLeg implements eventRelay, and relays streadway/amqp.Channel.NotifyReturn
// events to the original caller.
func (relay *notifyReturnRelay) RunRelayLeg() (done bool, err error) {
	for thisReturn := range relay.brokerReturns {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Str("EXCHANGE", thisReturn.Exchange).
				Str("ROUTING_KEY", thisReturn.MessageId).
				Str("MESSAGE_ID", thisReturn.MessageId).
				Bytes("BODY", thisReturn.Body).
				Msg("return notification sent")
		}

		relay.CallerReturns <- thisReturn
	}

	return false, nil
}

// Shutdown implements eventRelay, and closes the caller-facing event channel.
func (relay *notifyReturnRelay) Shutdown() error {
	defer close(relay.CallerReturns)
	return nil
}
