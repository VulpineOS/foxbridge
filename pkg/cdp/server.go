package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageHandler is called for each incoming CDP message.
type MessageHandler func(conn *Connection, msg *Message)

// FrameRecorder captures inbound and outbound CDP wire messages.
type FrameRecorder interface {
	Record(direction string, msg *Message) error
}

// Connection represents a single CDP WebSocket connection.
type Connection struct {
	ws       *websocket.Conn
	writeMu  sync.Mutex
	recorder FrameRecorder
}

// Send sends a CDP message to the client.
func (c *Connection) Send(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal CDP message: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}
	if c.recorder != nil {
		if err := c.recorder.Record("out", msg); err != nil {
			log.Printf("[record] outbound record error: %v", err)
		}
	}
	return nil
}

// Server is the CDP WebSocket server.
type Server struct {
	handler  MessageHandler
	upgrader websocket.Upgrader
	host     string
	port     int
	socket   string
	recorder FrameRecorder
	conns    map[*Connection]struct{}
	connsMu  sync.Mutex
	sessions *SessionManager
	serverMu sync.Mutex
	server   *http.Server
}

// NewServer creates a CDP server on the given port.
func NewServer(port int, handler MessageHandler, sessions *SessionManager) *Server {
	return &Server{
		handler:  handler,
		host:     "127.0.0.1",
		port:     port,
		sessions: sessions,
		conns:    make(map[*Connection]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// SetRecorder configures an optional wire recorder for inbound and outbound CDP frames.
func (s *Server) SetRecorder(recorder FrameRecorder) {
	s.recorder = recorder
}

// SetHost overrides the TCP host used when serving over a network port.
func (s *Server) SetHost(host string) {
	if host == "" {
		host = "127.0.0.1"
	}
	s.host = host
}

// SetUnixSocket configures the server to listen on a Unix domain socket.
func (s *Server) SetUnixSocket(path string) {
	s.socket = path
}

// ListenDescription returns the active listener target for logs and diagnostics.
func (s *Server) ListenDescription() string {
	if s.socket != "" {
		return "unix://" + s.socket
	}
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// BrowserWSURL returns the browser-level WebSocket URL exposed by discovery endpoints.
func (s *Server) BrowserWSURL() string {
	return s.discoveryBaseURL() + "/devtools/browser/foxbridge"
}

func (s *Server) targetWSURL(targetID string) string {
	return s.discoveryBaseURL() + "/devtools/page/" + targetID
}

func (s *Server) discoveryBaseURL() string {
	if s.socket != "" {
		return "ws://localhost"
	}
	return fmt.Sprintf("ws://%s:%d", s.host, s.port)
}

func (s *Server) listen() (net.Listener, error) {
	if s.socket != "" {
		if err := os.Remove(s.socket); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("remove stale unix socket: %w", err)
		}
		ln, err := net.Listen("unix", s.socket)
		if err != nil {
			return nil, fmt.Errorf("listen unix %s: %w", s.socket, err)
		}
		if err := os.Chmod(s.socket, 0o600); err != nil {
			ln.Close()
			return nil, fmt.Errorf("chmod unix socket: %w", err)
		}
		return ln, nil
	}
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen tcp %s: %w", addr, err)
	}
	return ln, nil
}

func (s *Server) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", s.handleVersion)
	mux.HandleFunc("/json/version/", s.handleVersion)
	mux.HandleFunc("/json", s.handleList)
	mux.HandleFunc("/json/list", s.handleList)
	mux.HandleFunc("/devtools/browser/", s.handleWS)
	mux.HandleFunc("/", s.handleWS)
	return mux
}

// Start begins listening for WebSocket connections.
func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}
	if s.socket != "" {
		defer os.Remove(s.socket)
	}
	srv := &http.Server{Handler: s.mux()}
	s.serverMu.Lock()
	if s.server != nil {
		s.serverMu.Unlock()
		ln.Close()
		return fmt.Errorf("server already running")
	}
	s.server = srv
	s.serverMu.Unlock()

	log.Printf("foxbridge CDP server listening on %s", s.ListenDescription())
	err = srv.Serve(ln)

	s.serverMu.Lock()
	if s.server == srv {
		s.server = nil
	}
	s.serverMu.Unlock()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the CDP HTTP server and closes any active WebSocket clients.
func (s *Server) Shutdown(ctx context.Context) error {
	s.serverMu.Lock()
	srv := s.server
	s.serverMu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Shutdown(ctx)
	s.closeConnections()
	return err
}

// Close immediately stops the CDP HTTP server and closes any active WebSocket clients.
func (s *Server) Close() error {
	s.serverMu.Lock()
	srv := s.server
	s.serverMu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Close()
	s.closeConnections()
	return err
}

func (s *Server) closeConnections() {
	s.connsMu.Lock()
	conns := make([]*Connection, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
		delete(s.conns, c)
	}
	s.connsMu.Unlock()

	for _, c := range conns {
		_ = c.ws.Close()
	}
}

// Broadcast sends a CDP message to all connected clients.
func (s *Server) Broadcast(msg *Message) {
	s.connsMu.Lock()
	conns := make([]*Connection, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.connsMu.Unlock()

	data, _ := json.Marshal(msg)
	log.Printf("[broadcast] %s to %d clients: %s", msg.Method, len(conns), string(data)[:min(len(data), 300)])
	for _, c := range conns {
		if err := c.Send(msg); err != nil {
			log.Printf("[broadcast] send error: %v", err)
		}
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"Browser":              "foxbridge/1.0",
		"Protocol-Version":     "1.3",
		"User-Agent":           "foxbridge",
		"webSocketDebuggerUrl": s.BrowserWSURL(),
	}
	if s.socket != "" {
		info["socketPath"] = s.socket
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.sessions == nil {
		w.Write([]byte("[]"))
		return
	}
	var targets []map[string]interface{}
	for _, info := range s.sessions.All() {
		if info.Type != "page" {
			continue
		}
		url := info.URL
		if url == "" {
			url = "about:blank"
		}
		targets = append(targets, map[string]interface{}{
			"id":                   info.TargetID,
			"type":                 "page",
			"title":                info.Title,
			"url":                  url,
			"devtoolsFrontendUrl":  "",
			"webSocketDebuggerUrl": s.targetWSURL(info.TargetID),
		})
		if s.socket != "" {
			targets[len(targets)-1]["socketPath"] = s.socket
		}
	}
	if targets == nil {
		targets = []map[string]interface{}{}
	}
	json.NewEncoder(w).Encode(targets)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	conn := &Connection{ws: ws, recorder: s.recorder}
	s.connsMu.Lock()
	s.conns[conn] = struct{}{}
	s.connsMu.Unlock()

	defer func() {
		s.connsMu.Lock()
		delete(s.conns, conn)
		s.connsMu.Unlock()
		ws.Close()
	}()

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("invalid CDP message: %v", err)
			continue
		}
		if s.recorder != nil {
			if err := s.recorder.Record("in", &msg); err != nil {
				log.Printf("[record] inbound record error: %v", err)
			}
		}

		log.Printf("[cdp-in] #%d %s (session=%s)", msg.ID, msg.Method, msg.SessionID)
		s.handler(conn, &msg)
	}
}
