// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp

import (
	"errors"
	"testing"
)

func TestFilterString(t *testing.T) {
	tests := []struct {
		name   string
		build  func(*Filter)
		want   string
		wantOK bool
	}{
		{
			name: "regular and nor",
			build: func(f *Filter) {
				f.Add(KeyAppID, "221100")
				f.Add(KeyMap, "chernarus")
				f.AddNor(KeyGameType, "external")
			},
			want:   `\appid\221100\map\chernarus\nor\1\gametype\external`,
			wantOK: true,
		},
		{
			name: "nand",
			build: func(f *Filter) {
				f.AddNand(KeyName, "*test*")
				f.AddNand(FilterKey("custom"), "value")
			},
			want:   `\nand\2\name_match\*test*\custom\value`,
			wantOK: true,
		},
		{
			name:   "empty",
			build:  func(*Filter) {},
			want:   "",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter Filter
			tt.build(&filter)
			got, err := filter.String()
			if (err == nil) != tt.wantOK {
				t.Fatalf("String() error = %v, wantOK = %t", err, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterStringRejectsInvalidClauses(t *testing.T) {
	tests := []struct {
		name  string
		build func(*Filter)
		want  error
	}{
		{
			name:  "nul in value",
			build: func(f *Filter) { f.Add(KeyName, "bad\x00value") },
			want:  ErrFilterKeyValueContainsNUL,
		},
		{
			name:  "nul in key",
			build: func(f *Filter) { f.Add(FilterKey("bad\x00key"), "value") },
			want:  ErrFilterKeyValueContainsNUL,
		},
		{
			name:  "empty key",
			build: func(f *Filter) { f.Add("", "value") },
			want:  ErrEmptyFilterKey,
		},
		{
			name:  "delimiter in value",
			build: func(f *Filter) { f.Add(KeyName, "name\\secure\\1") },
			want:  ErrFilterContainsDelimiter,
		},
		{
			name:  "delimiter in key",
			build: func(f *Filter) { f.Add(FilterKey("name\\appid"), "value") },
			want:  ErrFilterContainsDelimiter,
		},
		{
			name: "nor and nand",
			build: func(f *Filter) {
				f.AddNor(KeyName, "nor")
				f.AddNand(KeyName, "nand")
			},
			want: ErrConflictingFilterGroups,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter Filter
			tt.build(&filter)
			if _, err := filter.String(); !errors.Is(err, tt.want) {
				t.Fatalf("String() error = %v, want %v", err, tt.want)
			}
		})
	}
}
