package amqpmiddleware

import "context"

// MethodInfo holds extracted information from the context passed into a method
// handler.
type MethodInfo struct {
	// OpAttempt: Some operations retry over disconnections. This is the attempt number,
	// starting at 0. Will be -1 if this is not an operation that is re-tried.
	OpAttempt int
}

// GetMethodInfo extracts context info from a middleware context.
func GetMethodInfo(ctx context.Context) (info MethodInfo) {
	if opAttempt, ok := ctx.Value("opAttempt").(int); ok {
		info.OpAttempt = opAttempt
	} else {
		info.OpAttempt = -1
	}

	return info
}

// EventMetadata is metadata passed into an event handler. Events cannot be cancelled,
// and therefore do not have contexts, so middleware-centric metadata should be added to
// and fetched from this argument.
type EventMetadata map[string]interface{}

// EventInfo is information provided from the base library  that can be extracted from
// EventMetadata.
type EventInfo struct {
	// EventNum is the event number starting at 0. For Relays, this value will reset
	// each leg. -1 if unknown.
	EventNum int64
	// RelayLeg is the leg of the relay we are on. Each time a connection is
	// re-established, we hit a new relay "leg".
	//
	// -1 if this event feed does not pull directly from an underlying connection, and
	// therefore does not have a "LEG"
	RelayLeg int
}

// GetKey gets key from metadata. Returns nil if key could not be found
func (metadata EventMetadata) GetKey(key string) interface{} {
	value, _ := metadata[key]
	return value
}

// GetEventInfo extracts EventInfo from EventMetadata.
func GetEventInfo(metadata EventMetadata) (info EventInfo) {
	if eventNum, ok := metadata.GetKey("EventNum").(int64); ok {
		info.EventNum = eventNum
	} else {
		info.EventNum = -1
	}

	if relayLeg, ok := metadata.GetKey("LegNum").(int); ok {
		info.RelayLeg = relayLeg
	} else {
		info.RelayLeg = -1
	}

	return info
}
