package socketserver

import (
	"net"
	"time"
)

type wireConn interface {
	ReadMessage() (Message, error)
	WriteMessage(Message, time.Time) error
	Close() error
	RemoteAddr() net.Addr
}
