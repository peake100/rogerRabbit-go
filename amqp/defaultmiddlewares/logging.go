package defaultmiddlewares
//
//import (
//	"github.com/peake100/rogerRabbit-go/amqp/amqpmiddleware"
//	"github.com/rs/zerolog"
//	streadway "github.com/streadway/amqp"
//	"sync"
//)
//
//// NewChannelLoggerFactory creates a new factory for making connection and channel
//// logger middleware.
//func NewLoggerFactories(
//	logger zerolog.Logger,
//	id string,
//	successLogLevel zerolog.Level,
//	logArgsResultsLevel zerolog.Level,
//) (connectionFactory, channelFactory amqpmiddleware.ProviderFactory) {
//	channelInstance := -1
//	connectionInstance := -1
//	instanceSync := new(sync.Mutex)
//
//	logger = logger.With().Str("ID", id).Logger()
//
//	connectionFactory = func() amqpmiddleware.ProvidesMiddleware {
//		instanceSync.Lock()
//		defer instanceSync.Unlock()
//
//		connectionInstance++
//		connLogger := logger.With().
//			Int("INSTANCE", connectionInstance).
//			Str("TRANSPORT", amqpmiddleware.TransportTypeConnection).
//			Logger()
//
//		return LoggingMiddlewareConnection{
//			loggingMiddlewareCore{
//				Logger: connLogger,
//				SuccessLogLevel: successLogLevel,
//				LogArgsResultsLevel: logArgsResultsLevel,
//			},
//		}
//	}
//
//	channelFactory = func() amqpmiddleware.ProvidesMiddleware {
//		instanceSync.Lock()
//		defer instanceSync.Unlock()
//
//		channelInstance++
//		chanLogger := logger.With().
//			Int("INSTANCE", channelInstance).
//			Str("TRANSPORT", amqpmiddleware.TransportTypeChannel).
//			Logger()
//
//		return LoggingMiddlewareChannel{
//			loggingMiddlewareCore{
//				Logger: chanLogger,
//				SuccessLogLevel: successLogLevel,
//				LogArgsResultsLevel: logArgsResultsLevel,
//			},
//		}
//	}
//
//	return connectionFactory, channelFactory
//}
//
//// LoggingMiddlewareID can be used to retrieve the running instance of loggingMiddlewareCore
//// during testing.
//const LoggingMiddlewareID amqpmiddleware.ProviderTypeID = "DefaultLogging"
//
//// loggingMiddlewareCore implements basic logging on every middleware available.
//type loggingMiddlewareCore struct {
//	// Logger is the root zerolog.Logger.
//	Logger zerolog.Logger
//	// SuccessLogLevel is the log level to log a successful method call at.
//	SuccessLogLevel zerolog.Level
//	// LogArgsResultsLevel is the log level to log method args, results or events at.
//	LogArgsResultsLevel zerolog.Level
//}
//
//func (middleware loggingMiddlewareCore) createMethodLogger(
//	methodName string,
//) zerolog.Logger {
//	return middleware.Logger.
//		With().
//		Str("METHOD_CALL", methodName).
//		Timestamp().
//		Logger()
//}
//
//func (middleware loggingMiddlewareCore) logMethod(
//	methodLogger zerolog.Logger,
//	args interface{},
//	results interface{},
//	err error,
//) {
//	var event *zerolog.Event
//	var eventLevel zerolog.Level
//	if err != nil {
//		event = methodLogger.Err(err).Stack()
//		eventLevel = zerolog.ErrorLevel
//	} else {
//		event = methodLogger.WithLevel(middleware.SuccessLogLevel)
//		eventLevel = middleware.SuccessLogLevel
//	}
//
//	// If this event is disabled, return immediately.
//	if !event.Enabled() {
//		return
//	}
//
//	if middleware.LogArgsResultsLevel >= eventLevel {
//		event.Interface("ARGS", args)
//		if err == nil && results != nil {
//			event.Interface("RESULTS", results)
//		}
//	}
//
//	event.Send()
//}
//
//func (middleware loggingMiddlewareCore) TypeID() amqpmiddleware.ProviderTypeID {
//	return LoggingMiddlewareID
//}
//
//func (middleware loggingMiddlewareCore) Close(next amqpmiddleware.HandlerClose) amqpmiddleware.HandlerClose {
//	logger := middleware.createMethodLogger("Close")
//	return func(args amqpmiddleware.ArgsClose) error {
//		err := next(args)
//		middleware.logMethod(logger, args, nil, err)
//		return err
//	}
//}
//
//func (middleware loggingMiddlewareCore) NotifyClose(next amqpmiddleware.HandlerNotifyClose) amqpmiddleware.HandlerNotifyClose {
//	logger := middleware.createMethodLogger("NotifyClose")
//	return func(args amqpmiddleware.ArgsNotifyClose) chan *streadway.Error {
//		results := next(args)
//		middleware.logMethod(logger, args, results, nil)
//		return results
//	}
//}
//
//func (middleware loggingMiddlewareCore) NotifyDial(next amqpmiddleware.HandlerNotifyDial) amqpmiddleware.HandlerNotifyDial {
//	logger := middleware.createMethodLogger("NotifyDial")
//}
//
//func (middleware loggingMiddlewareCore) NotifyDisconnect(next amqpmiddleware.HandlerNotifyDisconnect) amqpmiddleware.HandlerNotifyDisconnect {
//	logger := middleware.createMethodLogger("NotifyDisconnect")
//}
//
//func (middleware loggingMiddlewareCore) NotifyDialEvents(next amqpmiddleware.HandlerNotifyDialEvents) amqpmiddleware.HandlerNotifyDialEvents {
//	logger := middleware.createMethodLogger("NotifyDialEvents")
//}
//
//func (middleware loggingMiddlewareCore) NotifyDisconnectEvents(next amqpmiddleware.HandlerNotifyDisconnectEvents) amqpmiddleware.HandlerNotifyDisconnectEvents {
//	logger := middleware.createMethodLogger("NotifyDisconnectEvents")
//}
//
//func (middleware loggingMiddlewareCore) NotifyCloseEvents(next amqpmiddleware.HandlerNotifyCloseEvents) amqpmiddleware.HandlerNotifyCloseEvents {
//	logger := middleware.createMethodLogger("NotifyCloseEvents")
//}
//
//type LoggingMiddlewareConnection struct {
//	loggingMiddlewareCore
//}
//
//func (middleware LoggingMiddlewareConnection) ConnectionReconnect(next amqpmiddleware.HandlerConnectionReconnect) amqpmiddleware.HandlerConnectionReconnect {
//	panic("implement me")
//}
//
//type LoggingMiddlewareChannel struct {
//	loggingMiddlewareCore
//}
//
//func (middleware LoggingMiddlewareChannel) ChannelReconnect(next amqpmiddleware.HandlerChannelReconnect) amqpmiddleware.HandlerChannelReconnect {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueDeclare(next amqpmiddleware.HandlerQueueDeclare) amqpmiddleware.HandlerQueueDeclare {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueDeclarePassive(next amqpmiddleware.HandlerQueueDeclare) amqpmiddleware.HandlerQueueDeclare {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueInspect(next amqpmiddleware.HandlerQueueInspect) amqpmiddleware.HandlerQueueInspect {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueDelete(next amqpmiddleware.HandlerQueueDelete) amqpmiddleware.HandlerQueueDelete {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueBind(next amqpmiddleware.HandlerQueueBind) amqpmiddleware.HandlerQueueBind {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueueUnbind(next amqpmiddleware.HandlerQueueUnbind) amqpmiddleware.HandlerQueueUnbind {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QueuePurge(next amqpmiddleware.HandlerQueuePurge) amqpmiddleware.HandlerQueuePurge {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ExchangeDeclare(next amqpmiddleware.HandlerExchangeDeclare) amqpmiddleware.HandlerExchangeDeclare {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ExchangeDeclarePassive(next amqpmiddleware.HandlerExchangeDeclare) amqpmiddleware.HandlerExchangeDeclare {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ExchangeDelete(next amqpmiddleware.HandlerExchangeDelete) amqpmiddleware.HandlerExchangeDelete {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ExchangeBind(next amqpmiddleware.HandlerExchangeBind) amqpmiddleware.HandlerExchangeBind {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ExchangeUnbind(next amqpmiddleware.HandlerExchangeUnbind) amqpmiddleware.HandlerExchangeUnbind {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) QoS(next amqpmiddleware.HandlerQoS) amqpmiddleware.HandlerQoS {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Flow(next amqpmiddleware.HandlerFlow) amqpmiddleware.HandlerFlow {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Confirm(next amqpmiddleware.HandlerConfirm) amqpmiddleware.HandlerConfirm {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Publish(next amqpmiddleware.HandlerPublish) amqpmiddleware.HandlerPublish {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Get(next amqpmiddleware.HandlerGet) amqpmiddleware.HandlerGet {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Consume(next amqpmiddleware.HandlerConsume) amqpmiddleware.HandlerConsume {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Ack(next amqpmiddleware.HandlerAck) amqpmiddleware.HandlerAck {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Nack(next amqpmiddleware.HandlerNack) amqpmiddleware.HandlerNack {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) Reject(next amqpmiddleware.HandlerReject) amqpmiddleware.HandlerReject {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyPublish(next amqpmiddleware.HandlerNotifyPublish) amqpmiddleware.HandlerNotifyPublish {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyConfirm(next amqpmiddleware.HandlerNotifyConfirm) amqpmiddleware.HandlerNotifyConfirm {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyConfirmOrOrphaned(next amqpmiddleware.HandlerNotifyConfirmOrOrphaned) amqpmiddleware.HandlerNotifyConfirmOrOrphaned {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyReturn(next amqpmiddleware.HandlerNotifyReturn) amqpmiddleware.HandlerNotifyReturn {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyCancel(next amqpmiddleware.HandlerNotifyCancel) amqpmiddleware.HandlerNotifyCancel {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyFlow(next amqpmiddleware.HandlerNotifyFlow) amqpmiddleware.HandlerNotifyFlow {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyPublishEvents(next amqpmiddleware.HandlerNotifyPublishEvents) amqpmiddleware.HandlerNotifyPublishEvents {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) ConsumeEvents(next amqpmiddleware.HandlerConsumeEvents) amqpmiddleware.HandlerConsumeEvents {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyConfirmEvents(next amqpmiddleware.HandlerNotifyConfirmEvents) amqpmiddleware.HandlerNotifyConfirmEvents {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyConfirmOrOrphanedEvents(next amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents) amqpmiddleware.HandlerNotifyConfirmOrOrphanedEvents {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyReturnEvents(next amqpmiddleware.HandlerNotifyReturnEvents) amqpmiddleware.HandlerNotifyReturnEvents {
//	panic("implement me")
//}
//
//func (middleware LoggingMiddlewareChannel) NotifyFlowEvents(next amqpmiddleware.HandlerNotifyFlowEvents) amqpmiddleware.HandlerNotifyFlowEvents {
//	panic("implement me")
//}
//
//
//
