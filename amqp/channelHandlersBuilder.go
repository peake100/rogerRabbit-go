package amqp

import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/peake100/rogerRabbit-go/amqp/datamodels"
	streadway "github.com/streadway/amqp"
)

// channelHandlerBuilder builds the base method handlers for a given robust
// connection + channel
type channelHandlerBuilder struct {
	connection *Connection
	channel    *Channel

	middlewares ChannelMiddlewares
}

// createChannelReconnect returns the base handler invoked on a Channel
// reconnection.
func (builder channelHandlerBuilder) createChannelReconnect() amqpmiddleware.HandlerChannelReconnect {
	// capture connection into the closure.
	connection := builder.connection

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsChannelReconnect,
	) (results amqpmiddleware.ResultsChannelReconnect, err error) {
		results.Channel, err = connection.getStreadwayChannel(args.Ctx)
		if err != nil {
			return results, err
		}
		return results, nil
	}

	for _, middleware := range builder.middlewares.channelReconnect {
		handler = middleware(handler)
	}

	return handler
}

// createQueueDeclare returns the base handler for Channel.QueueDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueDeclare() amqpmiddleware.HandlerQueueDeclare {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsQueueDeclare,
	) (results amqpmiddleware.ResultsQueueDeclare, err error) {
		results.Queue, err = channel.underlyingChannel.QueueDeclare(
			args.Name,
			args.Durable,
			args.AutoDelete,
			args.Exclusive,
			args.NoWait,
			args.Args,
		)
		return results, err
	}

	for _, middleware := range builder.middlewares.queueDeclare {
		handler = middleware(handler)
	}

	return handler
}

// createQueueDeclarePassive returns the base handler for
// Channel.QueueDeclarePassive that invokes the method of the underlying
// streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueDeclarePassive() amqpmiddleware.HandlerQueueDeclare {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsQueueDeclare,
	) (results amqpmiddleware.ResultsQueueDeclare, err error) {
		results.Queue, err = channel.underlyingChannel.QueueDeclarePassive(
			args.Name,
			args.Durable,
			args.AutoDelete,
			args.Exclusive,
			args.NoWait,
			args.Args,
		)
		return results, err
	}

	for _, middleware := range builder.middlewares.queueDeclarePassive {
		handler = middleware(handler)
	}

	return handler
}

// createQueueInspect returns the base handler for Channel.QueueInspect that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueInspect() amqpmiddleware.HandlerQueueInspect {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsQueueInspect,
	) (results amqpmiddleware.ResultsQueueInspect, err error) {
		results.Queue, err = channel.underlyingChannel.QueueInspect(args.Name)
		return results, err
	}

	for _, middleware := range builder.middlewares.queueInspect {
		handler = middleware(handler)
	}

	return handler
}

// createQueueDelete returns the base handler for Channel.QueueDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueDelete() amqpmiddleware.HandlerQueueDelete {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsQueueDelete,
	) (results amqpmiddleware.ResultsQueueDelete, err error) {
		results.Count, err = channel.underlyingChannel.QueueDelete(
			args.Name,
			args.IfUnused,
			args.IfEmpty,
			args.NoWait,
		)
		return results, err
	}

	for _, middleware := range builder.middlewares.queueDelete {
		handler = middleware(handler)
	}

	return handler
}

// createQueueBind returns the base handler for Channel.QueueBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueBind() amqpmiddleware.HandlerQueueBind {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsQueueBind) error {
		return channel.underlyingChannel.QueueBind(
			args.Name,
			args.Key,
			args.Exchange,
			args.NoWait,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.queueBind {
		handler = middleware(handler)
	}

	return handler
}

// createQueueUnbind returns the base handler for Channel.QueueUnbind that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueueUnbind() amqpmiddleware.HandlerQueueUnbind {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsQueueUnbind) error {
		return channel.underlyingChannel.QueueUnbind(
			args.Name,
			args.Key,
			args.Exchange,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.queueUnbind {
		handler = middleware(handler)
	}

	return handler
}

// createQueueUnbind returns the base handler for Channel.QueuePurge
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQueuePurge() amqpmiddleware.HandlerQueuePurge {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsQueuePurge,
	) (results amqpmiddleware.ResultsQueuePurge, err error) {
		results.Count, err = channel.underlyingChannel.QueuePurge(
			args.Name,
			args.NoWait,
		)
		return results, err
	}

	for _, middleware := range builder.middlewares.queuePurge {
		handler = middleware(handler)
	}

	return handler
}

// createExchangeDeclare returns the base handler for Channel.ExchangeDeclare
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createExchangeDeclare() amqpmiddleware.HandlerExchangeDeclare {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsExchangeDeclare) error {
		return channel.underlyingChannel.ExchangeDeclare(
			args.Name,
			args.Kind,
			args.Durable,
			args.AutoDelete,
			args.Internal,
			args.NoWait,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.exchangeDeclare {
		handler = middleware(handler)
	}

	return handler
}

// createExchangeDeclarePassive returns the base handler for
// Channel.ExchangeDeclarePassive that invokes the method of the underlying
// streadway/amqp.Channel.
func (builder channelHandlerBuilder) createExchangeDeclarePassive() amqpmiddleware.HandlerExchangeDeclare {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsExchangeDeclare) error {
		return channel.underlyingChannel.ExchangeDeclarePassive(
			args.Name,
			args.Kind,
			args.Durable,
			args.AutoDelete,
			args.Internal,
			args.NoWait,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.exchangeDeclarePassive {
		handler = middleware(handler)
	}

	return handler
}

// createExchangeDelete returns the base handler for Channel.ExchangeDelete
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createExchangeDelete() amqpmiddleware.HandlerExchangeDelete {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsExchangeDelete) error {
		return channel.underlyingChannel.ExchangeDelete(
			args.Name,
			args.IfUnused,
			args.NoWait,
		)
	}

	for _, middleware := range builder.middlewares.exchangeDelete {
		handler = middleware(handler)
	}

	return handler
}

// createExchangeBind returns the base handler for Channel.ExchangeBind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createExchangeBind() amqpmiddleware.HandlerExchangeBind {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsExchangeBind) error {
		return channel.underlyingChannel.ExchangeBind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.exchangeBind {
		handler = middleware(handler)
	}

	return handler
}

// createExchangeUnbind returns the base handler for Channel.ExchangeUnbind
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createExchangeUnbind() amqpmiddleware.HandlerExchangeUnbind {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsExchangeUnbind) error {
		return channel.underlyingChannel.ExchangeUnbind(
			args.Destination,
			args.Key,
			args.Source,
			args.NoWait,
			args.Args,
		)
	}

	for _, middleware := range builder.middlewares.exchangeUnbind {
		handler = middleware(handler)
	}

	return handler
}

// createQoS returns the base handler for Channel.Qos that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createQoS() amqpmiddleware.HandlerQoS {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsQoS) error {
		return channel.underlyingChannel.Qos(
			args.PrefetchCount,
			args.PrefetchSize,
			args.Global,
		)
	}

	for _, middleware := range builder.middlewares.qos {
		handler = middleware(handler)
	}

	return handler
}

// createFlow returns the base handler for Channel.Flow that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createFlow() amqpmiddleware.HandlerFlow {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsFlow) error {
		return channel.underlyingChannel.Flow(args.Active)
	}

	for _, middleware := range builder.middlewares.flow {
		handler = middleware(handler)
	}

	return handler
}

// createConfirm returns the base handler for Channel.Confirm that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createConfirm() amqpmiddleware.HandlerConfirm {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsConfirms) error {
		return channel.underlyingChannel.Confirm(
			args.NoWait,
		)
	}

	for _, middleware := range builder.middlewares.confirm {
		handler = middleware(handler)
	}

	return handler
}

// createPublish returns the base handler for Channel.Publish that invokes
// the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createPublish() amqpmiddleware.HandlerPublish {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsPublish) error {
		return channel.underlyingChannel.Publish(
			args.Exchange,
			args.Key,
			args.Mandatory,
			args.Immediate,
			args.Msg,
		)
	}

	for _, middleware := range builder.middlewares.publish {
		handler = middleware(handler)
	}

	return handler
}

// createGet returns the base handler for Channel.Get that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createGet() amqpmiddleware.HandlerGet {
	// capture the channel into the closure
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsGet,
	) (results amqpmiddleware.ResultsGet, err error) {
		var msgOrig streadway.Delivery
		msgOrig, ok, err := channel.underlyingChannel.Get(
			args.Queue, args.AutoAck,
		)
		results.Msg = datamodels.NewDelivery(msgOrig, builder.channel)
		results.Ok = ok
		return results, err
	}

	for _, middleware := range builder.middlewares.get {
		handler = middleware(handler)
	}

	return handler
}

// createConsume returns the base handler for Channel.Get that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createConsume() amqpmiddleware.HandlerConsume {
	// capture the channel into the closure
	channel := builder.channel
	// Capture the event middleware in this closure.
	eventMiddleware := builder.middlewares.consumeEvents

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsConsume,
	) (results amqpmiddleware.ResultsConsume, err error) {
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

		// Create our consumer relay
		relay := newConsumeRelay(callArgs, channel, eventMiddleware)

		// Pass it to our relay handler.
		channel.eventRelaySetupAndLaunch(relay)

		results.DeliveryChan = callArgs.callerDeliveryChan
		// If no error, pass the channel back to the caller
		return results, nil
	}

	for _, middleware := range builder.middlewares.consume {
		handler = middleware(handler)
	}

	return handler
}

// createAck returns the base handler for Channel.Ack that invokes the method
// of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createAck() amqpmiddleware.HandlerAck {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsAck) error {
		return channel.underlyingChannel.Ack(args.Tag, args.Multiple)
	}

	for _, middleware := range builder.middlewares.ack {
		handler = middleware(handler)
	}

	return handler
}

// createNack returns the base handler for Channel.Nack that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNack() amqpmiddleware.HandlerNack {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNack) error {
		return channel.underlyingChannel.Nack(
			args.Tag, args.Multiple, args.Requeue,
		)
	}

	for _, middleware := range builder.middlewares.nack {
		handler = middleware(handler)
	}

	return handler
}

// createReject returns the base handler for Channel.Reject that invokes the
// method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createReject() amqpmiddleware.HandlerReject {
	// Capture the channel in the closure
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsReject) error {
		return channel.underlyingChannel.Reject(args.Tag, args.Requeue)
	}

	for _, middleware := range builder.middlewares.reject {
		handler = middleware(handler)
	}

	return handler
}

// createNotifyPublish returns the base handler for Channel.NotifyPublish
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNotifyPublish() amqpmiddleware.HandlerNotifyPublish {
	// Capture the event middleware in the closure.
	eventMiddleware := builder.middlewares.notifyPublishEvents
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsNotifyPublish,
	) amqpmiddleware.ResultsNotifyPublish {
		relay := newNotifyPublishRelay(args.Confirm, eventMiddleware)

		channel.eventRelaySetupAndLaunch(relay)

		return amqpmiddleware.ResultsNotifyPublish{Confirm: args.Confirm}
	}

	for _, middleware := range builder.middlewares.notifyPublish {
		handler = middleware(handler)
	}

	return handler
}

// createNotifyConfirm returns the base handler for Channel.NotifyConfirm
// that invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNotifyConfirm() amqpmiddleware.HandlerNotifyConfirm {
	// capture te event middleware in the closure
	eventMiddleware := builder.middlewares.notifyConfirmEvents

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsNotifyConfirm,
	) amqpmiddleware.ResultsNotifyConfirm {

		// Set up the innermost event handler.
		eventHandler := func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyConfirm) {
			notifyConfirmHandleAckAndNack(event.Confirmation, args.Ack, args.Nack)
		}

		// Wrap the event handler in the user-supplied middleware.
		for _, middleware := range eventMiddleware {
			eventHandler = middleware(eventHandler)
		}

		// Run the event relay.
		go builder.runNotifyConfirm(args, eventHandler)

		return amqpmiddleware.ResultsNotifyConfirm{Ack: args.Ack, Nack: args.Nack}
	}

	for _, middleware := range builder.middlewares.notifyConfirm {
		handler = middleware(handler)
	}

	return handler
}

// runNotifyConfirm relay the NotifyConfirm events to the caller by calling
// NotifyPublish.
func (builder channelHandlerBuilder) runNotifyConfirm(
	args amqpmiddleware.ArgsNotifyConfirm, eventHandler amqpmiddleware.HandlerNotifyConfirmEvents,
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
		eventHandler(make(amqpmiddleware.EventMetadata), event)
	}
}

// createNotifyConfirmOrOrphaned returns the base handler for
// Channel.NotifyConfirmOrOrphaned.
func (builder channelHandlerBuilder) createNotifyConfirmOrOrphaned() amqpmiddleware.HandlerNotifyConfirmOrOrphaned {
	eventMiddlewares := builder.middlewares.notifyConfirmOrOrphanedEvents
	channel := builder.channel

	handler := func(
		ctx context.Context, args amqpmiddleware.ArgsNotifyConfirmOrOrphaned,
	) amqpmiddleware.ResultsNotifyConfirmOrOrphaned {
		ack, nack, orphaned := args.Ack, args.Nack, args.Orphaned

		confirmsEvents := channel.NotifyPublish(
			make(chan datamodels.Confirmation, cap(ack)+cap(nack)+cap(orphaned)),
		)

		eventHandler := builder.createEventNotifyConfirmOrOrphaned(
			args, eventMiddlewares,
		)

		go channel.runNotifyConfirmOrOrphaned(
			eventHandler, ack, nack, orphaned, confirmsEvents,
		)

		return amqpmiddleware.ResultsNotifyConfirmOrOrphaned{
			Ack: args.Ack, Nack: args.Nack, Orphaned: args.Orphaned,
		}
	}

	return handler
}

// createEventNotifyConfirmOrOrphaned creates an event handler for event on
// Channel.NotifyConfirmOrOrphaned using user-supplied middleware.
func (builder channelHandlerBuilder) createEventNotifyConfirmOrOrphaned(
	args amqpmiddleware.ArgsNotifyConfirmOrOrphaned,
	eventMiddlewares []amqpmiddleware.NotifyConfirmOrOrphanedEvents,
) amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents {
	// Capture the channel in the closure
	eventHandler := func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyConfirmOrOrphaned) {
		confirmation := event.Confirmation
		if confirmation.DisconnectOrphan {
			args.Orphaned <- confirmation.DeliveryTag
		} else {
			notifyConfirmHandleAckAndNack(confirmation, args.Ack, args.Nack)
		}
	}

	for _, thisMiddleware := range eventMiddlewares {
		eventHandler = thisMiddleware(eventHandler)
	}

	return eventHandler
}

// invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNotifyReturn() amqpmiddleware.HandlerNotifyReturn {
	eventMiddlewares := builder.middlewares.notifyReturnEvents
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyReturn) amqpmiddleware.ResultsNotifyReturn {
		relay := newNotifyReturnRelay(args.Returns, eventMiddlewares)
		channel.eventRelaySetupAndLaunch(relay)
		return amqpmiddleware.ResultsNotifyReturn{Returns: args.Returns}
	}

	return handler
}

// createNotifyCancel returns the base handler for Channel.NotifyCancel that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNotifyCancel() amqpmiddleware.HandlerNotifyCancel {
	eventMiddlewares := builder.middlewares.notifyCancelEvents
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyCancel) amqpmiddleware.ResultsNotifyCancel {
		relay := newNotifyCancelRelay(args.Cancellations, eventMiddlewares)

		channel.eventRelaySetupAndLaunch(relay)
		return amqpmiddleware.ResultsNotifyCancel{Cancellations: args.Cancellations}
	}

	return handler
}

// createNotifyFlow returns the base handler for Channel.NotifyFlow that
// invokes the method of the underlying streadway/amqp.Channel.
func (builder channelHandlerBuilder) createNotifyFlow() amqpmiddleware.HandlerNotifyFlow {
	eventMiddlewares := builder.middlewares.notifyFlowEvents
	channel := builder.channel

	handler := func(ctx context.Context, args amqpmiddleware.ArgsNotifyFlow) amqpmiddleware.ResultsNotifyFlow {
		// Create a new event relay.
		relay := newNotifyFlowRelay(
			channel.ctx,
			args.FlowNotifications,
			eventMiddlewares,
		)

		// Setup and launch the relay.
		channel.eventRelaySetupAndLaunch(relay)
		return amqpmiddleware.ResultsNotifyFlow{FlowNotifications: args.FlowNotifications}
	}

	return handler
}
