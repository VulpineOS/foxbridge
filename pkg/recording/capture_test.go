package recording

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
)

func TestRecorderRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}

	inbound := &cdp.Message{ID: 1, Method: "Browser.getVersion", Params: json.RawMessage(`{}`)}
	outbound := &cdp.Message{ID: 1, Result: json.RawMessage(`{"product":"foxbridge"}`)}
	if err := recorder.Record("in", inbound); err != nil {
		t.Fatalf("Record inbound: %v", err)
	}
	if err := recorder.Record("out", outbound); err != nil {
		t.Fatalf("Record outbound: %v", err)
	}
	if err := recorder.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if entries[0].Seq != 1 || entries[0].Direction != "in" || entries[0].Message.Method != "Browser.getVersion" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Seq != 2 || entries[1].Direction != "out" || string(entries[1].Message.Result) != `{"product":"foxbridge"}` {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
}

func TestLoad_AllowsLargeFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	recorder, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}

	large := strings.Repeat("x", 128*1024)
	msg := &cdp.Message{ID: 1, Method: "Runtime.evaluate", Params: json.RawMessage(`{"expression":"` + large + `"}`)}
	if err := recorder.Record("in", msg); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := recorder.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
	if got := string(entries[0].Message.Params); len(got) != len(msg.Params) {
		t.Fatalf("params len = %d, want %d", len(got), len(msg.Params))
	}
}
