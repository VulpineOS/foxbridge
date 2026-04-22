package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
	"github.com/gorilla/websocket"
)

type ReplayServer struct {
	host     string
	port     int
	socket   string
	entries  []Entry
	upgrader websocket.Upgrader
}

func NewReplayServer(port int, entries []Entry) *ReplayServer {
	return &ReplayServer{
		host:    "127.0.0.1",
		port:    port,
		entries: append([]Entry(nil), entries...),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *ReplayServer) SetUnixSocket(path string) {
	s.socket = path
}

func (s *ReplayServer) ListenDescription() string {
	if s.socket != "" {
		return "unix://" + s.socket
	}
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *ReplayServer) BrowserWSURL() string {
	if s.socket != "" {
		return "ws://localhost/devtools/browser/foxbridge"
	}
	return fmt.Sprintf("ws://%s:%d/devtools/browser/foxbridge", s.host, s.port)
}

func (s *ReplayServer) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}
	if s.socket != "" {
		defer os.Remove(s.socket)
	}
	log.Printf("foxbridge replay server listening on %s", s.ListenDescription())
	return http.Serve(ln, s.mux())
}

func (s *ReplayServer) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", s.handleVersion)
	mux.HandleFunc("/json", s.handleList)
	mux.HandleFunc("/json/list", s.handleList)
	mux.HandleFunc("/devtools/browser/", s.handleWS)
	mux.HandleFunc("/", s.handleWS)
	return mux
}

func (s *ReplayServer) listen() (net.Listener, error) {
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

func (s *ReplayServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"Browser":              "foxbridge-replay/1.0",
		"Protocol-Version":     "1.3",
		"User-Agent":           "foxbridge-replay",
		"webSocketDebuggerUrl": s.BrowserWSURL(),
	}
	if s.socket != "" {
		info["socketPath"] = s.socket
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func (s *ReplayServer) handleList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte("[]"))
}

func (s *ReplayServer) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("replay websocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	session := &replaySession{
		conn:    ws,
		entries: append([]Entry(nil), s.entries...),
	}

	if err := session.flushOutboundUntilInput(); err != nil {
		log.Printf("replay initial flush error: %v", err)
		return
	}

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			return
		}

		var msg cdp.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("replay invalid inbound message: %v", err)
			return
		}

		if err := session.acceptInbound(&msg); err != nil {
			log.Printf("replay inbound mismatch: %v", err)
			return
		}
		if err := session.flushOutboundUntilInput(); err != nil {
			log.Printf("replay outbound flush error: %v", err)
			return
		}
	}
}

type replaySession struct {
	mu      sync.Mutex
	conn    *websocket.Conn
	entries []Entry
	pos     int
}

func (s *replaySession) flushOutboundUntilInput() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.pos < len(s.entries) {
		entry := s.entries[s.pos]
		if entry.Direction != "out" {
			return nil
		}
		s.pos++

		data, err := json.Marshal(entry.Message)
		if err != nil {
			return fmt.Errorf("marshal replay outbound message: %w", err)
		}
		if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return fmt.Errorf("write replay outbound message: %w", err)
		}
	}

	return nil
}

func (s *replaySession) acceptInbound(msg *cdp.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pos >= len(s.entries) {
		return fmt.Errorf("received unexpected inbound message %s after replay exhausted", msg.Method)
	}

	entry := s.entries[s.pos]
	if entry.Direction != "in" {
		return fmt.Errorf("expected outbound replay frame, got inbound %s", msg.Method)
	}
	if !messagesEquivalent(entry.Message, msg) {
		return fmt.Errorf("expected inbound #%d %s, got #%d %s", entry.Message.ID, entry.Message.Method, msg.ID, msg.Method)
	}
	s.pos++
	return nil
}

func messagesEquivalent(a, b *cdp.Message) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
