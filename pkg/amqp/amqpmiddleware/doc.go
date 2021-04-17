/*
package amqpMiddleware defines middleware signatures for methods on *amqp.Channel.

Using per-method middleware allows for a general system of configuring behavior across
reconnection events without each feature needing to be a bespoke bit of logic spanning
several methods in the Channel type.

By implementing features like consistent caller-facing delivery tags across reconnect
as middleware, we can collect all logic for that feature in a single place, making the
logic of the feature easier to track.

The middleware approach also makes this library highly-extensible by the end user,
an added bonus.
*/
package amqpmiddleware
