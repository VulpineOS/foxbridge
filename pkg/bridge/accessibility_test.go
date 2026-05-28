package bridge

import (
	"encoding/json"
	"testing"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
)

func TestHandleAccessibility_Enable(t *testing.T) {
	b, _ := newTestBridge()
	msg := &cdp.Message{ID: 1, Method: "Accessibility.enable", Params: json.RawMessage(`{}`)}
	result, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr != nil {
		t.Fatalf("unexpected error: %s", cdpErr.Message)
	}
	if string(result) != "{}" {
		t.Errorf("result = %s, want {}", string(result))
	}
}

func TestHandleAccessibility_Disable(t *testing.T) {
	b, _ := newTestBridge()
	msg := &cdp.Message{ID: 1, Method: "Accessibility.disable", Params: json.RawMessage(`{}`)}
	result, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr != nil {
		t.Fatalf("unexpected error: %s", cdpErr.Message)
	}
	if string(result) != "{}" {
		t.Errorf("result = %s, want {}", string(result))
	}
}

func TestHandleAccessibility_GetFullAXTree(t *testing.T) {
	b, mb := newTestBridge()
	mb.SetResponse("", "Accessibility.getFullAXTree",
		json.RawMessage(`{"nodes":[{"nodeId":"1","role":{"value":"document"}}]}`), nil)

	msg := &cdp.Message{ID: 1, Method: "Accessibility.getFullAXTree", Params: json.RawMessage(`{}`)}
	result, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr != nil {
		t.Fatalf("unexpected error: %s", cdpErr.Message)
	}

	var res struct {
		Nodes []map[string]interface{} `json:"nodes"`
	}
	json.Unmarshal(result, &res)
	if len(res.Nodes) != 1 {
		t.Errorf("nodes length = %d, want 1", len(res.Nodes))
	}
}

func TestHandleAccessibility_GetFullAXTree_AdaptsJugglerTree(t *testing.T) {
	b, mb := newTestBridge()
	mb.SetResponse("", "Accessibility.getFullAXTree", json.RawMessage(`{
		"tree": {
			"role": "document",
			"name": "Agent Audit",
			"children": [
				{"role": "heading", "name": "Agent Audit", "level": 1},
				{"role": "pushbutton", "name": "Action Button", "focusable": true},
				{"role": "text leaf", "name": "ready"}
			]
		},
		"filtered": true
	}`), nil)

	msg := &cdp.Message{ID: 1, Method: "Accessibility.getFullAXTree", Params: json.RawMessage(`{}`)}
	result, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr != nil {
		t.Fatalf("unexpected error: %s", cdpErr.Message)
	}

	var res struct {
		Nodes []struct {
			NodeID     string   `json:"nodeId"`
			ChildIDs   []string `json:"childIds"`
			Role       axTestValue
			Name       axTestValue
			Properties []struct {
				Name  string      `json:"name"`
				Value axTestValue `json:"value"`
			} `json:"properties"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(result, &res); err != nil {
		t.Fatalf("unmarshal adapted tree: %v\n%s", err, result)
	}
	if len(res.Nodes) != 4 {
		t.Fatalf("nodes length = %d, want 4; result=%s", len(res.Nodes), result)
	}
	if res.Nodes[0].Role.Value != "RootWebArea" {
		t.Fatalf("root role = %v, want RootWebArea", res.Nodes[0].Role.Value)
	}
	if got, want := res.Nodes[0].ChildIDs, []string{"2", "3", "4"}; !equalStrings(got, want) {
		t.Fatalf("root childIds = %v, want %v", got, want)
	}
	if res.Nodes[2].Role.Value != "button" || res.Nodes[2].Name.Value != "Action Button" {
		t.Fatalf("button node = %+v", res.Nodes[2])
	}
	if !hasAXProperty(res.Nodes[2].Properties, "focusable", true) {
		t.Fatalf("button node missing focusable property: %+v", res.Nodes[2].Properties)
	}
}

func TestHandleAccessibility_GetFullAXTree_SessionResolution(t *testing.T) {
	b, mb := newTestBridge()
	b.sessions.Add(&cdp.SessionInfo{
		SessionID:        "cdp-s1",
		JugglerSessionID: "jug-s1",
		TargetID:         "t1",
	})
	mb.SetResponse("jug-s1", "Accessibility.getFullAXTree", json.RawMessage(`{"nodes":[]}`), nil)

	msg := &cdp.Message{ID: 1, Method: "Accessibility.getFullAXTree", SessionID: "cdp-s1", Params: json.RawMessage(`{}`)}
	_, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr != nil {
		t.Fatalf("unexpected error: %s", cdpErr.Message)
	}

	last, _ := mb.LastCall()
	if last.SessionID != "jug-s1" {
		t.Errorf("sessionID = %q, want jug-s1", last.SessionID)
	}
}

func TestHandleAccessibility_UnknownMethod(t *testing.T) {
	b, _ := newTestBridge()
	msg := &cdp.Message{ID: 1, Method: "Accessibility.doesNotExist", Params: json.RawMessage(`{}`)}
	_, cdpErr := b.handleAccessibility(nil, msg)
	if cdpErr == nil {
		t.Fatal("expected error for unknown method")
	}
	if cdpErr.Code != -32601 {
		t.Errorf("error code = %d, want -32601", cdpErr.Code)
	}
}

type axTestValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasAXProperty(properties []struct {
	Name  string      `json:"name"`
	Value axTestValue `json:"value"`
}, name string, value interface{}) bool {
	for _, property := range properties {
		if property.Name == name && property.Value.Value == value {
			return true
		}
	}
	return false
}
