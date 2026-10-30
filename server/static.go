// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package server

import (
	"context"
	"net/netip"

	"github.com/woozymasta/msqp"
)

// NewStaticHandler returns a deterministic handler
// useful for fixtures and simple local masters.
// The filter is intentionally passed through but not interpreted.
func NewStaticHandler(addresses []netip.AddrPort) Handler {
	copyAddresses := make([]netip.AddrPort, 0, len(addresses))
	seen := make(map[netip.AddrPort]struct{}, len(addresses))

	for _, address := range addresses {
		if _, exists := seen[address]; exists {
			continue
		}

		// A cursor must identify one position unambiguously across page boundaries.
		seen[address] = struct{}{}
		copyAddresses = append(copyAddresses, address)
	}

	return HandlerFunc(func(_ context.Context, req *msqp.Request) (msqp.Response, error) {
		start := 0
		if req.Cursor.Addr.IsValid() && req.Cursor.Addr != msqp.InitialCursor().Addr {
			found := false
			for i, address := range copyAddresses {
				if address == req.Cursor.Addr {
					// The cursor is the last address already returned, so continue after it.
					start = i + 1
					found = true
					break
				}
			}

			if !found {
				// Unknown cursors must not restart from the first page and duplicate results.
				return msqp.Response{NextCursor: msqp.InitialCursor(), Done: true}, nil
			}
		}

		if start >= len(copyAddresses) {
			return msqp.Response{NextCursor: msqp.InitialCursor(), Done: true}, nil
		}

		return msqp.Response{
			Servers:    append([]netip.AddrPort(nil), copyAddresses[start:]...),
			NextCursor: msqp.Cursor{Addr: copyAddresses[len(copyAddresses)-1]},
			Done:       true,
		}, nil
	})
}
