// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"context"
	"net"
	"net/netip"
	"testing"
)

func BenchmarkEncodeResponse220Servers(b *testing.B) {
	response := benchmarkResponse()
	b.ReportAllocs()
	b.SetBytes(int64(len(response.Servers) * 6))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := EncodeResponse(response); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeResponse220Servers(b *testing.B) {
	packet, err := EncodeResponse(benchmarkResponse())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(packet)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := DecodeResponse(packet); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClientQueryLoopback(b *testing.B) {
	listener, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	response, err := EncodeResponse(benchmarkResponse())
	if err != nil {
		_ = listener.Close()
		b.Fatal(err)
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		buffer := make([]byte, 256)
		for {
			_, addr, readErr := listener.ReadFrom(buffer)
			if readErr != nil {
				return
			}
			if _, writeErr := listener.WriteTo(response, addr); writeErr != nil {
				return
			}
		}
	}()
	b.Cleanup(func() {
		_ = listener.Close()
		<-serverDone
	})

	client, err := NewClient(listener.LocalAddr().String())
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = client.Close() })
	b.ReportAllocs()
	b.SetBytes(int64(len(response)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := client.Query(context.Background(), Request{}); err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkResponse builds a representative terminal page with 220 addresses.
func benchmarkResponse() Response {
	servers := make([]netip.AddrPort, 220)
	for index := range servers {
		servers[index] = netip.AddrPortFrom(
			netip.AddrFrom4([4]byte{198, 51, 100, byte(index + 1)}),
			uint16(27015+index),
		)
	}

	return Response{Servers: servers, Done: true}
}
