// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"errors"
	"net/netip"
	"reflect"
	"testing"
)

func TestRequestRoundTrip(t *testing.T) {
	want := Request{
		Region: 3,
		Cursor: Cursor{Addr: netip.MustParseAddrPort("192.0.2.10:27015")},
		Filter: `\appid\221100\secure\1`,
	}

	encoded, err := EncodeRequest(want)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}
	got, err := DecodeRequest(encoded)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DecodeRequest() = %#v, want %#v", got, want)
	}
}

func TestResponseRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		want Response
	}{
		{
			name: "continued",
			want: Response{
				Servers: []netip.AddrPort{
					netip.MustParseAddrPort("192.0.2.1:27015"),
					netip.MustParseAddrPort("192.0.2.2:27016"),
				},
				NextCursor: Cursor{Addr: netip.MustParseAddrPort("192.0.2.2:27016")},
			},
		},
		{
			name: "done",
			want: Response{
				Servers: []netip.AddrPort{
					netip.MustParseAddrPort("192.0.2.1:27015"),
				},
				NextCursor: Cursor{Addr: netip.MustParseAddrPort("192.0.2.1:27015")},
				Done:       true,
			},
		},
		{
			name: "empty done",
			want: Response{
				NextCursor: InitialCursor(),
				Done:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeResponse(tt.want)
			if err != nil {
				t.Fatalf("EncodeResponse() error = %v", err)
			}
			got, err := DecodeResponse(encoded)
			if err != nil {
				t.Fatalf("DecodeResponse() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("DecodeResponse() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecodeResponseRejectsMalformedPacket(t *testing.T) {
	tests := []struct {
		name   string
		packet []byte
		want   error
	}{
		{
			name:   "bad header",
			packet: []byte{0xff},
			want:   ErrInvalidResponseHeader,
		},
		{
			name:   "truncated entry",
			packet: append(responseHeader[:], 1, 2),
			want:   ErrTruncatedAddressEntry,
		},
		{
			name: "data after terminal",
			packet: append(
				append(responseHeader[:], 0, 0, 0, 0, 0, 0),
				[]byte{0, 0, 0, 1, 0x69, 0x87}...,
			),
			want: ErrDataAfterTerminalMarker,
		},
		{
			name:   "server with zero port",
			packet: append(responseHeader[:], 192, 0, 2, 1, 0, 0),
			want:   ErrServerPortMustBeNonzero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeResponse(tt.packet); !errors.Is(err, tt.want) {
				t.Fatalf("DecodeResponse() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEncodeRequestRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		request Request
		want    error
	}{
		{
			request: Request{
				Cursor: Cursor{
					Addr: netip.MustParseAddrPort("[2001:db8::1]:27015"),
				},
			},
			want: ErrCursorMustBeIPv4,
		},
		{
			request: Request{
				Cursor: Cursor{
					Addr: netip.MustParseAddrPort("192.0.2.1:0"),
				},
			},
			want: ErrCursorPortMustBeNonzero,
		},
		{
			request: Request{
				Cursor: Cursor{
					Addr: netip.MustParseAddrPort("192.0.2.1:27015"),
				},
				Filter: "bad\x00filter",
			},
			want: ErrMasterFilterContainsNUL,
		},
	}
	for _, test := range tests {
		if _, err := EncodeRequest(test.request); !errors.Is(err, test.want) {
			t.Fatalf("EncodeRequest(%#v) error = %v, want %v", test.request, err, test.want)
		}
	}
}

func TestEncodeResponseRejectsTerminalMarkerAsServer(t *testing.T) {
	_, err := EncodeResponse(Response{
		Servers: []netip.AddrPort{netip.MustParseAddrPort("0.0.0.0:0")},
	})
	if !errors.Is(err, ErrTerminalMarkerAsServer) {
		t.Fatalf("EncodeResponse() error = %v, want %v", err, ErrTerminalMarkerAsServer)
	}
}

func TestEncodeResponseRejectsServerWithZeroPort(t *testing.T) {
	_, err := EncodeResponse(Response{
		Servers: []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:0")},
	})
	if !errors.Is(err, ErrServerPortMustBeNonzero) {
		t.Fatalf("EncodeResponse() error = %v, want %v", err, ErrServerPortMustBeNonzero)
	}
}
