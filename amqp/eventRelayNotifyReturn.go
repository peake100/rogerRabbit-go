package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// Relays return notification to the cl
type notifyReturnRelay struct {
	// The channel we are relaying returns to from the broker
	CallerReturns chan<- Return

	// The current broker channel we are pulling from.
	brokerReturns <-chan Return

	// Logger
	logger zerolog.Logger
}

func (relay *notifyReturnRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_RETURN").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyReturnRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerChannel := make(chan Return, cap(relay.CallerReturns))
	relay.brokerReturns = brokerChannel
	newChannel.NotifyReturn(brokerChannel)
	return nil
}

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

func (relay *notifyReturnRelay) Shutdown() error {
	defer close(relay.CallerReturns)
	return nil
}
