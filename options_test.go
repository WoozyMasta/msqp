// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"errors"
	"testing"
)

func TestClientOptionErrorsAreClassifiable(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		want error
	}{
		{
			name: "non-positive datagram limit",
			opt:  WithMaxDatagramSize(0),
			want: ErrMaxDatagramSizeMustBePositive,
		},
		{
			name: "datagram limit exceeds UDP maximum",
			opt:  WithMaxDatagramSize(maxUDPDatagramSize + 1),
			want: ErrMaxDatagramSizeExceedsUDP,
		},
		{
			name: "non-positive I/O timeout",
			opt:  WithIOTimeout(0),
			want: ErrIOTimeoutMustBePositive,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient("127.0.0.1:27011", test.opt)
			if !errors.Is(err, test.want) || !errors.Is(err, ErrConfigureMasterClient) {
				t.Fatalf("NewClient() error = %v, want %v and %v", err, test.want, ErrConfigureMasterClient)
			}
		})
	}
}
