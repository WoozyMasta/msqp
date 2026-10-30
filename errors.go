// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import "errors"

var (
	// ErrClientUninitialized indicates that a client is nil or was not initialized.
	ErrClientUninitialized = errors.New("master client is nil or uninitialized")
	// ErrNilQueryContext indicates that Query received a nil context.
	ErrNilQueryContext = errors.New("master query context is nil")
	// ErrNilStreamContext indicates that Stream received a nil context.
	ErrNilStreamContext = errors.New("master stream context is nil")
	// ErrResolveMasterAddress marks failures resolving a master endpoint.
	ErrResolveMasterAddress = errors.New("resolve master address")
	// ErrDialMasterAddress marks failures opening a connection to a master.
	ErrDialMasterAddress = errors.New("dial master address")
	// ErrConfigureMasterClient marks failures applying client options.
	ErrConfigureMasterClient = errors.New("configure master client")
	// ErrEncodeMasterRequest marks failures encoding a client request.
	ErrEncodeMasterRequest = errors.New("encode master request")
	// ErrWriteMasterRequest marks failures writing a client request.
	ErrWriteMasterRequest = errors.New("write master request")
	// ErrReadMasterResponse marks failures reading a client response.
	ErrReadMasterResponse = errors.New("read master response")
	// ErrResponseExceedsDatagramLimit indicates a response exceeded the client limit.
	ErrResponseExceedsDatagramLimit = errors.New("master response exceeds configured datagram size")
	// ErrDecodeMasterResponse marks failures decoding a client response.
	ErrDecodeMasterResponse = errors.New("decode master response")
	// ErrCloseMasterClient marks failures closing the client socket.
	ErrCloseMasterClient = errors.New("close master client")
	// ErrMissingContinuationCursor indicates a continued response has no cursor.
	ErrMissingContinuationCursor = errors.New("master response has no continuation cursor")
	// ErrCursorDidNotAdvance indicates that a continued response repeats its cursor.
	ErrCursorDidNotAdvance = errors.New("master response cursor did not advance")

	// ErrMasterFilterContainsNUL indicates a wire filter contains a NUL byte.
	ErrMasterFilterContainsNUL = errors.New("master filter must not contain NUL")
	// ErrFilterKeyValueContainsNUL indicates a filter key or value contains NUL.
	ErrFilterKeyValueContainsNUL = errors.New("filter key and value must not contain NUL")
	// ErrInvalidRequestHeader indicates a request packet has an invalid header.
	ErrInvalidRequestHeader = errors.New("invalid master request header")
	// ErrUnterminatedRequestCursor indicates a request cursor lacks a terminator.
	ErrUnterminatedRequestCursor = errors.New("master cursor is not NUL-terminated")
	// ErrInvalidRequestCursor indicates a request cursor is not an IPv4 endpoint.
	ErrInvalidRequestCursor = errors.New("master request cursor is not a valid IPv4 address")
	// ErrUnterminatedRequestFilter indicates a request filter lacks a terminator.
	ErrUnterminatedRequestFilter = errors.New("master filter is not NUL-terminated")
	// ErrTrailingRequestData indicates bytes follow the request filter terminator.
	ErrTrailingRequestData = errors.New("trailing bytes after master filter")
	// ErrTerminalMarkerAsServer indicates a response lists the protocol terminator as a server.
	ErrTerminalMarkerAsServer = errors.New("master response servers must not contain the terminal marker")
	// ErrInvalidResponseHeader indicates a response packet has an invalid header.
	ErrInvalidResponseHeader = errors.New("invalid master response header")
	// ErrTruncatedAddressEntry indicates a response ends in a partial address entry.
	ErrTruncatedAddressEntry = errors.New("master response has truncated address entry")
	// ErrDecodeServerAddress marks failures decoding one response address.
	ErrDecodeServerAddress = errors.New("decode master server address")
	// ErrDataAfterTerminalMarker indicates bytes follow the terminal response entry.
	ErrDataAfterTerminalMarker = errors.New("master response contains data after terminal entry")
	// ErrInvalidCursorAddress indicates a cursor has no valid address.
	ErrInvalidCursorAddress = errors.New("master cursor address is invalid")
	// ErrCursorMustBeIPv4 indicates a cursor uses a non-IPv4 address.
	ErrCursorMustBeIPv4 = errors.New("legacy master cursor must be IPv4")
	// ErrCursorPortMustBeNonzero indicates a non-initial cursor has port zero.
	ErrCursorPortMustBeNonzero = errors.New("master cursor port must be non-zero for a non-zero address")
	// ErrServerAddressMustBeIPv4 indicates a server uses a non-IPv4 address.
	ErrServerAddressMustBeIPv4 = errors.New("legacy master server address must be IPv4")
	// ErrServerPortMustBeNonzero indicates a server address uses port zero.
	ErrServerPortMustBeNonzero = errors.New("master server port must be non-zero")

	// ErrConflictingFilterGroups indicates a filter combines NOR and NAND groups.
	ErrConflictingFilterGroups = errors.New("cannot use both nor and nand conditions in the same filter")
	// ErrFilterContainsDelimiter indicates a filter key or value contains a delimiter.
	ErrFilterContainsDelimiter = errors.New("filter key and value must not contain the backslash delimiter")
	// ErrEmptyFilterKey indicates a filter clause has an empty key.
	ErrEmptyFilterKey = errors.New("filter key must not be empty")

	// ErrMaxDatagramSizeMustBePositive indicates a non-positive client buffer limit.
	ErrMaxDatagramSizeMustBePositive = errors.New("max datagram size must be positive")
	// ErrMaxDatagramSizeExceedsUDP indicates a client limit exceeds UDP capacity.
	ErrMaxDatagramSizeExceedsUDP = errors.New("max datagram size exceeds the UDP payload limit")
	// ErrIOTimeoutMustBePositive indicates a non-positive client I/O timeout.
	ErrIOTimeoutMustBePositive = errors.New("I/O timeout must be positive")

	// ErrNilMasterHandler indicates that NewServer received a nil handler.
	ErrNilMasterHandler = errors.New("master handler is nil")
	// ErrNilPacketConnection indicates that Serve received a nil connection.
	ErrNilPacketConnection = errors.New("master packet connection is nil")
	// ErrMasterServerAlreadyServing indicates Serve was called more than once.
	ErrMasterServerAlreadyServing = errors.New("master server is already serving")
	// ErrNilShutdownContext indicates that Shutdown received a nil context.
	ErrNilShutdownContext = errors.New("master shutdown context is nil")
	// ErrConfigureMasterServer marks failures applying server options.
	ErrConfigureMasterServer = errors.New("configure master server")
	// ErrReadMasterRequest marks failures receiving a request packet.
	ErrReadMasterRequest = errors.New("read master request")
	// ErrMasterHandlerPanicked indicates the request handler panicked.
	ErrMasterHandlerPanicked = errors.New("master request handler panic")
	// ErrHandleMasterRequest marks errors returned by a request handler.
	ErrHandleMasterRequest = errors.New("handle master request")
	// ErrBoundMasterResponse marks failures limiting a response to packet size.
	ErrBoundMasterResponse = errors.New("bound master response")
	// ErrEncodeMasterResponse marks failures encoding a server response.
	ErrEncodeMasterResponse = errors.New("encode master response")
	// ErrEncodedResponseExceedsLimit indicates the encoded response exceeds its limit.
	ErrEncodedResponseExceedsLimit = errors.New("encoded master response exceeds configured limit")
	// ErrWriteMasterResponse marks failures sending a response packet.
	ErrWriteMasterResponse = errors.New("write master response")
	// ErrResponseCannotFitAddress indicates the configured limit cannot fit one address.
	ErrResponseCannotFitAddress = errors.New("response limit cannot fit an address")
	// ErrNonTerminalResponseEmpty indicates a continued response has no servers.
	ErrNonTerminalResponseEmpty = errors.New("non-terminal master response has no servers")
	// ErrMaxRequestSizeOutOfRange indicates a server request limit is invalid.
	ErrMaxRequestSizeOutOfRange = errors.New("max request size is outside the UDP payload limit")
	// ErrMaxResponseSizeOutOfRange indicates a server response limit is invalid.
	ErrMaxResponseSizeOutOfRange = errors.New("max response size is outside the supported UDP payload range")
	// ErrMaxConcurrentWorkersMustBePositive indicates a non-positive worker limit.
	ErrMaxConcurrentWorkersMustBePositive = errors.New("max concurrent workers must be positive")
)
