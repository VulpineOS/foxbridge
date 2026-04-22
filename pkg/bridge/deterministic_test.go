package bridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
)

func TestBuildDeterministicScript_UsesDefaults(t *testing.T) {
	script := buildDeterministicScript(DeterministicConfig{TimeMS: 1234})
	if script == "" {
		t.Fatal("expected deterministic script")
	}
	for _, want := range []string{
		"Date",
		"Math",
		"performance",
		"getRandomValues",
		"__foxbridgeSeed = 1 >>> 0",
		"__foxbridgeBaseTime = 1234",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("expected %q in deterministic script", want)
		}
	}
}

func TestSetupEventSubscriptions_AttachedToTarget_AppliesDeterministicPrelude(t *testing.T) {
	b, mb := newTestBridge()
	b.SetDeterministicConfig(DeterministicConfig{TimeMS: 1234, Seed: 99})
	b.SetupEventSubscriptions()

	mb.mu.Lock()
	handlers := mb.handlers["Browser.attachedToTarget"]
	mb.mu.Unlock()
	if len(handlers) == 0 {
		t.Fatal("no handlers registered for Browser.attachedToTarget")
	}

	params := json.RawMessage(`{
		"sessionId": "jug-s1",
		"targetInfo": {
			"targetId": "t1",
			"browserContextId": "ctx-1",
			"type": "page",
			"url": "about:blank"
		}
	}`)
	handlers[0]("", params)
	time.Sleep(10 * time.Millisecond)

	info, ok := b.sessions.GetByTarget("t1")
	if !ok {
		t.Fatal("expected session for target t1")
	}

	addCalls := mb.CallsForMethod("Page.addScriptToEvaluateOnNewDocument")
	if len(addCalls) != 1 {
		t.Fatalf("addScriptToEvaluateOnNewDocument calls = %d, want 1", len(addCalls))
	}
	if addCalls[0].SessionID != info.JugglerSessionID {
		t.Fatalf("addScript session = %q, want %q", addCalls[0].SessionID, info.JugglerSessionID)
	}

	evalCalls := mb.CallsForMethod("Runtime.evaluate")
	if len(evalCalls) != 1 {
		t.Fatalf("Runtime.evaluate calls = %d, want 1", len(evalCalls))
	}
	if evalCalls[0].SessionID != info.JugglerSessionID {
		t.Fatalf("Runtime.evaluate session = %q, want %q", evalCalls[0].SessionID, info.JugglerSessionID)
	}
}

func TestHandleTarget_CreateTarget_AppliesDeterministicPreludeBeforeNavigate(t *testing.T) {
	b, mb := newTestBridge()
	b.SetDeterministicConfig(DeterministicConfig{TimeMS: 1234, Seed: 42})
	mb.SetResponse("", "Browser.newPage", json.RawMessage(`{"targetId":"page-1"}`), nil)

	go func() {
		time.Sleep(20 * time.Millisecond)
		b.sessions.Add(&cdp.SessionInfo{
			SessionID:        "page-s1",
			JugglerSessionID: "jug-s1",
			TargetID:         "page-1",
			FrameID:          "frame-1",
			Type:             "page",
		})
	}()

	msg := &cdp.Message{
		ID:     1,
		Method: "Target.createTarget",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}

	if _, cdpErr := b.handleTarget(nil, msg); cdpErr != nil {
		t.Fatalf("handleTarget returned error: %v", cdpErr.Message)
	}

	var addIdx, evalIdx, navIdx int = -1, -1, -1
	mb.mu.Lock()
	for i, call := range mb.calls {
		switch call.Method {
		case "Page.addScriptToEvaluateOnNewDocument":
			addIdx = i
		case "Runtime.evaluate":
			evalIdx = i
		case "Page.navigate":
			navIdx = i
		}
	}
	mb.mu.Unlock()

	if addIdx == -1 || evalIdx == -1 || navIdx == -1 {
		t.Fatalf("expected addScript, Runtime.evaluate, and Page.navigate calls")
	}
	if addIdx > navIdx {
		t.Fatalf("expected addScript before navigate, got add=%d navigate=%d", addIdx, navIdx)
	}
	if evalIdx > navIdx {
		t.Fatalf("expected Runtime.evaluate before navigate, got eval=%d navigate=%d", evalIdx, navIdx)
	}
}
