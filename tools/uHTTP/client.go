package uHTTP

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrResponseTooLarge = errors.New("http client: response exceeds byte limit")
var ErrUploadTooLarge = errors.New("http client: upload exceeds byte limit")
var ErrClosed = errors.New("http client: closed")

type Client struct {
	closeOnce sync.Once
	closeDone chan struct{}
	native    *http.Client
	transport *http.Transport
	options   Config
	mu        sync.Mutex
	closed    bool
	wg        sync.WaitGroup
	root      context.Context
	cancel    context.CancelFunc
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.DialTimeout <= 0 || cfg.Timeout <= 0 || cfg.TLSHandshakeTimeout <= 0 || cfg.ResponseHeaderTimeout <= 0 || cfg.MaxResponseBytes < 0 || cfg.MaxResponseBytes > 1<<30 || cfg.MaxUploadBytes < 0 || cfg.MaxUploadBytes > 1<<30 || cfg.DialTimeout > 86400 || cfg.Timeout > 86400 || cfg.TLSHandshakeTimeout > 86400 || cfg.ResponseHeaderTimeout > 86400 || cfg.DialKeepAlive < 0 || cfg.DialKeepAlive > 86400 || cfg.ExpectContinueTimeout < 0 || cfg.ExpectContinueTimeout > 86400 {
		return nil, errors.New("http client: positive timeouts and nonnegative byte limits required")
	}
	if cfg.MaxResponseBytes == 0 {
		cfg.MaxResponseBytes = 8 << 20
	}
	if cfg.MaxUploadBytes == 0 {
		cfg.MaxUploadBytes = 64 << 20
	}
	t := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: time.Duration(cfg.DialTimeout) * time.Second, KeepAlive: time.Duration(cfg.DialKeepAlive) * time.Second}).DialContext, TLSHandshakeTimeout: time.Duration(cfg.TLSHandshakeTimeout) * time.Second, ResponseHeaderTimeout: time.Duration(cfg.ResponseHeaderTimeout) * time.Second, ExpectContinueTimeout: time.Duration(cfg.ExpectContinueTimeout) * time.Second, IdleConnTimeout: 90 * time.Second, MaxIdleConns: 100, MaxIdleConnsPerHost: 10, MaxConnsPerHost: 100, ForceAttemptHTTP2: true}
	root, cancel := context.WithCancel(context.Background())
	return &Client{native: &http.Client{Transport: t, Timeout: time.Duration(cfg.Timeout) * time.Second}, transport: t, options: cfg, root: root, cancel: cancel, closeDone: make(chan struct{})}, nil
}

var defaultClient = func() *Client {
	c, err := NewClient(DefaultConfig())
	if err != nil {
		panic(err)
	}
	return c
}()

func clientFor(cfg Config) (*Client, bool, error) {
	if cfg == DefaultConfig() {
		return defaultClient, false, nil
	}
	c, err := NewClient(cfg)
	return c, true, err
}
func (c *Client) Do(ctx context.Context, method, url string, headers map[string]string, body io.Reader) (int, http.Header, []byte, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, nil, nil, ErrClosed
	}
	c.wg.Add(1)
	c.mu.Unlock()
	defer c.wg.Done()
	call, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.root, cancel)
	defer func() { stop(); cancel() }()
	req, err := http.NewRequestWithContext(call, method, url, body)
	if err != nil {
		return 0, nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.native.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.options.MaxResponseBytes+1))
	if int64(len(data)) > c.options.MaxResponseBytes {
		return resp.StatusCode, resp.Header, nil, ErrResponseTooLarge
	}
	return resp.StatusCode, resp.Header, data, err
}
func (c *Client) Upload(ctx context.Context, url string, fields, files map[string]string, headers map[string]string) (int, http.Header, []byte, error) {
	type fileEntry struct {
		name string
		file *os.File
		size int64
	}
	entries := []fileEntry{}
	defer func() {
		for _, entry := range entries {
			entry.file.Close()
		}
	}()
	var total int64
	for name, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return 0, nil, nil, err
		}
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			f.Close()
			return 0, nil, nil, errors.New("http client: upload requires regular files")
		}
		entries = append(entries, fileEntry{name, f, info.Size()})
		if info.Size() > c.options.MaxUploadBytes-total {
			return 0, nil, nil, ErrUploadTooLarge
		}
		total += info.Size()
	}
	for _, v := range fields {
		if int64(len(v)) > c.options.MaxUploadBytes-total {
			return 0, nil, nil, ErrUploadTooLarge
		}
		total += int64(len(v))
	}
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	done := make(chan error, 1)
	go func() {
		var failure error
		defer func() { writer.CloseWithError(failure); done <- failure }()
		for _, entry := range entries {
			part, err := form.CreateFormFile(entry.name, filepath.Base(entry.file.Name()))
			if err != nil {
				failure = err
				return
			}
			n, err := io.Copy(part, io.LimitReader(entry.file, entry.size+1))
			if err != nil {
				failure = err
				return
			}
			if n > entry.size {
				failure = ErrUploadTooLarge
				return
			}
		}
		for key, value := range fields {
			if err := form.WriteField(key, value); err != nil {
				failure = err
				return
			}
		}
		failure = form.Close()
	}()
	copied := map[string]string{}
	for k, v := range headers {
		copied[k] = v
	}
	copied["Content-Type"] = form.FormDataContentType()
	code, h, b, err := c.Do(ctx, http.MethodPost, url, copied, reader)
	reader.CloseWithError(errors.New("http upload completed"))
	uploadErr := <-done
	if err == nil && uploadErr != nil {
		err = fmt.Errorf("http upload: %w", uploadErr)
	}
	return code, h, b, err
}
func (c *Client) CloseContext(ctx context.Context) error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		c.mu.Unlock()
		c.transport.CloseIdleConnections()
		go func() { c.wg.Wait(); c.transport.CloseIdleConnections(); close(c.closeDone) }()
	})
	select {
	case <-c.closeDone:
		return nil
	default:
	}
	select {
	case <-c.closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *Client) Close() error { return c.CloseContext(context.Background()) }
