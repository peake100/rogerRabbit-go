package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

type notifyCancelRelay struct {
	// The channel we are relaying returns to from the broker
	CallerCancellations chan<- string

	// The current broker channel we are pulling from.
	brokerCancellations <-chan string

	// Logger
	logger zerolog.Logger
}

func (relay *notifyCancelRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_CANCEL").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyCancelRelay) SetupForRelayLeg(newChannel *streadway.Channel) error {
	brokerChannel := make(chan string, cap(relay.CallerCancellations))
	relay.brokerCancellations = brokerChannel
	newChannel.NotifyCancel(brokerChannel)
	return nil
}

func (relay *notifyCancelRelay) RunRelayLeg() (done bool, err error) {
	for thisCancellation := range relay.brokerCancellations {
		if relay.logger.Debug().Enabled() {
			relay.logger.Debug().
				Str("CANCELLATION", thisCancellation).
				Msg("cancel notification sent")
		}

		relay.CallerCancellations <- thisCancellation
	}

	return false, nil
}

func (relay *notifyCancelRelay) Shutdown() error {
	defer close(relay.CallerCancellations)
	return nil
}
