// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package server_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/woozymasta/msqp"
	"github.com/woozymasta/msqp/server"
)

func TestServerPaginatesOversizedStaticResponse(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}

	addresses := make([]netip.AddrPort, 10)
	for index := range addresses {
		addresses[index] = netip.AddrPortFrom(netip.AddrFrom4([4]byte{192, 0, 2, byte(index + 1)}), 27015)
	}
	master, err := server.NewServer(
		server.NewStaticHandler(append(addresses, addresses[3])),
		server.WithMaxResponseSize(30),
	)
	if err != nil {
		_ = conn.Close()
		t.Fatalf("NewServer() error = %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- master.Serve(conn) }()

	client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(time.Second))
	if err != nil {
		_ = master.Shutdown(context.Background())
		t.Fatalf("NewClient() error = %v", err)
	}

	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var got []netip.AddrPort
	var pages int
	for result := range client.Stream(ctx, msqp.Request{}) {
		if result.Err != nil {
			t.Fatalf("Stream() error = %v", result.Err)
		}
		pages++
		got = append(got, result.Page.Servers...)
	}
	if len(got) != len(addresses) || pages < 2 {
		t.Fatalf(
			"Stream() returned %d servers in %d pages; want %d servers and multiple pages",
			len(got), pages, len(addresses))
	}
	for index := range addresses {
		if got[index] != addresses[index] {
			t.Fatalf("server[%d] = %s, want %s", index, got[index], addresses[index])
		}
	}

	if err := master.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}

func TestServerReportsHandlerErrorsAndPanics(t *testing.T) {
	handlerFailure := errors.New("handler failure")
	for _, test := range []struct {
		name    string
		handler server.Handler
		want    error
		cause   error
	}{
		{
			name: "error",
			handler: server.HandlerFunc(func(context.Context, *msqp.Request) (msqp.Response, error) {
				return msqp.Response{}, handlerFailure
			}),
			want:  msqp.ErrHandleMasterRequest,
			cause: handlerFailure,
		},
		{
			name: "panic",
			handler: server.HandlerFunc(func(context.Context, *msqp.Request) (msqp.Response, error) {
				panic("handler panic")
			}),
			want: msqp.ErrMasterHandlerPanicked,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("ListenPacket() error = %v", err)
			}

			reportedErrors := make(chan error, 1)
			master, err := server.NewServer(test.handler, server.WithErrorHandler(func(err error) {
				reportedErrors <- err
			}))
			if err != nil {
				_ = conn.Close()
				t.Fatalf("NewServer() error = %v", err)
			}

			serveErr := make(chan error, 1)
			go func() { serveErr <- master.Serve(conn) }()

			client, err := msqp.NewClient(conn.LocalAddr().String(), msqp.WithIOTimeout(50*time.Millisecond))
			if err != nil {
				_ = master.Shutdown(context.Background())
				t.Fatalf("NewClient() error = %v", err)
			}
			_, _ = client.Query(context.Background(), msqp.Request{})
			_ = client.Close()

			select {
			case err := <-reportedErrors:
				if !errors.Is(err, test.want) {
					t.Fatalf("error handler received %v, want %v", err, test.want)
				}
				if test.cause != nil && !errors.Is(err, test.cause) {
					t.Fatalf("error handler received %v, want wrapped cause %v", err, test.cause)
				}

			case <-time.After(time.Second):
				t.Fatal("error handler was not called")
			}

			if err := master.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
			if err := <-serveErr; err != nil {
				t.Fatalf("Serve() error = %v", err)
			}
		})
	}
}

func TestServerOptionErrorsAreClassifiable(t *testing.T) {
	tests := []struct {
		name string
		opt  server.Option
		want error
	}{
		{
			name: "non-positive request limit",
			opt:  server.WithMaxRequestSize(0),
			want: msqp.ErrMaxRequestSizeOutOfRange,
		},
		{
			name: "response limit too small",
			opt:  server.WithMaxResponseSize(11),
			want: msqp.ErrMaxResponseSizeOutOfRange,
		},
		{
			name: "non-positive worker limit",
			opt:  server.WithMaxConcurrentWorkers(0),
			want: msqp.ErrMaxConcurrentWorkersMustBePositive,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := server.NewServer(server.NewStaticHandler(nil), test.opt)
			if !errors.Is(err, test.want) || !errors.Is(err, msqp.ErrConfigureMasterServer) {
				t.Fatalf("NewServer() error = %v, want %v and %v", err, test.want, msqp.ErrConfigureMasterServer)
			}
		})
	}
}
