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

	// Console.enable / Console.disable — always on in Juggler.
	if method == "Console.enable" || method == "Console.disable" {
		return json.RawMessage(`{}`), nil
	}

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

	return nil, &cdp.Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", method)}
}

// handleNetwork stubs or translates Network domain calls.
func (b *Bridge) handleNetwork(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "Network.enable", "Network.disable":
		return json.RawMessage(`{}`), nil
	case "Network.setCacheDisabled":
		return json.RawMessage(`{}`), nil
	case "Network.setExtraHTTPHeaders":
		return json.RawMessage(`{}`), nil
	case "Network.setUserAgentOverride":
		return json.RawMessage(`{}`), nil
	default:
		return json.RawMessage(`{}`), nil
	}
}

// handleEmulation stubs or translates Emulation domain calls.
func (b *Bridge) handleEmulation(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "Emulation.setDeviceMetricsOverride":
		return json.RawMessage(`{}`), nil
	case "Emulation.clearDeviceMetricsOverride":
		return json.RawMessage(`{}`), nil
	case "Emulation.setTouchEmulationEnabled":
		return json.RawMessage(`{}`), nil
	case "Emulation.setGeolocationOverride":
		return json.RawMessage(`{}`), nil
	case "Emulation.setLocaleOverride":
		return json.RawMessage(`{}`), nil
	case "Emulation.setTimezoneOverride":
		return json.RawMessage(`{}`), nil
	case "Emulation.setEmulatedMedia":
		return json.RawMessage(`{}`), nil
	case "Emulation.setScrollbarsHidden":
		return json.RawMessage(`{}`), nil
	default:
		return json.RawMessage(`{}`), nil
	}
}

// handleDOM stubs or translates DOM domain calls.
func (b *Bridge) handleDOM(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "DOM.enable", "DOM.disable":
		return json.RawMessage(`{}`), nil
	case "DOM.getDocument":
		return marshalResult(map[string]interface{}{
			"root": map[string]interface{}{
				"nodeId":       1,
				"backendNodeId": 1,
				"nodeType":     9,
				"nodeName":     "#document",
				"localName":    "",
				"nodeValue":    "",
				"childNodeCount": 0,
			},
		})
	default:
		return json.RawMessage(`{}`), nil
	}
}

// handleAccessibility stubs or translates Accessibility domain calls.
func (b *Bridge) handleAccessibility(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "Accessibility.enable", "Accessibility.disable":
		return json.RawMessage(`{}`), nil
	case "Accessibility.getFullAXTree":
		// Route to Juggler's accessibility tree.
		result, err := b.callJuggler(msg.SessionID, "Page.getFullAXTree", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return result, nil
	default:
		return json.RawMessage(`{}`), nil
	}
}
