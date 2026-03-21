package backend

import "encoding/json"

// EventHandler is called when a browser protocol event is received.
// sessionID identifies which page session the event belongs to (empty for browser events).
type EventHandler = func(sessionID string, params json.RawMessage)

// Backend is the interface for browser protocol backends (Juggler, BiDi).
type Backend interface {
	Call(sessionID, method string, params json.RawMessage) (json.RawMessage, error)
	Subscribe(event string, handler EventHandler)
	Close() error
}
