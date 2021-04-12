package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	"github.com/rs/zerolog"
	streadway "github.com/streadway/amqp"
)

// middlewareBaseBuilder builds the base method handlers for a given robust
// connection + channel
type middlewareBaseBuilder struct {
	connection     *Connection
	channel        *Channel
	underlyingChan *streadway.Channel
}

// createBaseHandlerReconnect returns the base handler invoked on a Channel
// reconnection.
func (builder *middlewareBaseBuilder) createBaseHandlerReconnect() (
	handler amqpmiddleware.HandlerReconnect,
) {
	handler = func(
		ctx context.Context, attempt uint64, logger zerolog.Logger,
	) (*streadway.Channel, error) {
		channel, err := builder.connection.getStreadwayChannel(ctx)
		if err != nil {
			return nil, err
		}
		builder.underlyingChan = channel
		return channel, nil
	}

	return handler
}

// createBaseHandlerQueueDeclare returns the base handler for Channel.QueueDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueDeclare() (
	handler amqpmiddleware.HandlerQueueDeclare,
) {
	handler = func(args amqpmiddleware.ArgsQueueDeclare) (Queue, error) {
		return builder.underlyingChan.QueueDeclare(
			args.Name,
			args.Durable,
			args.AutoDelete,
			args.Exclusive,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerQueueDeclarePassive returns the base handler for
// Channel.QueueDeclarePassive that invokes the method of the underlying
// streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueDeclarePassive() (
	handler amqpmiddleware.HandlerQueueDeclare,
) {
	handler = func(args amqpmiddleware.ArgsQueueDeclare) (Queue, error) {
		return builder.underlyingChan.QueueDeclarePassive(
			args.Name,
			args.Durable,
			args.AutoDelete,
			args.Exclusive,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerQueueInspect returns the base handler for Channel.QueueInspect that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueInspect() (
	handler amqpmiddleware.HandlerQueueInspect,
) {
	handler = func(args amqpmiddleware.ArgsQueueInspect) (Queue, error) {
		return builder.underlyingChan.QueueInspect(args.Name)
	}

	return handler
}

// createBaseHandlerQueueDelete returns the base handler for Channel.QueueDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueDelete() (
	handler amqpmiddleware.HandlerQueueDelete,
) {
	handler = func(args amqpmiddleware.ArgsQueueDelete) (int, error) {
		return builder.underlyingChan.QueueDelete(
			args.Name,
			args.IfUnused,
			args.IfEmpty,
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerQueueBind returns the base handler for Channel.QueueBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueBind() (
	handler amqpmiddleware.HandlerQueueBind,
) {
	handler = func(args amqpmiddleware.ArgsQueueBind) error {
		return builder.underlyingChan.QueueBind(
			args.Name,
			args.Key,
			args.Exchange,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerQueueUnbind returns the base handler for Channel.QueueUnbind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueueUnbind() (
	handler amqpmiddleware.HandlerQueueUnbind,
) {
	handler = func(args amqpmiddleware.ArgsQueueUnbind) error {
		return builder.underlyingChan.QueueUnbind(
			args.Name,
			args.Key,
			args.Exchange,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerQueueUnbind returns the base handler for Channel.QueuePurge
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQueuePurge() (
	handler amqpmiddleware.HandlerQueuePurge,
) {
	handler = func(args amqpmiddleware.ArgsQueuePurge) (int, error) {
		return builder.underlyingChan.QueuePurge(
			args.Name,
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerExchangeDeclare returns the base handler for Channel.ExchangeDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDeclare() (
	handler amqpmiddleware.HandlerExchangeDeclare,
) {
	handler = func(args amqpmiddleware.ArgsExchangeDeclare) error {
		return builder.underlyingChan.ExchangeDeclare(
			args.Name,
			args.Kind,
			args.Durable,
			args.AutoDelete,
			args.Internal,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerExchangeDeclarePassive returns the base handler for
// Channel.ExchangeDeclarePassive that invokes the method of the underlying
// streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDeclarePassive() (
	handler amqpmiddleware.HandlerExchangeDeclare,
) {
	handler = func(args amqpmiddleware.ArgsExchangeDeclare) error {
		return builder.underlyingChan.ExchangeDeclarePassive(
			args.Name,
			args.Kind,
			args.Durable,
			args.AutoDelete,
			args.Internal,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerExchangeDelete returns the base handler for Channel.ExchangeDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeDelete() (
	handler amqpmiddleware.HandlerExchangeDelete,
) {
	handler = func(args amqpmiddleware.ArgsExchangeDelete) error {
		return builder.underlyingChan.ExchangeDelete(
			args.Name,
			args.IfUnused,
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerExchangeBind returns the base handler for Channel.ExchangeBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeBind() (
	handler amqpmiddleware.HandlerExchangeBind,
) {
	handler = func(args amqpmiddleware.ArgsExchangeBind) error {
		return builder.underlyingChan.ExchangeBind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerExchangeUnbind returns the base handler for Channel.ExchangeUnbind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerExchangeUnbind() (
	handler amqpmiddleware.HandlerExchangeUnbind,
) {
	handler = func(args amqpmiddleware.ArgsExchangeUnbind) error {
		return builder.underlyingChan.ExchangeUnbind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	return handler
}

// createBaseHandlerQoS returns the base handler for Channel.Qos that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerQoS() (
	handler amqpmiddleware.HandlerQoS,
) {
	handler = func(args amqpmiddleware.ArgsQoS) error {
		return builder.underlyingChan.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
	}

	return handler
}

// createBaseHandlerFlow returns the base handler for Channel.Flow that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerFlow() (
	handler amqpmiddleware.HandlerFlow,
) {
	handler = func(args amqpmiddleware.ArgsFlow) error {
		return builder.underlyingChan.Flow(args.Active)
	}

	return handler
}

// createBaseHandlerConfirm returns the base handler for Channel.Confirm that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerConfirm() (
	handler amqpmiddleware.HandlerConfirm,
) {
	handler = func(args amqpmiddleware.ArgsConfirms) error {
		return builder.underlyingChan.Confirm(
			args.NoWait,
		)
	}

	return handler
}

// createBaseHandlerPublish returns the base handler for Channel.Publish that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerPublish() (
	handler amqpmiddleware.HandlerPublish,
) {
	handler = func(args amqpmiddleware.ArgsPublish) error {
		return builder.underlyingChan.Publish(
			args.Exchange,
			args.Key,
			args.Mandatory,
			args.Immediate,
			args.Msg,
		)
	}

	return handler
}

// createBaseHandlerGet returns the base handler for Channel.Get that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerGet() (
	handler amqpmiddleware.HandlerGet,
) {
	handler = func(args amqpmiddleware.ArgsGet) (msg Delivery, ok bool, err error) {
		var msgOrig streadway.Delivery
		msgOrig, ok, err = builder.underlyingChan.Get(args.Queue, args.AutoAck)
		msg = datamodels.NewDelivery(msgOrig, builder.channel)
		return msg, ok, err
	}

	return handler
}

// createBaseHandlerConsume returns the base handler for Channel.Get that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerConsume() (
	handler amqpmiddleware.HandlerConsume,
) {
	handler = func(
		args amqpmiddleware.ArgsConsume,
	) (deliveryChan <-chan datamodels.Delivery, err error) {
		channel := builder.channel

		callArgs := &consumeArgs{
			queue:     args.Queue,
			consumer:  args.Consumer,
			autoAck:   args.AutoAck,
			exclusive: args.Exclusive,
			noLocal:   args.NoLocal,
			noWait:    args.NoWait,
			args:      args.Args,
			// Make a buffered channel so we don't cause latency from waiting for queues
			// to be ready
			callerDeliveryChan: make(chan datamodels.Delivery, 16),
		}
		deliveryChan = callArgs.callerDeliveryChan

		// Create our consumer relay
		relay := newConsumeRelay(
			callArgs,
			channel,
			channel.transportChannel.handlers.consumeEvent,
		)

		// Pass it to our relay handler.
		err = channel.setupAndLaunchEventRelay(relay)
		if err != nil {
			return nil, err
		}

		// If no error, pass the channel back to the caller
		return deliveryChan, nil
	}

	return handler
}

// createBaseHandlerAck returns the base handler for Channel.Ack that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerAck() (
	handler amqpmiddleware.HandlerAck,
) {
	handler = func(args amqpmiddleware.ArgsAck) error {
		return builder.underlyingChan.Ack(args.Tag, args.Multiple)
	}

	return handler
}

// createBaseHandlerNack returns the base handler for Channel.Nack that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNack() (
	handler amqpmiddleware.HandlerNack,
) {
	handler = func(args amqpmiddleware.ArgsNack) error {
		return builder.underlyingChan.Nack(args.Tag, args.Multiple, args.Requeue)
	}

	return handler
}

// createBaseHandlerReject returns the base handler for Channel.Reject that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerReject() (
	handler amqpmiddleware.HandlerReject,
) {
	handler = func(args amqpmiddleware.ArgsReject) error {
		return builder.underlyingChan.Reject(args.Tag, args.Requeue)
	}

	return handler
}

// createBaseHandlerNotifyPublish returns the base handler for Channel.NotifyPublish
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyPublish() (
	handler amqpmiddleware.HandlerNotifyPublish,
) {
	channel := builder.channel

	handler = func(args amqpmiddleware.ArgsNotifyPublish) chan Confirmation {
		relay := newNotifyPublishRelay(
			args.Confirm,
			channel.transportChannel.handlers.notifyPublishEvent,
		)

		err := channel.setupAndLaunchEventRelay(relay)
		// On an error, close the channel.
		if err != nil {
			channel.logger.Error().
				Err(err).
				Msg("error setting up NotifyPublish event relay")
			close(args.Confirm)
		}
		return args.Confirm
	}

	return handler
}

// createBaseHandlerNotifyConfirm returns the base handler for Channel.NotifyConfirm
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyConfirm() (
	handler amqpmiddleware.HandlerNotifyConfirm,
) {
	channel := builder.channel

	handler = func(args amqpmiddleware.ArgsNotifyConfirm) (chan uint64, chan uint64) {
		logger := channel.logger.With().
			Str("EVENT_TYPE", "NOTIFY_CONFIRM").
			Logger()

		// Set up the innermost event handler.
		var eventHandler amqpmiddleware.HandlerNotifyConfirmEvents = func(
			event amqpmiddleware.EventNotifyConfirm,
		) {
			notifyConfirmHandleAckAndNack(
				event.Confirmation, args.Ack, args.Nack, logger,
			)
		}

		// Wrap the event handler in the user-supplied middleware.
		middlewares := channel.transportChannel.handlers.notifyConfirmEvent
		for _, thisMiddleware := range middlewares {
			eventHandler = thisMiddleware(eventHandler)
		}

		// Run the event relay.
		go builder.runNotifyConfirm(args, eventHandler)

		return args.Ack, args.Nack
	}

	return handler
}

// runNotifyConfirm relay the NotifyConfirm events to the caller by calling
// NotifyPublish.
func (builder *middlewareBaseBuilder) runNotifyConfirm(
	args amqpmiddleware.ArgsNotifyConfirm,
	eventHandler amqpmiddleware.HandlerNotifyConfirmEvents,
) {
	defer notifyConfirmCloseConfirmChannels(args.Ack, args.Nack)

	channel := builder.channel
	confirmsEvents := channel.NotifyPublish(
		make(chan datamodels.Confirmation, cap(args.Ack)+cap(args.Nack)),
	)

	// range over confirmation events and place them in the ack and nack
	// channels.
	for confirmation := range confirmsEvents {
		// Create the middleware event.
		event := amqpmiddleware.EventNotifyConfirm{Confirmation: confirmation}

		// Pass it to the handler.
		eventHandler(event)
	}
}

// createBaseHandlerNotifyConfirmOrOrphaned returns the base handler for
// Channel.NotifyConfirmOrOrphaned.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyConfirmOrOrphaned() (
	handler amqpmiddleware.HandlerNotifyConfirmOrOrphaned,
) {
	channel := builder.channel

	handler = func(
		args amqpmiddleware.ArgsNotifyConfirmOrOrphaned,
	) (chan uint64, chan uint64, chan uint64) {
		ack, nack, orphaned := args.Ack, args.Nack, args.Orphaned

		confirmsEvents := channel.NotifyPublish(
			make(chan datamodels.Confirmation, cap(ack)+cap(nack)+cap(orphaned)),
		)

		eventHandler := builder.createEventHandlerNotifyConfirmOrOrphaned(args)

		go channel.runNotifyConfirmOrOrphaned(
			eventHandler, ack, nack, orphaned, confirmsEvents,
		)

		return ack, nack, orphaned
	}

	return handler
}

// createEventHandlerNotifyConfirmOrOrphaned creates an event handler for event on
// Channel.NotifyConfirmOrOrphaned using user-supplied middleware.
func (builder *middlewareBaseBuilder) createEventHandlerNotifyConfirmOrOrphaned(
	args amqpmiddleware.ArgsNotifyConfirmOrOrphaned,
) amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents {
	logger := builder.channel.logger.With().
		Str("EVENT_TYPE", "NOTIFY_CONFIRM_OR_ORPHAN").
		Logger()

	var eventHandler amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents = func(
		event amqpmiddleware.EventNotifyConfirmOrOrphaned,
	) {
		confirmation := event.Confirmation
		if confirmation.DisconnectOrphan {
			if logger.Debug().Enabled() {
				logger.Debug().
					Uint64("DELIVERY_TAG", confirmation.DeliveryTag).
					Bool("ACK", confirmation.Ack).
					Str("CHANNEL", "ORPHANED").
					Msg("orphaned confirmation sent")
			}
			args.Orphaned <- confirmation.DeliveryTag
		} else {
			notifyConfirmHandleAckAndNack(confirmation, args.Ack, args.Nack, logger)
		}
	}

	middlewares := builder.channel.transportChannel.handlers.
		notifyConfirmOrOrphanedEvent

	for _, thisMiddleware := range middlewares {
		eventHandler = thisMiddleware(eventHandler)
	}

	return eventHandler
}

// invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyReturn() (
	handler amqpmiddleware.HandlerNotifyReturn,
) {
	channel := builder.channel

	handler = func(args amqpmiddleware.ArgsNotifyReturn) chan Return {
		relay := newNotifyReturnRelay(
			args.Returns, channel.transportChannel.handlers.notifyReturnEvents,
		)

		err := channel.setupAndLaunchEventRelay(relay)
		if err != nil {
			close(args.Returns)
			channel.logger.Err(err).Msg("error setting up notify return relay")
		}
		return args.Returns
	}

	return handler
}

// createBaseHandlerNotifyCancel returns the base handler for Channel.NotifyCancel that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyCancel() (
	handler amqpmiddleware.HandlerNotifyCancel,
) {
	channel := builder.channel

	handler = func(args amqpmiddleware.ArgsNotifyCancel) chan string {
		relay := newNotifyCancelRelay(
			args.Cancellations, channel.transportChannel.handlers.notifyCancelEvents,
		)

		err := channel.setupAndLaunchEventRelay(relay)
		if err != nil {
			close(args.Cancellations)
			channel.logger.Err(err).Msg("error setting up notify cancel relay")
		}

		return args.Cancellations
	}

	return handler
}

// createBaseHandlerNotifyFlow returns the base handler for Channel.NotifyFlow that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder *middlewareBaseBuilder) createBaseHandlerNotifyFlow() (
	handler amqpmiddleware.HandlerNotifyFlow,
) {
	channel := builder.channel

	handler = func(args amqpmiddleware.ArgsNotifyFlow) chan bool {
		// Create a new event relay.
		relay := newNotifyFlowRelay(
			channel.ctx,
			args.FlowNotifications,
			channel.transportChannel.handlers.notifyFlowEvents,
		)

		// Setup and launch the relay.
		err := channel.setupAndLaunchEventRelay(relay)
		if err != nil {
			close(args.FlowNotifications)
			channel.logger.Err(err).Msg("error setting up notify cancel relay")
		}

		return args.FlowNotifications
	}

	return handler
}
