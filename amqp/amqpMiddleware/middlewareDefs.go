package amqpMiddleware

// Middleware definitions for channel methods.

type Reconnect = func(next HandlerReconnect) HandlerReconnect

type QueueDeclare = func(next HandlerQueueDeclare) HandlerQueueDeclare

type QueueDelete = func(next HandlerQueueDelete) HandlerQueueDelete

type QueueBind = func(next HandlerQueueBind) HandlerQueueBind

type QueueUnbind = func(next HandlerQueueUnbind) HandlerQueueUnbind

type ExchangeDeclare func(next HandlerExchangeDeclare) HandlerExchangeDeclare

type ExchangeDelete func(next HandlerExchangeDelete) HandlerExchangeDelete

type ExchangeBind func(next HandlerExchangeBind) HandlerExchangeBind

type ExchangeUnbind func(next HandlerExchangeUnbind) HandlerExchangeUnbind

type QoS func(next HandlerQoS) HandlerQoS

type Confirm func(next HandlerConfirm) HandlerConfirm
