package socketserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
)

var ErrSessionNotFound = errors.New("socketserver: session not found")
var ErrClosed = errors.New("socketserver: session closed")
var ErrQueueFull = errors.New("socketserver: write queue full")
var ErrReadBudgetExceeded = errors.New("socketserver: server read byte budget exceeded")
var ErrMessageTooLarge = errors.New("socketserver: message or queue byte limit exceeded")
var ErrTooManyConnections = errors.New("socketserver: connection limit reached")
var ErrDuplicateID = errors.New("socketserver: duplicate session ID")

type MessageType uint8

const (
	TextMessage   MessageType = 1
	BinaryMessage MessageType = 2
)

type Message struct {
	Type MessageType
	Data []byte
}
type CloseReason struct {
	Code string
	Err  error
}
type ErrorEvent struct {
	Session   *Session
	Operation string
	Err       error
}
type Hooks struct {
	OnConnect    func(context.Context, *Session) error
	OnMessage    func(context.Context, *Session, Message) error
	OnDisconnect func(context.Context, *Session, CloseReason) error
	OnMaintain   func(context.Context, *Session) error
	OnError      func(context.Context, ErrorEvent)
}
type settings struct {
	routes func(*http.ServeMux)
	hooks  Hooks
	id     func() (string, error)
	codec  func() FrameCodec
}
type Option func(*settings)

func WithHooks(h Hooks) Option { return func(s *settings) { s.hooks = h } }
func WithHTTPRoutes(routes func(*http.ServeMux)) Option {
	return func(s *settings) { s.routes = routes }
}
func WithIDGenerator(id func() (string, error)) Option { return func(s *settings) { s.id = id } }
func WithCodecFactory(factory func() FrameCodec) Option {
	return func(s *settings) { s.codec = factory }
}
func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

type hookPanic struct{ kind string }

func (e *hookPanic) Error() string { return "socketserver: hook panic (" + e.kind + ")" }
func invoke(fn func() error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = &hookPanic{kind: fmt.Sprintf("%T", p)}
		}
	}()
	return fn()
}
