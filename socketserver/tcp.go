package socketserver

import (
	"bufio"
	"errors"
	"net"
	"time"
)

type tcpConn struct {
	net.Conn
	reader *bufio.Reader
	codec  FrameCodec
}

func (c *tcpConn) ReadMessage() (Message, error) {
	data, err := c.codec.ReadFrame(c.reader)
	return Message{Type: BinaryMessage, Data: data}, err
}
func (c *tcpConn) WriteMessage(message Message, deadline time.Time) error {
	if err := c.Conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return c.codec.WriteFrame(c.Conn, message.Data)
}
func (s *Server) serveTCP(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		// Admission and route hooks run per session, never in the accept loop.
		var codec FrameCodec
		if err = invoke(func() error {
			codec = s.newCodec()
			if codec == nil {
				return errors.New("socketserver: codec factory returned nil")
			}
			return nil
		}); err == nil {
			err = s.admit(&tcpConn{Conn: conn, reader: bufio.NewReaderSize(conn, s.options.ReadBufferSize), codec: codec})
		}
		if err != nil {
			conn.Close()
			s.report(nil, "admit", err)
		}
	}
}

// Built-in codecs return uniquely owned payloads. Custom codecs retain the
// conservative copy contract, even when they reuse a scratch buffer.
func (c *tcpConn) ownsReadBuffers() bool {
	switch c.codec.(type) {
	case *rawCodec, *lineCodec, *lengthCodec:
		return true
	default:
		return false
	}
}
