package amqp

import (
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
	"sync/atomic"
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
	SetupForRelayLeg(newChannel *streadway.Channel, settings channelSettings) error
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
			setupErr = relay.SetupForRelayLeg(
				channel.transportChannel.Channel, channel.transportChannel.settings,
			)
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
		err := relay.SetupForRelayLeg(
			channel.transportChannel.Channel, channel.transportChannel.settings,
		)
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

// Holds args for consume operation
type consumeArgs struct {
	queue, consumer string
	autoAck, exclusive, noLocal, noWait bool
	args Table
	callerDeliveryChan chan Delivery
}

// Relays Deliveries across channel disconnects.
type consumeRelay struct {
	// Arguments to call on Consume
	ConsumeArgs consumeArgs
	// Delivery channel to pass deliveries back to the client.
	CallerDeliveries chan <- Delivery

	// The pointer to the delivery tag count we need to atomically update with each
	// consume.
	deliveryTagCount *uint64
	// The delivery tag offset to add to our deliveries.
	deliveryTagOffset uint64
	// The current delivery channel coming from the broker.
	brokerDeliveries <- chan streadway.Delivery
}

func (relay *consumeRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	return parent.With().
		Str("EVENT_TYPE", "CONSUME").
		Str("CONSUMER_QUEUE", relay.ConsumeArgs.queue).
		Logger()
}

func (relay *consumeRelay) SetupForRelayLeg(
	newChannel *streadway.Channel, settings channelSettings,
) error {
	brokerDeliveries, err := newChannel.Consume(
		relay.ConsumeArgs.queue,
		relay.ConsumeArgs.consumer,
		relay.ConsumeArgs.autoAck,
		relay.ConsumeArgs.exclusive,
		relay.ConsumeArgs.noLocal,
		relay.ConsumeArgs.noWait,
		relay.ConsumeArgs.args,
	)

	if err != nil {
		return err
	}

	relay.brokerDeliveries = brokerDeliveries
	relay.deliveryTagCount = settings.tagConsumeCount
	relay.deliveryTagOffset = *settings.tagConsumeOffset

	return nil
}

func (relay *consumeRelay) RunRelayLeg() (done bool, err error) {
	// Drain consumer events
	for brokerDelivery := range relay.brokerDeliveries {
		atomic.AddUint64(relay.deliveryTagCount, 1)
		brokerDelivery.DeliveryTag += relay.deliveryTagOffset

		clientDelivery := Delivery{
			Delivery:  brokerDelivery,
			tagOffset: relay.deliveryTagOffset,
		}

		relay.CallerDeliveries <- clientDelivery
	}

	return false, nil
}

func (relay *consumeRelay) Shutdown() error {
	close(relay.CallerDeliveries)
	return nil
}

// Relays Deliveries across channel disconnects.
type notifyPublishRelay struct {
	// Delivery channel to pass deliveries back to the client.
	CallerConfirmations chan <- Confirmation

	// The number of confirmations we have sent on this relay.
	confirmsSent uint64

	// The delivery tag offset to add to our confirmations.
	deliveryTagOffset uint64
	// The current delivery channel coming from the broker.
	brokerConfirmations <- chan streadway.Confirmation

	// Logger
	logger zerolog.Logger
}

func (relay *notifyPublishRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_PUBLISH").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyPublishRelay) SetupForRelayLeg(
	newChannel *streadway.Channel, settings channelSettings,
) error {
	// Get our broker channel. We will make it with the same capacity of the channel the
	// caller sent into it.
	brokerConfirmations := make(
		chan streadway.Confirmation, cap(relay.CallerConfirmations),
	)
	relay.brokerConfirmations = newChannel.NotifyPublish(brokerConfirmations)

	// The offset is the number of messages our last channel published.
	relay.deliveryTagOffset = *settings.tagPublishCount

	return nil
}

func (relay *notifyPublishRelay) logConfirmation(confirmation Confirmation) {
	relay.logger.Debug().
		Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
		Bool("ACK", confirmation.Ack).
		Bool("ORPHAN", confirmation.DisconnectOrphan).
		Msg("publish confirmation event sent")
}

func (relay *notifyPublishRelay) RunRelayLeg() (done bool, err error) {
	// The goal of this library is to simulate the behavior of streadway/amqp. Since
	// the streadway lib guarantees that all confirms will be in an ascending, ordered,
	// unbroken stream, we need to handle a case where a channel was terminated before
	// all deliveries were acknowledged, and continuing to send confirmations would
	// result in a DeliveryTag gap.
	//
	// It's possible that when the last connection went down, we missed some
	// confirmations. We are going to check that the offset matches the number we
	// have sent so far and, if not, nack the difference. We are only going to do this
	// on re-connections to better mock the behavior of the original lib, where if the
	// channel is forcibly closed, the final messages will not be confirmed.
	for relay.confirmsSent < relay.deliveryTagOffset {
		confirmation := Confirmation{
			Confirmation:     streadway.Confirmation{
				DeliveryTag: relay.confirmsSent + 1,
				Ack:         false,
			},
			DisconnectOrphan: true,
		}
		if relay.logger.Debug().Enabled() {
			relay.logConfirmation(confirmation)
		}
		relay.CallerConfirmations <- confirmation
		relay.confirmsSent++
	}

	// Range over the confirmations from the broker.
	for brokerConf := range relay.brokerConfirmations  {
		brokerConf.DeliveryTag += relay.deliveryTagOffset
		// Apply the offset to the delivery tag.
		confirmation := Confirmation{
			Confirmation:     brokerConf,
			DisconnectOrphan: false,
		}
		if relay.logger.Debug().Enabled() {
			relay.logConfirmation(confirmation)
		}
		relay.CallerConfirmations <- confirmation
		relay.confirmsSent++
	}

	// Otherwise continue to the next channel.
	return false, nil
}

func (relay *notifyPublishRelay) Shutdown() error {
	close(relay.CallerConfirmations)
	return nil
}

// Relays return notification to the cl
type notifyReturnRelay struct {
	// The channel we are relaying returns to from the broker
	CallerReturns chan <- Return

	// The current broker channel we are pulling from.
	brokerReturns <- chan Return

	// Logger
	logger zerolog.Logger
}

func (relay *notifyReturnRelay) Logger(parent zerolog.Logger) zerolog.Logger {
	logger := parent.With().Str("EVENT_TYPE", "NOTIFY_RETURN").Logger()
	relay.logger = logger
	return relay.logger
}

func (relay *notifyReturnRelay) SetupForRelayLeg(
	newChannel *streadway.Channel, settings channelSettings,
) error {
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
