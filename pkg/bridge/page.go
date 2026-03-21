package bridge

import (
	"encoding/json"
	"fmt"

	"github.com/PopcornDev1/foxbridge/pkg/cdp"
)

func (b *Bridge) handlePage(conn *cdp.Connection, msg *cdp.Message) (json.RawMessage, *cdp.Error) {
	switch msg.Method {
	case "Page.enable", "Page.setLifecycleEventsEnabled":
		// No-op — Juggler always emits lifecycle events.
		return json.RawMessage(`{}`), nil

	case "Page.navigate":
		var params struct {
			URL            string `json:"url"`
			Referrer       string `json:"referrer"`
			TransitionType string `json:"transitionType"`
			FrameID        string `json:"frameId"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, &cdp.Error{Code: -32602, Message: "invalid params"}
		}

		jugglerParams := map[string]interface{}{
			"url": params.URL,
		}
		if params.Referrer != "" {
			jugglerParams["referer"] = params.Referrer
		}
		if params.FrameID != "" {
			jugglerParams["frameId"] = params.FrameID
		}

		result, err := b.callJuggler(msg.SessionID, "Page.navigate", jugglerParams)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}

		// Juggler returns { navigationId, frameId }. CDP expects { frameId, loaderId }.
		var navResult struct {
			NavigationID string `json:"navigationId"`
			FrameID      string `json:"frameId"`
		}
		json.Unmarshal(result, &navResult)

		return marshalResult(map[string]interface{}{
			"frameId":  navResult.FrameID,
			"loaderId": navResult.NavigationID,
		})

	case "Page.reload":
		_, err := b.callJuggler(msg.SessionID, "Page.reload", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return json.RawMessage(`{}`), nil

	case "Page.close":
		_, err := b.callJuggler(msg.SessionID, "Page.close", nil)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}
		return json.RawMessage(`{}`), nil

	case "Page.captureScreenshot":
		var params struct {
			Format      string `json:"format"`
			Quality     int    `json:"quality"`
			Clip        *struct {
				X      float64 `json:"x"`
				Y      float64 `json:"y"`
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
				Scale  float64 `json:"scale"`
			} `json:"clip"`
			FromSurface bool `json:"fromSurface"`
		}
		if msg.Params != nil {
			json.Unmarshal(msg.Params, &params)
		}

		jugglerParams := map[string]interface{}{}
		if params.Format != "" {
			mimeType := "image/png"
			if params.Format == "jpeg" || params.Format == "jpg" {
				mimeType = "image/jpeg"
			}
			jugglerParams["mimeType"] = mimeType
		}
		if params.Clip != nil {
			jugglerParams["clip"] = map[string]interface{}{
				"x":      params.Clip.X,
				"y":      params.Clip.Y,
				"width":  params.Clip.Width,
				"height": params.Clip.Height,
			}
		}

		result, err := b.callJuggler(msg.SessionID, "Page.screenshot", jugglerParams)
		if err != nil {
			return nil, &cdp.Error{Code: -32000, Message: err.Error()}
		}

		// Juggler returns { data }. CDP expects { data }.
		var ssResult struct {
			Data string `json:"data"`
		}
		json.Unmarshal(result, &ssResult)

		return marshalResult(map[string]string{"data": ssResult.Data})

	case "Page.getFrameTree":
		// Return a minimal frame tree.
		return marshalResult(map[string]interface{}{
			"frameTree": map[string]interface{}{
				"frame": map[string]interface{}{
					"id":             "main",
					"loaderId":       "",
					"url":            "",
					"securityOrigin": "",
					"mimeType":       "text/html",
				},
				"childFrames": []interface{}{},
			},
		})

	case "Page.setInterceptFileChooserDialog":
		return json.RawMessage(`{}`), nil

	default:
		return nil, &cdp.Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", msg.Method)}
	}
}
