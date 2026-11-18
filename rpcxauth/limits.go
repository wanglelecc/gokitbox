package rpcxauth

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"

	"github.com/smallnest/rpcx/protocol"
)

// The native SDK limit is process-wide. Install once during package initialization,
// never mutate it while clients or servers are running.
const MaxFrameBytes = 8 << 20

func init() {
	protocol.MaxMessageLength = MaxFrameBytes
	protocol.Compressors[protocol.Gzip] = boundedGzip{}
}

type boundedGzip struct{}

func (boundedGzip) Zip(data []byte) ([]byte, error) {
	if len(data) > MaxFrameBytes {
		return nil, errors.New("rpcx: payload exceeds limit")
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func (boundedGzip) Unzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	body, err := io.ReadAll(io.LimitReader(r, MaxFrameBytes+1))
	if len(body) > MaxFrameBytes {
		return nil, errors.New("rpcx: expanded payload exceeds limit")
	}
	return body, err
}
