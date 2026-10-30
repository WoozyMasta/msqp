// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/netip"
)

// requestType identifies an A2M_GET_SERVERS_BATCH2 request packet.
const requestType byte = 0x31

// responseHeader prefixes every legacy master-server response page.
var responseHeader = [...]byte{0xff, 0xff, 0xff, 0xff, 0x66, 0x0a}

// Region is the legacy master-server geographic region code.
type Region uint8

// Cursor identifies the last IPv4 endpoint returned by a master server.
type Cursor struct {
	// Addr is the cursor endpoint;
	// the unspecified IPv4 address with port zero starts a query.
	Addr netip.AddrPort `json:"addr"`
}

// InitialCursor returns the cursor used for the first query page.
func InitialCursor() Cursor {
	return Cursor{Addr: netip.AddrPortFrom(netip.IPv4Unspecified(), 0)}
}

// Request is a legacy master-server query.
type Request struct {
	// Cursor resumes after this endpoint; an invalid address starts from the first page.
	Cursor Cursor `json:"cursor"`
	// Filter is the Steam master-server filter expression, without a trailing NUL.
	Filter string `json:"filter"`
	// Region selects a legacy Steam master-server region code.
	Region Region `json:"region"`
}

// Response contains one decoded master-server page.
type Response struct {
	// NextCursor identifies the last returned endpoint for the next page.
	NextCursor Cursor `json:"next_cursor"`
	// Servers contains the endpoints returned in this page.
	Servers []netip.AddrPort `json:"servers"`
	// Done reports whether the page carries the protocol's terminal marker.
	Done bool `json:"done"`
}

// EncodeRequest encodes a legacy master-server query packet.
func EncodeRequest(req Request) ([]byte, error) {
	cursor, err := encodeCursor(req.Cursor)
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte([]byte(req.Filter), 0) >= 0 {
		return nil, ErrMasterFilterContainsNUL
	}

	packet := make([]byte, 0, 3+len(cursor)+len(req.Filter))
	packet = append(packet, requestType, byte(req.Region))
	packet = append(packet, cursor...)
	packet = append(packet, 0)
	packet = append(packet, req.Filter...)
	packet = append(packet, 0)
	return packet, nil
}

// DecodeRequest decodes a legacy master-server query packet.
func DecodeRequest(packet []byte) (Request, error) {
	if len(packet) < 3 || packet[0] != requestType {
		return Request{}, ErrInvalidRequestHeader
	}

	// The cursor is the first NUL-terminated field after the two-byte header.
	firstNUL := bytes.IndexByte(packet[2:], 0)
	if firstNUL < 0 {
		return Request{}, ErrUnterminatedRequestCursor
	}

	cursorEnd := 2 + firstNUL
	cursorAddr, err := netip.ParseAddrPort(string(packet[2:cursorEnd]))
	if err != nil || !cursorAddr.Addr().Is4() {
		return Request{}, ErrInvalidRequestCursor
	}

	// A valid request ends exactly at the filter's NUL terminator.
	filterStart := cursorEnd + 1
	filterEnd := bytes.IndexByte(packet[filterStart:], 0)
	if filterEnd < 0 {
		return Request{}, ErrUnterminatedRequestFilter
	}
	if filterStart+filterEnd+1 != len(packet) {
		return Request{}, ErrTrailingRequestData
	}

	return Request{
		Region: Region(packet[1]),
		Cursor: Cursor{Addr: cursorAddr},
		Filter: string(packet[filterStart : filterStart+filterEnd]),
	}, nil
}

// EncodeResponse encodes a master-server page.
// Done adds the terminal 0.0.0.0:0 entry required by the protocol.
func EncodeResponse(response Response) ([]byte, error) {
	packet := make([]byte, 0, len(response.Servers)*6+len(responseHeader)+6)
	packet = append(packet, responseHeader[:]...)
	for _, server := range response.Servers {
		if server == netip.AddrPortFrom(netip.IPv4Unspecified(), 0) {
			return nil, ErrTerminalMarkerAsServer
		}

		encoded, err := encodeAddrPort(server)
		if err != nil {
			return nil, err
		}

		packet = append(packet, encoded[:]...)
	}

	if response.Done {
		packet = append(packet, 0, 0, 0, 0, 0, 0)
	}

	return packet, nil
}

// DecodeResponse decodes a master-server page.
// A page without a terminal marker is considered continued
// and uses its last address as the cursor.
func DecodeResponse(packet []byte) (Response, error) {
	if len(packet) < len(responseHeader) ||
		!bytes.Equal(packet[:len(responseHeader)], responseHeader[:]) {
		return Response{}, ErrInvalidResponseHeader
	}

	remaining := packet[len(responseHeader):]
	if len(remaining)%6 != 0 {
		return Response{}, ErrTruncatedAddressEntry
	}

	response := Response{NextCursor: InitialCursor()}
	if len(remaining) > 0 {
		// Capacity may include the terminal marker;
		// it avoids repeated growth per address.
		response.Servers = make([]netip.AddrPort, 0, len(remaining)/6)
	}

	for offset := 0; offset < len(remaining); offset += 6 {
		addr, err := decodeIPv4Port(remaining[offset:])
		if err != nil {
			return Response{}, fmt.Errorf("%w: %w", ErrDecodeServerAddress, err)
		}

		if addr == netip.AddrPortFrom(netip.IPv4Unspecified(), 0) {
			response.Done = true
			// The marker is meaningful only as the final entry in a response datagram.
			if offset+6 != len(remaining) {
				return Response{}, ErrDataAfterTerminalMarker
			}

			if len(response.Servers) == 0 {
				response.Servers = nil
			}

			return response, nil
		}
		if addr.Port() == 0 {
			return Response{}, fmt.Errorf("%w: %w", ErrDecodeServerAddress, ErrServerPortMustBeNonzero)
		}

		response.Servers = append(response.Servers, addr)
		response.NextCursor = Cursor{Addr: addr}
	}

	return response, nil
}

// encodeCursor validates the IPv4 cursor before placing it in the text request field.
func encodeCursor(cursor Cursor) (string, error) {
	addr := cursor.Addr
	if !addr.IsValid() {
		return "", ErrInvalidCursorAddress
	}
	if !addr.Addr().Is4() {
		return "", ErrCursorMustBeIPv4
	}
	if addr.Port() == 0 && addr.Addr() != netip.IPv4Unspecified() {
		// Port zero is reserved for the all-zero initial cursor in this protocol.
		return "", ErrCursorPortMustBeNonzero
	}

	return addr.String(), nil
}

// encodeAddrPort writes an IPv4 address and port in the protocol's six-byte layout.
func encodeAddrPort(addr netip.AddrPort) ([6]byte, error) {
	if !addr.IsValid() || !addr.Addr().Is4() {
		return [6]byte{}, ErrServerAddressMustBeIPv4
	}
	if addr.Port() == 0 {
		return [6]byte{}, ErrServerPortMustBeNonzero
	}

	encoded := [6]byte{}
	ip := addr.Addr().As4()
	copy(encoded[:4], ip[:])
	binary.BigEndian.PutUint16(encoded[4:], addr.Port())

	return encoded, nil
}

// decodeIPv4Port reads one six-byte IPv4 address and port entry.
func decodeIPv4Port(data []byte) (netip.AddrPort, error) {
	if len(data) < 6 {
		return netip.AddrPort{}, ErrTruncatedAddressEntry
	}

	var ip [4]byte
	copy(ip[:], data[:4])

	return netip.AddrPortFrom(netip.AddrFrom4(ip), binary.BigEndian.Uint16(data[4:6])), nil
}
