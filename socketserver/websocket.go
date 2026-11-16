package socketserver

import (
	"net"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

type websocketConn struct{ conn *ws.Conn }

func (c *websocketConn) ReadMessage() (Message, error) {
	kind, data, err := c.conn.ReadMessage()
	return Message{Type: MessageType(kind), Data: data}, err
}
func (c *websocketConn) WriteMessage(message Message, deadline time.Time) error {
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return c.conn.WriteMessage(int(message.Type), message.Data)
}
func (c *websocketConn) Close() error         { return c.conn.Close() }
func (c *websocketConn) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := ws.Upgrader{ReadBufferSize: s.options.ReadBufferSize, WriteBufferSize: s.options.ReadBufferSize, HandshakeTimeout: s.options.HookTimeout}
	if s.options.AllowAllOrigins {
		upgrader.CheckOrigin = func(*http.Request) bool { return true }
	} else if len(s.options.AllowedOrigins) > 0 {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, allowed := range s.options.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.report(nil, "upgrade", err)
		return
	}
	if s.options.MaxMessageBytes > 0 {
		conn.SetReadLimit(s.options.MaxMessageBytes)
	}
	if err = s.admit(&websocketConn{conn: conn}); err != nil {
		conn.Close()
		s.report(nil, "admit", err)
	}
}

func (c *websocketConn) ownsReadBuffers() bool { return true }
