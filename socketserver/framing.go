package socketserver

import (
	"bufio"
	"encoding/binary"
	"io"
)

// A fresh codec is constructed per TCP session. ReadFrame must preserve stream
// boundaries; a non-nil payload with an error is delivered before termination.
type FrameCodec interface {
	ReadFrame(*bufio.Reader) ([]byte, error)
	WriteFrame(io.Writer, []byte) error
}
type rawCodec struct{ size int }

func (c *rawCodec) ReadFrame(r *bufio.Reader) ([]byte, error) {
	data := make([]byte, c.size)
	n, err := r.Read(data)
	if n == 0 {
		return nil, err
	}
	return data[:n], err
}
func (c *rawCodec) WriteFrame(w io.Writer, data []byte) error { return writeAll(w, data) }

type lineCodec struct {
	delimiter byte
	limit     int64
}

func (c *lineCodec) ReadFrame(r *bufio.Reader) ([]byte, error) {
	var data []byte
	for {
		part, err := r.ReadSlice(c.delimiter)
		if int64(len(data))+int64(len(part)) > c.limit+1 {
			return nil, ErrMessageTooLarge
		}
		data = append(data, part...)
		if err == nil {
			return data[:len(data)-1], nil
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF && len(data) > 0 {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
}
func (c *lineCodec) WriteFrame(w io.Writer, data []byte) error {
	if err := writeAll(w, data); err != nil {
		return err
	}
	return writeAll(w, []byte{c.delimiter})
}

type lengthCodec struct{ limit int64 }

func (c *lengthCodec) ReadFrame(r *bufio.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if int64(size) > c.limit {
		return nil, ErrMessageTooLarge
	}
	data := make([]byte, int(size))
	_, err := io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (c *lengthCodec) WriteFrame(w io.Writer, data []byte) error {
	if uint64(len(data)) > uint64(^uint32(0)) {
		return ErrMessageTooLarge
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, data)
}
func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if n < 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
func (s *Server) newCodec() FrameCodec {
	if s.settings.codec != nil {
		return s.settings.codec()
	}
	switch s.options.TCPFraming {
	case "line":
		return &lineCodec{delimiter: s.options.LineDelimiter[0], limit: s.options.MaxMessageBytes}
	case "length-prefix":
		return &lengthCodec{limit: s.options.MaxMessageBytes}
	default:
		size := s.options.ReadBufferSize
		if s.options.MaxMessageBytes > 0 && int64(size) > s.options.MaxMessageBytes {
			size = int(s.options.MaxMessageBytes)
		}
		return &rawCodec{size: size}
	}
}
