// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package server

import "github.com/woozymasta/msqp"

const (
	// defaultMaxRequestSize bounds packet-copy memory for untrusted requests.
	defaultMaxRequestSize = 4096
	// defaultMaxResponseSize stays below common path MTUs to avoid IP fragmentation.
	defaultMaxResponseSize = 1400
	// defaultMaxConcurrentWorkers bounds simultaneous handler executions.
	defaultMaxConcurrentWorkers = 64
	// minResponseSize fits the six-byte header and six-byte terminal marker.
	minResponseSize = 12 // MSQP header plus terminal entry.
	// maxDatagramSize is the largest IPv4 UDP payload accepted by these options.
	maxDatagramSize = 65507
)

// Option configures an MSQP server.
type Option func(*serverConfig) error

// serverConfig holds validated limits and callbacks used to construct a Server.
type serverConfig struct {
	// errorHandler receives handler and response I/O failures; calls are serialized.
	errorHandler func(error)
	// maxRequestSize bounds each datagram copied for asynchronous handling.
	maxRequestSize int
	// maxResponseSize bounds each encoded response datagram.
	maxResponseSize int
	// maxConcurrentWorkers caps active handler goroutines.
	maxConcurrentWorkers int
}

// WithMaxRequestSize limits the size of a received UDP request.
func WithMaxRequestSize(size int) Option {
	return func(config *serverConfig) error {
		if size <= 0 || size > maxDatagramSize {
			return msqp.ErrMaxRequestSizeOutOfRange
		}

		config.maxRequestSize = size
		return nil
	}
}

// WithMaxResponseSize bounds encoded UDP responses.
func WithMaxResponseSize(size int) Option {
	return func(config *serverConfig) error {
		// Responses must fit both the protocol terminator and at least one address.
		if size < minResponseSize || size > maxDatagramSize {
			return msqp.ErrMaxResponseSizeOutOfRange
		}

		config.maxResponseSize = size
		return nil
	}
}

// WithMaxConcurrentWorkers limits handler callbacks in a server.
func WithMaxConcurrentWorkers(workers int) Option {
	return func(config *serverConfig) error {
		if workers <= 0 {
			return msqp.ErrMaxConcurrentWorkersMustBePositive
		}

		config.maxConcurrentWorkers = workers
		return nil
	}
}

// WithErrorHandler configures a callback for handler,
// response-encoding, and packet-write failures.
// The server serializes callback invocations.
func WithErrorHandler(handler func(error)) Option {
	return func(config *serverConfig) error {
		config.errorHandler = handler
		return nil
	}
}
