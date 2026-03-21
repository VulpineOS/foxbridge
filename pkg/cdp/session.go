package cdp

import "sync"

// SessionInfo tracks a CDP session mapped to a Juggler session.
type SessionInfo struct {
	SessionID        string // CDP session ID
	JugglerSessionID string // Juggler session ID (may be same or mapped)
	TargetID         string // Target (page) ID
	BrowserContextID string
	URL              string
	Title            string
	Type             string // "page", "background_page", etc.
}

// SessionManager tracks CDP sessions and their mappings to Juggler sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionInfo // keyed by CDP sessionID
	targets  map[string]*SessionInfo // keyed by targetID
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionInfo),
		targets:  make(map[string]*SessionInfo),
	}
}

// Add registers a new session.
func (sm *SessionManager) Add(info *SessionInfo) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[info.SessionID] = info
	if info.TargetID != "" {
		sm.targets[info.TargetID] = info
	}
}

// Remove deletes a session by CDP session ID.
func (sm *SessionManager) Remove(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if info, ok := sm.sessions[sessionID]; ok {
		delete(sm.targets, info.TargetID)
		delete(sm.sessions, sessionID)
	}
}

// Get returns session info by CDP session ID.
func (sm *SessionManager) Get(sessionID string) (*SessionInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	info, ok := sm.sessions[sessionID]
	return info, ok
}

// GetByTarget returns session info by target ID.
func (sm *SessionManager) GetByTarget(targetID string) (*SessionInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	info, ok := sm.targets[targetID]
	return info, ok
}

// All returns all sessions.
func (sm *SessionManager) All() []*SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]*SessionInfo, 0, len(sm.sessions))
	for _, info := range sm.sessions {
		result = append(result, info)
	}
	return result
}
