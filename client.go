// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

// Package msqp implements the Steam Master Server Query Protocol.
package msqp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// maxUDPDatagramSize is the largest IPv4 UDP payload accepted by the API.
	maxUDPDatagramSize = 65507
	// defaultMaxDatagramSize allows any protocol-valid response unless overridden.
	defaultMaxDatagramSize = maxUDPDatagramSize
	// defaultIOTimeout bounds a query that has no earlier context deadline.
	defaultIOTimeout = time.Second
)

// Result is one item emitted by [Client.Stream].
// Exactly one of Err or Page is meaningful:
// Err reports a failed stream, while Page contains a successful page.
type Result struct {
	// Err is non-nil when the stream stopped because of an error.
	Err error
	// Page contains the successfully queried server page when Err is nil.
	Page Response
}

// Client queries a legacy Steam master server over UDP.
type Client struct {
	// conn is connected to the single master endpoint queried by this client.
	conn *net.UDPConn
	// readBuffer has one extra byte so Query can detect responses over the limit.
	readBuffer []byte
	// maxDatagramSize is the largest response accepted by Query.
	maxDatagramSize int
	// ioTimeout bounds each socket operation when the context has no earlier deadline.
	ioTimeout time.Duration
	// queryMu serializes request/response pairs and protects the reusable read buffer.
	queryMu sync.Mutex
	// stateMu protects closed while Query and Close run concurrently.
	stateMu sync.Mutex
	// closed lets later Query calls detect that Close has started.
	closed bool
}

// NewClient creates a client connected to address,
// which must be a UDP endpoint such as "master.example.net:27011".
func NewClient(address string, opts ...Option) (*Client, error) {
	config := clientConfig{
		maxDatagramSize: defaultMaxDatagramSize,
		ioTimeout:       defaultIOTimeout,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&config); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrConfigureMasterClient, err)
		}
	}

	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrResolveMasterAddress, address, err)
	}

	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrDialMasterAddress, address, err)
	}

	return &Client{
		conn:            conn,
		maxDatagramSize: config.maxDatagramSize,
		readBuffer:      make([]byte, config.maxDatagramSize+1),
		ioTimeout:       config.ioTimeout,
	}, nil
}

// Query retrieves one page from the configured master server.
func (c *Client) Query(ctx context.Context, req Request) (Response, error) {
	if c == nil || c.conn == nil {
		return Response{}, ErrClientUninitialized
	}
	if ctx == nil {
		return Response{}, ErrNilQueryContext
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	if !req.Cursor.Addr.IsValid() {
		req.Cursor = InitialCursor()
	}

	packet, err := EncodeRequest(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrEncodeMasterRequest, err)
	}

	// MSQP has no request ID, so concurrent reads could consume each other's pages.
	c.queryMu.Lock()
	defer c.queryMu.Unlock()
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	if err := c.isClosed(); err != nil {
		return Response{}, err
	}

	if ctx.Done() != nil {
		// A canceled context must interrupt socket I/O even without its own deadline.
		cancelDone := make(chan struct{})
		stopCancellation := context.AfterFunc(ctx, func() {
			_ = c.conn.SetDeadline(time.Now())
			close(cancelDone)
		})

		defer func() {
			if !stopCancellation() {
				// Do not release queryMu while the callback can still change socket deadlines.
				<-cancelDone
			}
		}()
	}

	if err := c.setDeadline(ctx, true); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	if _, err := c.conn.Write(packet); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{}, ctxErr
		}
		return Response{}, fmt.Errorf("%w: %w", ErrWriteMasterRequest, err)
	}

	if err := c.setDeadline(ctx, false); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	n, err := c.conn.Read(c.readBuffer)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{}, ctxErr
		}
		if closedErr := c.isClosed(); closedErr != nil {
			return Response{}, closedErr
		}

		return Response{}, fmt.Errorf("%w: %w", ErrReadMasterResponse, err)
	}

	if n > c.maxDatagramSize {
		return Response{}, fmt.Errorf("%w: %d", ErrResponseExceedsDatagramLimit, c.maxDatagramSize)
	}

	page, err := DecodeResponse(c.readBuffer[:n])
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrDecodeMasterResponse, err)
	}

	return page, nil
}

// Stream follows sequential master-server cursors
// until a terminal page or an error is received.
// The caller must receive until the channel closes
// or cancel ctx to stop the producer when abandoning the stream.
func (c *Client) Stream(ctx context.Context, req Request) <-chan Result {
	if ctx == nil {
		results := make(chan Result, 1)
		results <- Result{Err: ErrNilStreamContext}
		close(results)
		return results
	}

	results := make(chan Result)
	go func() {
		defer close(results)
		if !req.Cursor.Addr.IsValid() {
			req.Cursor = InitialCursor()
		}

		for {
			page, err := c.Query(ctx, req)
			if err != nil {
				sendResult(ctx, results, Result{Err: err})
				return
			}

			if !page.Done && (len(page.Servers) == 0 || !page.NextCursor.Addr.IsValid()) {
				// A continuation without content and a valid cursor cannot make progress.
				sendResult(ctx, results, Result{Err: ErrMissingContinuationCursor})
				return
			}

			if !page.Done && page.NextCursor == req.Cursor {
				sendResult(ctx, results, Result{Err: ErrCursorDidNotAdvance})
				return
			}

			if !sendResult(ctx, results, Result{Page: page}) {
				return
			}

			if page.Done {
				return
			}

			req.Cursor = page.NextCursor
		}
	}()

	return results
}

// sendResult lets Stream stop waiting when its caller cancels without receiving.
func sendResult(ctx context.Context, results chan<- Result, result Result) bool {
	select {
	case results <- result:
		return true

	case <-ctx.Done():
		return false
	}
}

// Close closes the client's UDP socket.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	c.stateMu.Lock()
	if c.closed {
		c.stateMu.Unlock()
		return nil
	}

	// Only the first caller closes the socket; repeated Close calls are harmless.
	c.closed = true
	err := c.conn.Close()
	c.stateMu.Unlock()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCloseMasterClient, err)
	}

	return nil
}

// isClosed returns net.ErrClosed when Close has already started.
func (c *Client) isClosed() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.closed {
		return net.ErrClosed
	}

	return nil
}

// setDeadline uses the earlier of the client I/O timeout and the context deadline.
func (c *Client) setDeadline(ctx context.Context, write bool) error {
	deadline := time.Now().Add(c.ioTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}

	if write {
		return c.conn.SetWriteDeadline(deadline)
	}

	return c.conn.SetReadDeadline(deadline)
}
