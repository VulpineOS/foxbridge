package juggler

import "encoding/json"

// Message represents a Juggler protocol message.
// Requests have ID+Method+Params, responses have ID+Result/Error,
// events have Method+Params (no ID).
type Message struct {
	ID        int             `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// Error represents a Juggler protocol error.
type Error struct {
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *Error) Error() string {
	if e.Data != "" {
		return e.Message + ": " + e.Data
	}
	return e.Message
}

// IsEvent returns true if this message is an event (no ID).
func (m *Message) IsEvent() bool {
	return m.ID == 0 && m.Method != ""
}

// IsResponse returns true if this message is a response (has ID, no Method).
func (m *Message) IsResponse() bool {
	return m.ID != 0 && m.Method == ""
}
