package cdp

import "encoding/json"

// Message represents a CDP protocol message (request or response).
type Message struct {
	ID        int             `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// Error represents a CDP protocol error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
