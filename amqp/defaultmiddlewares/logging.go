package defaultmiddlewares


import (
	"context"
	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
	"github.com/rs/zerolog"
	"sync"
)

const MetadataKey = "DefaultLogger"

// LoggingMiddlewareID can be used to retrieve the running instance of loggingMiddlewareCore
// during testing.
const LoggingMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultLogging"

// loggingMiddlewareCore implements basic logging on every middleware available.
type loggingMiddlewareCore struct {
	// Logger is the root zerolog.Logger.
	Logger zerolog.Logger
	// SuccessLogLevel is the log level to log a successful method call at.
	SuccessLogLevel zerolog.Level
	// LogArgsResultsLevel is the log level to log method args, results or events at.
	LogArgsResultsLevel zerolog.Level
}

func (middleware loggingMiddlewareCore) createMethodLogger(
	methodName string,
) zerolog.Logger {
	return middleware.Logger.
		With().
		Str("METHOD_CALL", methodName).
		Timestamp().
		Logger()
}

func (middleware loggingMiddlewareCore) createEventLogger(
	eventType string,
) zerolog.Logger {
	return middleware.Logger.
		With().
		Str("EVENTS", eventType).
		Timestamp().
		Logger()
}

func (middleware loggingMiddlewareCore) logMethod(
	ctx context.Context,
	methodLogger zerolog.Logger,
	args interface{},
	results interface{},
	err error,
) {
	var event *zerolog.Event
	var eventLevel zerolog.Level

	if err != nil {
		event = methodLogger.Err(err).Stack()
		eventLevel = zerolog.ErrorLevel
	} else {
		event = methodLogger.WithLevel(middleware.SuccessLogLevel)
		eventLevel = middleware.SuccessLogLevel
	}

	// If this event is disabled, return immediately.
	if !event.Enabled() {
		return
	}

	// Add the op attempt info
	methodInfo := amqpmiddleware.GetMethodInfo(ctx)
	event.Int("OP_ATTEMPT", methodInfo.OpAttempt)

	if middleware.LogArgsResultsLevel >= eventLevel {
		event.Interface("zARGS", args)
		if err == nil && results != nil {
			event.Interface("zRESULTS", results)
		}
	}

	event.Send()
}

func (middleware loggingMiddlewareCore) logEvent(
	meta amqpmiddleware.EventMetadata,
	eventLogger zerolog.Logger,
	eventVal interface{},
) {
	event := eventLogger.WithLevel(middleware.SuccessLogLevel)
	eventLevel := middleware.SuccessLogLevel

	// If this event is disabled, return immediately.
	if !event.Enabled() {
		return
	}

	// Add the op attempt info
	eventInfo := amqpmiddleware.GetEventInfo(meta)
	if eventInfo.EventNum > -1 {
		event.Int64("NUM", eventInfo.EventNum)
	}
	if eventInfo.RelayLeg > -1 {
		event.Int("RELAY_LEG", eventInfo.RelayLeg)
	}

	if middleware.LogArgsResultsLevel >= eventLevel {
		event.Interface("VALUE", eventVal)
	}

	event.Send()
}

func (loggingMiddlewareCore) addCtxLogger(
	ctx context.Context, methodLogger zerolog.Logger,
) context.Context {
	return context.WithValue(ctx, MetadataKey, methodLogger)
}

func (loggingMiddlewareCore) addMetadataLogger(
	meta amqpmiddleware.EventMetadata, methodLogger zerolog.Logger,
) amqpmiddleware.EventMetadata {
	meta[MetadataKey] = methodLogger
	return meta
}

func (middleware loggingMiddlewareCore) TypeID() amqpmiddleware.ProviderTypeID {
	return LoggingMiddlewareID
}

func (middleware loggingMiddlewareCore) Close(next amqpmiddleware.HandlerClose) amqpmiddleware.HandlerClose {
	logger := middleware.createMethodLogger("Close")
	return func(ctx context.Context, args amqpmiddleware.ArgsClose) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware loggingMiddlewareCore) NotifyClose(next amqpmiddleware.HandlerNotifyClose) amqpmiddleware.HandlerNotifyClose {
	logger := middleware.createMethodLogger("NotifyClose")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyClose) amqpmiddleware.ResultsNotifyClose {
		ctx = middleware.addCtxLogger(ctx, logger)
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware loggingMiddlewareCore) NotifyDial(next amqpmiddleware.HandlerNotifyDial) amqpmiddleware.HandlerNotifyDial {
	logger := middleware.createMethodLogger("NotifyDial")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyDial) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware loggingMiddlewareCore) NotifyDisconnect(next amqpmiddleware.HandlerNotifyDisconnect) amqpmiddleware.HandlerNotifyDisconnect {
	logger := middleware.createMethodLogger("NotifyDisconnect")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyDisconnect) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware loggingMiddlewareCore) NotifyDialEvents(
	next amqpmiddleware.HandlerNotifyDialEvents,
) amqpmiddleware.HandlerNotifyDialEvents {
	logger := middleware.createEventLogger("NotifyDial")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyDial) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware loggingMiddlewareCore) NotifyDisconnectEvents(next amqpmiddleware.HandlerNotifyDisconnectEvents) amqpmiddleware.HandlerNotifyDisconnectEvents {
	logger := middleware.createMethodLogger("NotifyDisconnectEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyDisconnect) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware loggingMiddlewareCore) NotifyCloseEvents(next amqpmiddleware.HandlerNotifyCloseEvents) amqpmiddleware.HandlerNotifyCloseEvents {
	logger := middleware.createMethodLogger("NotifyCloseEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyClose) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

type LoggingMiddlewareConnection struct {
	loggingMiddlewareCore
}

func (middleware LoggingMiddlewareConnection) ConnectionReconnect(next amqpmiddleware.HandlerConnectionReconnect) amqpmiddleware.HandlerConnectionReconnect {
	logger := middleware.createMethodLogger("ConnectionReconnect")
	return func(
		ctx context.Context, args amqpmiddleware.ArgsConnectionReconnect,
	) (amqpmiddleware.ResultsConnectionReconnect, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

type LoggingMiddlewareChannel struct {
	loggingMiddlewareCore
}

func (middleware LoggingMiddlewareChannel) ChannelReconnect(next amqpmiddleware.HandlerChannelReconnect) amqpmiddleware.HandlerChannelReconnect {
	logger := middleware.createMethodLogger("ConnectionReconnect")
	return func(ctx context.Context, args amqpmiddleware.ArgsChannelReconnect) (amqpmiddleware.ResultsChannelReconnect, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) QueueDeclare(next amqpmiddleware.HandlerQueueDeclare) amqpmiddleware.HandlerQueueDeclare {
	logger := middleware.createMethodLogger("QueueDeclare")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueDeclare) (amqpmiddleware.ResultsQueueDeclare, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) QueueDeclarePassive(next amqpmiddleware.HandlerQueueDeclare) amqpmiddleware.HandlerQueueDeclare {
	logger := middleware.createMethodLogger("QueueDeclarePassive")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueDeclare) (amqpmiddleware.ResultsQueueDeclare, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) QueueInspect(next amqpmiddleware.HandlerQueueInspect) amqpmiddleware.HandlerQueueInspect {
	logger := middleware.createMethodLogger("QueueInspect")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueInspect) (amqpmiddleware.ResultsQueueInspect, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) QueueDelete(next amqpmiddleware.HandlerQueueDelete) amqpmiddleware.HandlerQueueDelete {
	logger := middleware.createMethodLogger("QueueDelete")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueDelete) (amqpmiddleware.ResultsQueueDelete, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) QueueBind(next amqpmiddleware.HandlerQueueBind) amqpmiddleware.HandlerQueueBind {
	logger := middleware.createMethodLogger("QueueBind")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueBind) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) QueueUnbind(next amqpmiddleware.HandlerQueueUnbind) amqpmiddleware.HandlerQueueUnbind {
	logger := middleware.createMethodLogger("QueueUnbind")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueueUnbind) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) QueuePurge(next amqpmiddleware.HandlerQueuePurge) amqpmiddleware.HandlerQueuePurge {
	logger := middleware.createMethodLogger("QueuePurge")
	return func(ctx context.Context, args amqpmiddleware.ArgsQueuePurge) (amqpmiddleware.ResultsQueuePurge, error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		result, err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, result, err)
		return result, err
	}
}

func (middleware LoggingMiddlewareChannel) ExchangeDeclare(next amqpmiddleware.HandlerExchangeDeclare) amqpmiddleware.HandlerExchangeDeclare {
	logger := middleware.createMethodLogger("ExchangeDeclare")
	return func(ctx context.Context, args amqpmiddleware.ArgsExchangeDeclare) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) ExchangeDeclarePassive(next amqpmiddleware.HandlerExchangeDeclare) amqpmiddleware.HandlerExchangeDeclare {
	logger := middleware.createMethodLogger("ExchangeDeclarePassive")
	return func(ctx context.Context, args amqpmiddleware.ArgsExchangeDeclare) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) ExchangeDelete(next amqpmiddleware.HandlerExchangeDelete) amqpmiddleware.HandlerExchangeDelete {
	logger := middleware.createMethodLogger("ExchangeDelete")
	return func(ctx context.Context, args amqpmiddleware.ArgsExchangeDelete) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) ExchangeBind(next amqpmiddleware.HandlerExchangeBind) amqpmiddleware.HandlerExchangeBind {
	logger := middleware.createMethodLogger("ExchangeBind")
	return func(ctx context.Context, args amqpmiddleware.ArgsExchangeBind) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) ExchangeUnbind(next amqpmiddleware.HandlerExchangeUnbind) amqpmiddleware.HandlerExchangeUnbind {
	logger := middleware.createMethodLogger("ExchangeUnbind")
	return func(ctx context.Context, args amqpmiddleware.ArgsExchangeUnbind) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) QoS(next amqpmiddleware.HandlerQoS) amqpmiddleware.HandlerQoS {
	logger := middleware.createMethodLogger("QoS")
	return func(ctx context.Context, args amqpmiddleware.ArgsQoS) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Flow(next amqpmiddleware.HandlerFlow) amqpmiddleware.HandlerFlow {
	logger := middleware.createMethodLogger("Flow")
	return func(ctx context.Context, args amqpmiddleware.ArgsFlow) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Confirm(next amqpmiddleware.HandlerConfirm) amqpmiddleware.HandlerConfirm {
	logger := middleware.createMethodLogger("Confirm")
	return func(ctx context.Context, args amqpmiddleware.ArgsConfirms) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Publish(next amqpmiddleware.HandlerPublish) amqpmiddleware.HandlerPublish {
	logger := middleware.createMethodLogger("Publish")
	return func(ctx context.Context, args amqpmiddleware.ArgsPublish) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Get(next amqpmiddleware.HandlerGet) amqpmiddleware.HandlerGet {
	logger := middleware.createMethodLogger("Get")
	return func(ctx context.Context, args amqpmiddleware.ArgsGet) (results amqpmiddleware.ResultsGet, err error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err = next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) Consume(next amqpmiddleware.HandlerConsume) amqpmiddleware.HandlerConsume {
	logger := middleware.createMethodLogger("Consume")
	return func(ctx context.Context, args amqpmiddleware.ArgsConsume) (results amqpmiddleware.ResultsConsume, err error) {
		ctx = middleware.addCtxLogger(ctx, logger)
		results, err = next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return results, err
	}
}

func (middleware LoggingMiddlewareChannel) Ack(next amqpmiddleware.HandlerAck) amqpmiddleware.HandlerAck {
	logger := middleware.createMethodLogger("Ack")
	return func(ctx context.Context, args amqpmiddleware.ArgsAck) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Nack(next amqpmiddleware.HandlerNack) amqpmiddleware.HandlerNack {
	logger := middleware.createMethodLogger("Nack")
	return func(ctx context.Context, args amqpmiddleware.ArgsNack) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) Reject(next amqpmiddleware.HandlerReject) amqpmiddleware.HandlerReject {
	logger := middleware.createMethodLogger("Reject")
	return func(ctx context.Context, args amqpmiddleware.ArgsReject) error {
		ctx = middleware.addCtxLogger(ctx, logger)
		err := next(ctx, args)
		middleware.logMethod(ctx, logger, args, nil, err)
		return err
	}
}

func (middleware LoggingMiddlewareChannel) NotifyPublish(next amqpmiddleware.HandlerNotifyPublish) amqpmiddleware.HandlerNotifyPublish {
	logger := middleware.createMethodLogger("NotifyPublish")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyPublish) amqpmiddleware.ResultsNotifyPublish {ctx = middleware.addCtxLogger(ctx, logger)
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyConfirm(next amqpmiddleware.HandlerNotifyConfirm) amqpmiddleware.HandlerNotifyConfirm {
	logger := middleware.createMethodLogger("NotifyConfirm")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyConfirm) amqpmiddleware.ResultsNotifyConfirm {
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyConfirmOrOrphaned(next amqpmiddleware.HandlerNotifyConfirmOrOrphaned) amqpmiddleware.HandlerNotifyConfirmOrOrphaned {
	logger := middleware.createMethodLogger("NotifyConfirmOrOrphaned")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyConfirmOrOrphaned) amqpmiddleware.ResultsNotifyConfirmOrOrphaned {
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyReturn(next amqpmiddleware.HandlerNotifyReturn) amqpmiddleware.HandlerNotifyReturn {
	logger := middleware.createMethodLogger("NotifyReturn")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyReturn) amqpmiddleware.ResultsNotifyReturn {
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyCancel(next amqpmiddleware.HandlerNotifyCancel) amqpmiddleware.HandlerNotifyCancel {
	logger := middleware.createMethodLogger("NotifyCancel")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyCancel) amqpmiddleware.ResultsNotifyCancel {
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyFlow(next amqpmiddleware.HandlerNotifyFlow) amqpmiddleware.HandlerNotifyFlow {
	logger := middleware.createMethodLogger("NotifyFlow")
	return func(ctx context.Context, args amqpmiddleware.ArgsNotifyFlow) amqpmiddleware.ResultsNotifyFlow {
		results := next(ctx, args)
		middleware.logMethod(ctx, logger, args, results, nil)
		return results
	}
}

func (middleware LoggingMiddlewareChannel) NotifyPublishEvents(next amqpmiddleware.HandlerNotifyPublishEvents) amqpmiddleware.HandlerNotifyPublishEvents {
	logger := middleware.createEventLogger("NotifyPublishEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyPublish) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware LoggingMiddlewareChannel) ConsumeEvents(next amqpmiddleware.HandlerConsumeEvents) amqpmiddleware.HandlerConsumeEvents {
	logger := middleware.createEventLogger("ConsumeEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventConsume) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware LoggingMiddlewareChannel) NotifyConfirmEvents(next amqpmiddleware.HandlerNotifyConfirmEvents) amqpmiddleware.HandlerNotifyConfirmEvents {
	logger := middleware.createEventLogger("NotifyConfirmEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyConfirm) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware LoggingMiddlewareChannel) NotifyConfirmOrOrphanedEvents(next amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents) amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents {
	logger := middleware.createEventLogger("NotifyConfirmOrOrphanedEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyConfirmOrOrphaned) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware LoggingMiddlewareChannel) NotifyReturnEvents(next amqpmiddleware.HandlerNotifyReturnEvents) amqpmiddleware.HandlerNotifyReturnEvents {
	logger := middleware.createEventLogger("NotifyReturnEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyReturn) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}

func (middleware LoggingMiddlewareChannel) NotifyFlowEvents(next amqpmiddleware.HandlerNotifyFlowEvents) amqpmiddleware.HandlerNotifyFlowEvents {
	logger := middleware.createEventLogger("NotifyFlowEvents")
	return func(metadata amqpmiddleware.EventMetadata, event amqpmiddleware.EventNotifyFlow) {
		middleware.logEvent(metadata, logger, event)
		metadata = middleware.addMetadataLogger(metadata, logger)
		next(metadata, event)
	}
}


// NewChannelLoggerFactory creates a new factory for making connection and channel
// logger middleware.
func NewLoggerFactories(
	logger zerolog.Logger,
	id string,
	successLogLevel zerolog.Level,
	logArgsResultsLevel zerolog.Level,
) (connectionFactory, channelFactory amqpmiddleware.ProviderFactory) {
	channelInstance := -1
	connectionInstance := -1
	instanceSync := new(sync.Mutex)

	logger = logger.With().Str("ID", id).Logger()

	connectionFactory = func() amqpmiddleware.ProvidesMiddleware {
		instanceSync.Lock()
		defer instanceSync.Unlock()

		connectionInstance++
		connLogger := logger.With().
			Int("INSTANCE", connectionInstance).
			Str("TRANSPORT", amqpmiddleware.TransportTypeConnection).
			Logger()

		return LoggingMiddlewareConnection{
			loggingMiddlewareCore{
				Logger: connLogger,
				SuccessLogLevel: successLogLevel,
				LogArgsResultsLevel: logArgsResultsLevel,
			},
		}
	}

	channelFactory = func() amqpmiddleware.ProvidesMiddleware {
		instanceSync.Lock()
		defer instanceSync.Unlock()

		channelInstance++
		chanLogger := logger.With().
			Int("INSTANCE", channelInstance).
			Str("TRANSPORT", amqpmiddleware.TransportTypeChannel).
			Logger()

		return LoggingMiddlewareChannel{
			loggingMiddlewareCore{
				Logger: chanLogger,
				SuccessLogLevel: successLogLevel,
				LogArgsResultsLevel: logArgsResultsLevel,
			},
		}
	}

	return connectionFactory, channelFactory
}
