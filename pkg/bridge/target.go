package bridge

import (
	"encoding/json"
	"fmt"

	"github.com/PopcornDev1/foxbridge/pkg/cdp"
	"github.com/google/uuid"
)

func (b *Bridge) handleTarget(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "Target.setDiscoverTargets":
		// Enable browser-level events in Juggler.
		_, err := b.callJuggler("", "Browser.enable", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return json.RawMessage(`{}`), nil

	case "Target.createTarget":
		var params struct {
			URL              string `json:"url"`
			BrowserContextID string `json:"browserContextId"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, &cdp.Error{Code: -32602, Message: "invalid params"}
		}

		jugglerParams := map[string]interface{}{}
		if params.URL != "" {
			jugglerParams["url"] = params.URL
		}
		if params.BrowserContextID != "" {
			jugglerParams["browserContextId"] = params.BrowserContextID
		}

		result, err := b.callJuggler("", "Browser.newPage", jugglerParams)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}

		// Juggler returns { pageId, sessionId }
		var pageResult struct {
			PageID    string `json:"pageId"`
			SessionID string `json:"sessionId"`
		}
		json.Unmarshal(result, &pageResult)

		targetID := pageResult.PageID
		if targetID == "" {
			targetID = uuid.New().String()
		}

		return marshalResult(map[string]string{"targetId": targetID})

	case "Target.closeTarget":
		var params struct {
			TargetID string `json:"targetId"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, &cdp.Error{Code: -32602, Message: "invalid params"}
		}

		info, ok := b.sessions.GetByTarget(params.TargetID)
		if !ok {
			return nil, &cdp.Error{Code: -32000, Message: fmt.Sprintf("target %s not found", params.TargetID)}
		}

		_, err := b.callJuggler(info.JugglerSessionID, "Page.close", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		b.sessions.Remove(info.SessionID)

		return json.RawMessage(`{"success":true}`), nil

	case "Target.createBrowserContext":
		result, err := b.callJuggler("", "Browser.createBrowserContext", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}

		var ctxResult struct {
			BrowserContextID string `json:"browserContextId"`
		}
		json.Unmarshal(result, &ctxResult)

		return marshalResult(map[string]string{"browserContextId": ctxResult.BrowserContextID})

	case "Target.disposeBrowserContext":
		var params struct {
			BrowserContextID string `json:"browserContextId"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, &cdp.Error{Code: -32602, Message: "invalid params"}
		}

		_, err := b.callJuggler("", "Browser.removeBrowserContext", map[string]string{
			"browserContextId": params.BrowserContextID,
		})
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return json.RawMessage(`{}`), nil

	case "Target.getTargets":
		targets := []map[string]interface{}{}
		for _, s := range b.sessions.All() {
			targets = append(targets, map[string]interface{}{
				"targetId":         s.TargetID,
				"type":             s.Type,
				"title":            s.Title,
				"url":              s.URL,
				"attached":         true,
				"browserContextId": s.BrowserContextID,
			})
		}
		return marshalResult(map[string]interface{}{"targetInfos": targets})

	case "Target.attachToTarget":
		var params struct {
			TargetID string `json:"targetId"`
			Flatten  bool   `json:"flatten"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, &cdp.Error{Code: -32602, Message: "invalid params"}
		}

		// Check if we already have a session for this target.
		if info, ok := b.sessions.GetByTarget(params.TargetID); ok {
			return marshalResult(map[string]string{"sessionId": info.SessionID})
		}

		// Create a new CDP session for this target.
		sessionID := uuid.New().String()
		b.sessions.Add(&cdp.SessionInfo{
			SessionID:        sessionID,
			JugglerSessionID: params.TargetID, // use targetID as juggler session
			TargetID:         params.TargetID,
			Type:             "page",
		})

		// Emit Target.attachedToTarget event.
		b.emitEvent("Target.attachedToTarget", map[string]interface{}{
			"sessionId": sessionID,
			"targetInfo": map[string]interface{}{
				"targetId": params.TargetID,
				"type":     "page",
				"title":    "",
				"url":      "",
				"attached": true,
			},
			"waitingForDebugger": false,
		}, "")

		return marshalResult(map[string]string{"sessionId": sessionID})

	case "Target.setAutoAttach":
		// Accepted but no-op; we handle attachment explicitly.
		return json.RawMessage(`{}`), nil

	case "Target.activateTarget":
		return json.RawMessage(`{}`), nil

	case "Target.getBrowserContexts":
		// Return list of browser context IDs
		contexts := b.sessions.GetBrowserContexts()
		return marshalResult(map[string]interface{}{"browserContextIds": contexts})

	case "Target.getTargetInfo":
		var params struct {
			TargetID string `json:"targetId"`
		}
		json.Unmarshal(msg.Params, &params)
		if info, ok := b.sessions.GetByTarget(params.TargetID); ok {
			return marshalResult(map[string]interface{}{
				"targetInfo": map[string]interface{}{
					"targetId":         info.TargetID,
					"type":             info.Type,
					"title":            info.Title,
					"url":              info.URL,
					"attached":         true,
					"browserContextId": info.BrowserContextID,
				},
			})
		}
		return nil, &cdp.Error{Code: -32000, Message: "target not found"}

	default:
		return nil, &cdp.Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", msg.Method)}
	}
}

func marshalResult(v interface{}) (json.RawMessage, *cdp.Error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, &cdp.Error{Code: -32000, Message: err.Error()}
	}
	return data, nil
}
