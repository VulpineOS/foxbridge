package recording

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
	"github.com/gorilla/websocket"
)

func TestReplayServer_ReplaysFramesInRecordedOrder(t *testing.T) {
	entries := []Entry{
		{
			Seq:       1,
			Direction: "in",
			Message:   &cdp.Message{ID: 1, Method: "Browser.getVersion", Params: json.RawMessage(`{}`)},
		},
		{
			Seq:       2,
			Direction: "out",
			Message:   &cdp.Message{ID: 1, Result: json.RawMessage(`{"product":"foxbridge"}`)},
		},
		{
			Seq:       3,
			Direction: "out",
			Message:   &cdp.Message{Method: "Page.loadEventFired", Params: json.RawMessage(`{"timestamp":0}`)},
		},
		{
			Seq:       4,
			Direction: "in",
			Message:   &cdp.Message{ID: 2, Method: "Page.enable", Params: json.RawMessage(`{}`)},
		},
		{
			Seq:       5,
			Direction: "out",
			Message:   &cdp.Message{ID: 2, Result: json.RawMessage(`{}`)},
		},
	}

	server := NewReplayServer(0, entries)
	ts := httptest.NewServer(server.mux())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/devtools/browser/foxbridge"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(entries[0].Message); err != nil {
		t.Fatalf("WriteJSON first request: %v", err)
	}

	var first cdp.Message
	if err := conn.ReadJSON(&first); err != nil {
		t.Fatalf("ReadJSON first response: %v", err)
	}
	if first.ID != 1 || string(first.Result) != `{"product":"foxbridge"}` {
		t.Fatalf("unexpected first response: %+v", first)
	}

	var event cdp.Message
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON event: %v", err)
	}
	if event.Method != "Page.loadEventFired" {
		t.Fatalf("unexpected event: %+v", event)
	}

	if err := conn.WriteJSON(entries[3].Message); err != nil {
		t.Fatalf("WriteJSON second request: %v", err)
	}

	var second cdp.Message
	if err := conn.ReadJSON(&second); err != nil {
		t.Fatalf("ReadJSON second response: %v", err)
	}
	if second.ID != 2 || string(second.Result) != `{}` {
		t.Fatalf("unexpected second response: %+v", second)
	}
}
