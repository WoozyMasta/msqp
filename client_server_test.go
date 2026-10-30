// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/woozymasta/msqp"
	msqpserver "github.com/woozymasta/msqp/server"
)

func TestClientAndServer(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}

	addresses := []netip.AddrPort{
		netip.MustParseAddrPort("192.0.2.1:27015"),
		netip.MustParseAddrPort("192.0.2.2:27016"),
	}
	server, err := msqpserver.NewServer(msqpserver.NewStaticHandler(addresses))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(conn) }()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(20*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	page, err := client.Query(context.Background(), msqp.Request{
		Region: 3,
		Cursor: msqp.InitialCursor(),
		Filter: `\appid\221100`,
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if !page.Done || len(page.Servers) != len(addresses) {
		t.Fatalf("Query() = %#v, want done page with %d servers", page, len(addresses))
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}

func TestClientStream(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	server, err := msqpserver.NewServer(msqpserver.HandlerFunc(func(_ context.Context, req *msqp.Request) (msqp.Response, error) {
		if req.Cursor.Addr == msqp.InitialCursor().Addr {
			return msqp.Response{
				Servers:    []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:27015")},
				NextCursor: msqp.Cursor{Addr: netip.MustParseAddrPort("192.0.2.1:27015")},
			}, nil
		}

		return msqp.Response{
			Servers:    []netip.AddrPort{netip.MustParseAddrPort("192.0.2.2:27016")},
			NextCursor: msqp.Cursor{Addr: netip.MustParseAddrPort("192.0.2.2:27016")},
			Done:       true,
		}, nil
	}))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(conn) }()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(20*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	var pages int
	for result := range client.Stream(context.Background(), msqp.Request{Cursor: msqp.InitialCursor()}) {
		if result.Err != nil {
			t.Fatalf("Stream() result error = %v", result.Err)
		}
		pages++
	}
	if pages != 2 {
		t.Fatalf("Stream() pages = %d, want 2", pages)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}

func TestClientStreamRejectsNilContext(t *testing.T) {
	var client msqp.Client
	results := client.Stream(nil, msqp.Request{}) //nolint:staticcheck // verify public nil-context handling
	result, ok := <-results
	if !ok {
		t.Fatal("Stream() closed without an error result")
	}
	if !errors.Is(result.Err, msqp.ErrNilStreamContext) {
		t.Fatalf("Stream() error = %v", result.Err)
	}
	if _, ok := <-results; ok {
		t.Fatal("Stream() returned more than one result")
	}
}

func TestZeroClientReturnsErrorsInsteadOfPanicking(t *testing.T) {
	var client msqp.Client
	if _, err := client.Query(context.Background(), msqp.Request{}); err == nil {
		t.Fatal("Query() error = nil, want uninitialized client error")
	}
	result := <-client.Stream(context.Background(), msqp.Request{})
	if result.Err == nil {
		t.Fatal("Stream() error = nil, want uninitialized client error")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestClientQueryReturnsAfterIOTimeout(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	requestReceived := make(chan struct{})
	go func() {
		buffer := make([]byte, 64)
		_, _, _ = conn.ReadFrom(buffer)
		close(requestReceived)
	}()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(25*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Query(context.Background(), msqp.Request{})
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("Query() error = %v, want timeout", err)
	}
	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("server did not receive the query")
	}
}

func TestClientQueryReturnsOnContextCancellation(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	requestReceived := make(chan struct{})
	go func() {
		buffer := make([]byte, 64)
		_, _, _ = conn.ReadFrom(buffer)
		close(requestReceived)
	}()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	queryDone := make(chan error, 1)
	go func() {
		_, queryErr := client.Query(ctx, msqp.Request{})
		queryDone <- queryErr
	}()
	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("server did not receive the query")
	}
	cancel()

	select {
	case err := <-queryDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Query() error = %v, want context cancellation", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Query() did not return promptly after context cancellation")
	}
}

func TestClientQueryDetectsOversizedDatagram(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	defer func() { _ = conn.Close() }()
	go func() {
		buffer := make([]byte, 64)
		_, addr, readErr := conn.ReadFrom(buffer)
		if readErr != nil {
			return
		}

		packet, encodeErr := msqp.EncodeResponse(msqp.Response{
			Servers: []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:27015")},
			Done:    true,
		})
		if encodeErr == nil {
			_, _ = conn.WriteTo(packet, addr)
		}
	}()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithMaxDatagramSize(17))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()
	if _, err := client.Query(context.Background(), msqp.Request{}); !errors.Is(err, msqp.ErrResponseExceedsDatagramLimit) {
		t.Fatalf("Query() error = %v, want %v", err, msqp.ErrResponseExceedsDatagramLimit)
	}
}

func TestClientStreamRejectsNonAdvancingCursor(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	address := netip.MustParseAddrPort("192.0.2.1:27015")
	server, err := msqpserver.NewServer(msqpserver.HandlerFunc(func(_ context.Context, _ *msqp.Request) (msqp.Response, error) {
		return msqp.Response{
			Servers:    []netip.AddrPort{address},
			NextCursor: msqp.Cursor{Addr: address},
		}, nil
	}))
	if err != nil {
		_ = conn.Close()
		t.Fatalf("NewServer() error = %v", err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(conn) }()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(time.Second))
	if err != nil {
		_ = server.Shutdown(context.Background())
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	results := client.Stream(context.Background(), msqp.Request{})
	first := <-results
	if first.Err != nil || len(first.Page.Servers) != 1 {
		t.Fatalf("first Stream() result = %#v, want one server page", first)
	}
	second := <-results
	if !errors.Is(second.Err, msqp.ErrCursorDidNotAdvance) {
		t.Fatalf("second Stream() result = %#v, want cursor progress error", second)
	}
	if _, ok := <-results; ok {
		t.Fatal("Stream() returned results after cursor progress error")
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}
