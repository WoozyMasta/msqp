// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"net/netip"
	"testing"
)

func FuzzDecodeRequest(f *testing.F) {
	seed, err := EncodeRequest(Request{
		Cursor: Cursor{Addr: netip.MustParseAddrPort("192.0.2.1:27015")},
		Filter: `\appid\730`,
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte{0x31, 0x00, '0', 0x00, 0x00})

	f.Fuzz(func(_ *testing.T, input []byte) {
		_, _ = DecodeRequest(input)
	})
}

func FuzzDecodeResponse(f *testing.F) {
	seed, err := EncodeResponse(Response{
		Servers: []netip.AddrPort{netip.MustParseAddrPort("192.0.2.1:27015")},
		Done:    true,
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add(responseHeader[:])

	f.Fuzz(func(_ *testing.T, input []byte) {
		_, _ = DecodeResponse(input)
	})
}
