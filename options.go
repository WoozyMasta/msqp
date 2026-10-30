// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import "time"

// Option configures a legacy master client.
type Option func(*clientConfig) error

// clientConfig holds validated values applied when constructing a Client.
type clientConfig struct {
	// maxDatagramSize controls the receive limit and reusable buffer capacity.
	maxDatagramSize int
	// ioTimeout bounds each UDP read or write when the context has no earlier deadline.
	ioTimeout time.Duration
}

// WithMaxDatagramSize limits the receive buffer allocated by the client.
func WithMaxDatagramSize(size int) Option {
	return func(config *clientConfig) error {
		if size <= 0 {
			return ErrMaxDatagramSizeMustBePositive
		}
		if size > maxUDPDatagramSize {
			return ErrMaxDatagramSizeExceedsUDP
		}

		config.maxDatagramSize = size
		return nil
	}
}

// WithIOTimeout bounds individual UDP I/O attempts
// when the context has no earlier deadline.
func WithIOTimeout(timeout time.Duration) Option {
	return func(config *clientConfig) error {
		if timeout <= 0 {
			return ErrIOTimeoutMustBePositive
		}

		config.ioTimeout = timeout
		return nil
	}
}
