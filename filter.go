// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

// Package msqp implements Steam master-server query protocol support.
package msqp

import (
	"fmt"
	"strings"
)

// FilterKey is a Steam master-server filter key. Custom keys are supported.
type FilterKey string

const (
	// KeyDedicated filters for dedicated servers.
	KeyDedicated FilterKey = "dedicated"
	// KeySecure filters for secure servers.
	KeySecure FilterKey = "secure"
	// KeyGameDir filters by game directory.
	KeyGameDir FilterKey = "gamedir"
	// KeyMap filters by map name.
	KeyMap FilterKey = "map"
	// KeyLinux filters by Linux server support.
	KeyLinux FilterKey = "linux"
	// KeyPassword filters by password protection.
	KeyPassword FilterKey = "password"
	// KeyEmpty filters empty servers.
	KeyEmpty FilterKey = "empty"
	// KeyFull filters full servers.
	KeyFull FilterKey = "full"
	// KeyProxy filters proxy servers.
	KeyProxy FilterKey = "proxy"
	// KeyAppID filters by application ID.
	KeyAppID FilterKey = "appid"
	// KeyNotAppID excludes an application ID.
	KeyNotAppID FilterKey = "napp"
	// KeyNoPlayers filters servers without players.
	KeyNoPlayers FilterKey = "noplayers"
	// KeyWhite filters whitelist state.
	KeyWhite FilterKey = "white"
	// KeyGameType filters by game type.
	KeyGameType FilterKey = "gametype"
	// KeyGameData filters by game data.
	KeyGameData FilterKey = "gamedata"
	// KeyGameDataOr filters by any matching game data tag.
	KeyGameDataOr FilterKey = "gamedataor"
	// KeyName filters by server name.
	KeyName FilterKey = "name_match"
	// KeyVersion filters by version.
	KeyVersion FilterKey = "version_match"
	// KeySingleAddr collapses servers by address hash.
	KeySingleAddr FilterKey = "collapse_addr_hash"
	// KeyGameAddr filters by game address.
	KeyGameAddr FilterKey = "gameaddr"
)

// Filter builds the canonical Steam master-server filter string.
type Filter struct {
	// conditions are applied alongside the optional NOR or NAND group.
	conditions []filterClause
	// norConditions are clauses serialized in the mutually exclusive nor group.
	norConditions []filterClause
	// nandConditions are clauses serialized in the mutually exclusive nand group.
	nandConditions []filterClause
}

// filterClause is one unescaped key/value pair before serialization.
type filterClause struct {
	// key names the Steam master-server filter field.
	key FilterKey
	// value is the unescaped value associated with key.
	value string
}

// Add appends a regular key/value clause.
func (f *Filter) Add(key FilterKey, value string) {
	f.conditions = append(f.conditions, filterClause{key: key, value: value})
}

// AddNor appends a clause to the NOR group.
func (f *Filter) AddNor(key FilterKey, value string) {
	f.norConditions = append(f.norConditions, filterClause{key: key, value: value})
}

// AddNand appends a clause to the NAND group.
func (f *Filter) AddNand(key FilterKey, value string) {
	f.nandConditions = append(f.nandConditions, filterClause{key: key, value: value})
}

// String returns the canonical filter string without modifying the builder.
func (f Filter) String() (string, error) {
	if len(f.norConditions) > 0 && len(f.nandConditions) > 0 {
		return "", ErrConflictingFilterGroups
	}

	parts := make([]string, 0, len(f.conditions)+1)
	regular, err := renderClauses(f.conditions)
	if err != nil {
		return "", err
	}

	if regular != "" {
		parts = append(parts, regular)
	}

	// NOR and NAND cannot coexist; choose the populated group after validation.
	group := f.norConditions
	groupName := "nor"
	if len(group) == 0 {
		group = f.nandConditions
		groupName = "nand"
	}

	if len(group) > 0 {
		values, err := renderClauses(group)
		if err != nil {
			return "", err
		}

		parts = append(parts, fmt.Sprintf("%s\\%d\\%s", groupName, len(group), values))
	}

	result := strings.Join(parts, "\\")
	if result == "" {
		return "", nil
	}

	return "\\" + result, nil
}

// renderClauses serializes clauses while rejecting bytes that would break delimiters.
func renderClauses(clauses []filterClause) (string, error) {
	parts := make([]string, 0, len(clauses)*2)

	for _, clause := range clauses {
		if strings.IndexByte(string(clause.key), 0) >= 0 || strings.IndexByte(clause.value, 0) >= 0 {
			return "", ErrFilterKeyValueContainsNUL
		}

		if strings.ContainsRune(string(clause.key), '\\') || strings.ContainsRune(clause.value, '\\') {
			return "", ErrFilterContainsDelimiter
		}

		if clause.key == "" {
			return "", ErrEmptyFilterKey
		}

		parts = append(parts, string(clause.key), clause.value)
	}

	return strings.Join(parts, "\\"), nil
}
