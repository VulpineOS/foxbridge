package bridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PopcornDev1/foxbridge/pkg/cdp"
)

// stubDomains are domains that return success no-ops.
var stubDomains = map[string]bool{
	"Debugger":      true,
	"Profiler":      true,
	"Performance":   true,
	"HeapProfiler":  true,
	"Memory":        true,
	"ServiceWorker": true,
	"CacheStorage":  true,
	"IndexedDB":     true,
	"Log":           true,
	"Security":      true,
	"Fetch":         true,
	"CSS":           true,
	"Overlay":       true,
	"DOMStorage":    true,
}

func (b *Bridge) handleStub(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	method := msg.Method

	// Browser.getVersion → Browser.getInfo
	if method == "Browser.getVersion" {
		result, err := b.callJuggler("", "Browser.getInfo", nil)
		if err != nil {
			// Fallback with static info.
			return marshalResult(map[string]string{
				"protocolVersion": "1.3",
				"product":         "foxbridge (Firefox/Camoufox)",
				"revision":        "",
				"userAgent":       "",
				"jsVersion":       "",
			})
		}
		return result, nil
	}

	// Browser.close
	if method == "Browser.close" {
		_, err := b.callJuggler("", "Browser.close", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return json.RawMessage(`{}`), nil
	}

	// Check if the domain is a known stub domain.
	parts := strings.SplitN(method, ".", 2)
	if len(parts) == 2 && stubDomains[parts[0]] {
		return json.RawMessage(`{}`), nil
	}

	// .enable / .disable methods are generally safe to no-op.
	if strings.HasSuffix(method, ".enable") || strings.HasSuffix(method, ".disable") {
		return json.RawMessage(`{}`), nil
	}

	// Specific method stubs needed for Puppeteer compatibility.
	switch method {
	case "Runtime.runIfWaitingForDebugger":
		return json.RawMessage(`{}`), nil
	case "Page.getFrameTree":
		// Return the frame tree with the correct frame ID from session state
		frameID := "main"
		return marshalResult(map[string]interface{}{
			"frameTree": map[string]interface{}{
				"frame": map[string]interface{}{
					"id":              frameID,
					"loaderId":        "",
					"url":             "about:blank",
					"domainAndRegistry": "",
					"securityOrigin":  "",
					"mimeType":        "text/html",
					"secureContextType": "InsecureScheme",
					"crossOriginIsolatedContextType": "NotIsolated",
					"gatedAPIFeatures": []string{},
				},
			},
		})
	case "Page.setLifecycleEventsEnabled":
		return json.RawMessage(`{}`), nil
	case "Page.addScriptToEvaluateOnNewDocument":
		return marshalResult(map[string]string{"identifier": "1"})
	case "Page.createIsolatedWorld":
		var params struct {
			FrameID              string `json:"frameId"`
			WorldName            string `json:"worldName"`
			GrantUniversalAccess bool   `json:"grantUniveralAccess"`
		}
		json.Unmarshal(msg.Params, &params)

		// Generate a unique context ID for the isolated world
		// Use a high number to avoid collision with main world contexts
		b.ctxMapMu.Lock()
		ctxID := 10000 + len(b.ctxMap)
		// Map to the MAIN world's Juggler context — isolated worlds in Juggler
		// just use the same execution context, so we find the latest one
		latestJugglerCtx := ""
		for _, v := range b.ctxMap {
			latestJugglerCtx = v
		}
		if latestJugglerCtx != "" {
			b.ctxMap[ctxID] = latestJugglerCtx
		}
		b.ctxMapMu.Unlock()

		uniqueID := fmt.Sprintf("isolated-%s-%s", params.FrameID, params.WorldName)

		// Store the mapping for the isolated world too
		if latestJugglerCtx != "" {
			b.ctxMapMu.Lock()
			b.ctxMap[ctxID] = latestJugglerCtx
			b.ctxMapMu.Unlock()
		}

		sessionID := msg.SessionID

		// Emit the context event AFTER returning the response
		go func() {
			b.emitEvent("Runtime.executionContextCreated", map[string]interface{}{
				"context": map[string]interface{}{
					"id":       ctxID,
					"origin":   "",
					"name":     params.WorldName,
					"uniqueId": uniqueID,
					"auxData": map[string]interface{}{
						"isDefault": false,
						"type":      "isolated",
						"frameId":   params.FrameID,
					},
				},
			}, sessionID)
		}()

		return marshalResult(map[string]interface{}{"executionContextId": ctxID})
	case "Page.setInterceptFileChooserDialog":
		return json.RawMessage(`{}`), nil
	case "Emulation.setDefaultBackgroundColorOverride":
		return json.RawMessage(`{}`), nil
	}

	return nil, &cdp.Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", method)}
}
