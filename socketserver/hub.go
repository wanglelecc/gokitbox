package socketserver

import (
	"context"
	"sync"
)

// Hub owns only active sessions. It never runs network I/O or hooks under its lock.
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func newHub() *Hub { return &Hub{sessions: map[string]*Session{}} }
func (h *Hub) add(s *Session) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions[s.id] != nil {
		return ErrDuplicateID
	}
	h.sessions[s.id] = s
	return nil
}
func (h *Hub) remove(s *Session) {
	h.mu.Lock()
	if h.sessions[s.id] == s {
		delete(h.sessions, s.id)
	}
	h.mu.Unlock()
}
func (h *Hub) Get(id string) (*Session, error) {
	h.mu.RLock()
	s := h.sessions[id]
	h.mu.RUnlock()
	if s == nil || s.Closed() {
		return nil, ErrSessionNotFound
	}
	return s, nil
}
func (h *Hub) Count() int { h.mu.RLock(); defer h.mu.RUnlock(); return len(h.sessions) }
func (h *Hub) HasOnline(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s, err := h.Get(id)
	if err != nil {
		return "", err
	}
	return s.ID(), nil
}
func (h *Hub) WriteMessage(ctx context.Context, id string, data []byte) error {
	s, err := h.Get(id)
	if err != nil {
		return err
	}
	return s.Send(ctx, Message{Type: TextMessage, Data: data})
}
func (h *Hub) Login(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, err := h.Get(id)
	if err != nil {
		return err
	}
	return s.MarkAuthenticated()
}
func (h *Hub) Disconnect(ctx context.Context, id string, reason CloseReason) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, err := h.Get(id)
	if err != nil {
		return err
	}
	s.Close(reason)
	return nil
}
