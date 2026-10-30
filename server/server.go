// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

// Package server implements the MSQP UDP server.
package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/woozymasta/msqp"
)

// Handler serves one MSQP query.
type Handler interface {
	HandleMaster(context.Context, *msqp.Request) (msqp.Response, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, *msqp.Request) (msqp.Response, error)

// HandleMaster implements Handler.
func (f HandlerFunc) HandleMaster(ctx context.Context, req *msqp.Request) (msqp.Response, error) {
	return f(ctx, req)
}

// Server exposes a Handler through MSQP over UDP.
type Server struct {
	// handler supplies the server list for each decoded request.
	handler Handler
	// conn is the packet socket currently owned by Serve.
	conn net.PacketConn
	// cancel stops the context shared by active handlers during Shutdown.
	cancel context.CancelFunc
	// serveDone closes after Serve stops accepting packets.
	serveDone chan struct{}
	// errorHandler reports per-request failures without terminating Serve.
	errorHandler func(error)
	// workerDone tracks handlers started by Serve.
	workerDone sync.WaitGroup
	// maxRequestSize bounds accepted request packets.
	maxRequestSize int
	// maxResponseSize bounds encoded response packets.
	maxResponseSize int
	// maxConcurrentWorkers bounds active handler goroutines.
	maxConcurrentWorkers int
	// shutdown makes the close-and-cancel transition idempotent.
	shutdown sync.Once
	// mu protects lifecycle state shared by Serve, Shutdown, and workers.
	mu sync.Mutex
	// errorMu serializes callbacks because a configured handler may not be concurrent-safe.
	errorMu sync.Mutex
	// closed prevents serving again after Shutdown begins.
	closed bool
}

// NewServer creates a legacy UDP master server.
func NewServer(handler Handler, opts ...Option) (*Server, error) {
	if handler == nil {
		return nil, msqp.ErrNilMasterHandler
	}

	config := serverConfig{
		maxRequestSize:       defaultMaxRequestSize,
		maxResponseSize:      defaultMaxResponseSize,
		maxConcurrentWorkers: defaultMaxConcurrentWorkers,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(&config); err != nil {
			return nil, fmt.Errorf("%w: %w", msqp.ErrConfigureMasterServer, err)
		}
	}

	return &Server{
		handler:              handler,
		maxRequestSize:       config.maxRequestSize,
		maxResponseSize:      config.maxResponseSize,
		maxConcurrentWorkers: config.maxConcurrentWorkers,
		errorHandler:         config.errorHandler,
		serveDone:            make(chan struct{}),
	}, nil
}

// Serve reads and handles requests from conn until it is closed or fails.
func (s *Server) Serve(conn net.PacketConn) error {
	if conn == nil {
		return msqp.ErrNilPacketConnection
	}

	s.mu.Lock()

	if s.conn != nil {
		s.mu.Unlock()
		return msqp.ErrMasterServerAlreadyServing
	}
	if s.closed {
		s.mu.Unlock()
		return net.ErrClosed
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.conn = conn
	s.cancel = cancel
	s.mu.Unlock()
	defer close(s.serveDone)
	defer cancel()

	workers := make(chan struct{}, s.maxConcurrentWorkers)
	// One extra byte distinguishes an oversized request from a packet at the limit.
	buffer := make([]byte, s.maxRequestSize+1)

	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			if s.isClosed() {
				return nil
			}
			return fmt.Errorf("%w: %w", msqp.ErrReadMasterRequest, err)
		}

		if n > s.maxRequestSize {
			continue
		}

		packet := append([]byte(nil), buffer[:n]...)

		// The bounded channel is a semaphore; do not launch an unbounded goroutine per packet.
		select {
		case workers <- struct{}{}:
		case <-ctx.Done():
			return nil
		}

		s.workerDone.Add(1)
		go func() {
			defer s.workerDone.Done()
			defer func() { <-workers }()
			s.handlePacket(ctx, conn, addr, packet)
		}()
	}
}

// Shutdown stops serving and waits for active handlers or ctx cancellation.
func (s *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return msqp.ErrNilShutdownContext
	}

	s.shutdown.Do(func() {
		s.mu.Lock()

		s.closed = true
		conn := s.conn
		cancel := s.cancel

		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if conn != nil {
			_ = conn.Close()
		}
	})

	s.mu.Lock()
	serving := s.conn != nil
	s.mu.Unlock()

	if !serving {
		return nil
	}

	select {
	case <-s.serveDone:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Serve has stopped adding workers, so Wait cannot race with a later Add call.
	done := make(chan struct{})
	go func() {
		s.workerDone.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handlePacket validates one request, invokes the handler, and writes a bounded response.
func (s *Server) handlePacket(ctx context.Context, conn net.PacketConn, addr net.Addr, packet []byte) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// A faulty handler must not crash the serving process or worker pool.
			s.reportError(fmt.Errorf("%w: %v", msqp.ErrMasterHandlerPanicked, recovered))
		}
	}()

	request, err := msqp.DecodeRequest(packet)
	if err != nil {
		return
	}

	response, err := s.handler.HandleMaster(ctx, &request)
	if err != nil {
		s.reportError(fmt.Errorf("%w: %w", msqp.ErrHandleMasterRequest, err))
		return
	}

	response, err = fitResponse(response, s.maxResponseSize)
	if err != nil {
		s.reportError(fmt.Errorf("%w: %w", msqp.ErrBoundMasterResponse, err))
		return
	}

	encoded, err := msqp.EncodeResponse(response)
	if err != nil {
		s.reportError(fmt.Errorf("%w: %w", msqp.ErrEncodeMasterResponse, err))
		return
	}

	if len(encoded) > s.maxResponseSize {
		s.reportError(fmt.Errorf(
			"%w: got %d bytes, limit %d",
			msqp.ErrEncodedResponseExceedsLimit,
			len(encoded),
			s.maxResponseSize,
		))
		return
	}

	if _, err := conn.WriteTo(encoded, addr); err != nil {
		s.reportError(fmt.Errorf("%w: %w", msqp.ErrWriteMasterResponse, err))
	}
}

// fitResponse truncates a page to the datagram limit and derives its continuation cursor.
func fitResponse(response msqp.Response, maxSize int) (msqp.Response, error) {
	// A page costs 6 header bytes plus 6 per address; Done also needs 6 marker bytes.
	maxWithoutTerminator := (maxSize - 6) / 6
	if maxWithoutTerminator <= 0 {
		return msqp.Response{}, msqp.ErrResponseCannotFitAddress
	}

	maxWithTerminator := (maxSize - 12) / 6
	if response.Done && len(response.Servers) <= maxWithTerminator {
		return response, nil
	}

	if response.Done {
		// Continue an oversized terminal result so the next cursor request can fetch its tail.
		response.Done = false
	}
	if len(response.Servers) == 0 {
		return msqp.Response{}, msqp.ErrNonTerminalResponseEmpty
	}
	if len(response.Servers) > maxWithoutTerminator {
		response.Servers = response.Servers[:maxWithoutTerminator]
	}

	// The protocol derives continuation from the final address actually sent.
	response.NextCursor = msqp.Cursor{Addr: response.Servers[len(response.Servers)-1]}

	return response, nil
}

// reportError invokes the optional callback serially when a request cannot be served.
func (s *Server) reportError(err error) {
	if s.errorHandler == nil {
		return
	}

	s.errorMu.Lock()
	defer s.errorMu.Unlock()
	s.errorHandler(err)
}

// isClosed reports whether Shutdown has begun.
func (s *Server) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
