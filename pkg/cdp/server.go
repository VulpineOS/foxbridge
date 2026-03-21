package cdp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageHandler is called for each incoming CDP message.
type MessageHandler func(conn *Connection, msg *Message)

// Connection represents a single CDP WebSocket connection.
type Connection struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
}

// Send sends a CDP message to the client.
func (c *Connection) Send(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal CDP message: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

// Server is the CDP WebSocket server.
type Server struct {
	handler  MessageHandler
	upgrader websocket.Upgrader
	port     int
	conns    map[*Connection]struct{}
	connsMu  sync.Mutex
}

// NewServer creates a CDP server on the given port.
func NewServer(port int, handler MessageHandler) *Server {
	return &Server{
		handler: handler,
		port:    port,
		conns:   make(map[*Connection]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Start begins listening for WebSocket connections.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// CDP discovery endpoint
	mux.HandleFunc("/json/version", s.handleVersion)
	mux.HandleFunc("/json", s.handleList)
	mux.HandleFunc("/json/list", s.handleList)
	mux.HandleFunc("/devtools/browser/", s.handleWS)
	// Also accept root path for convenience
	mux.HandleFunc("/", s.handleWS)

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	log.Printf("foxbridge CDP server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
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
		"webSocketDebuggerUrl": fmt.Sprintf("ws://127.0.0.1:%d/devtools/browser/foxbridge", s.port),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	conn := &Connection{ws: ws}
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

		s.handler(conn, &msg)
	}
}
