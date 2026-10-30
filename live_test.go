//go:build msqp_live

// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/woozymasta/msqp"
)

// liveMastersFile contains only endpoints explicitly enabled for live tests.
const liveMastersFile = "testdata/masters.txt"

func TestLiveMasters(t *testing.T) {
	addresses, err := readLiveMasters(liveMastersFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(addresses) == 0 {
		t.Fatal("live master list is empty")
	}

	for _, address := range addresses {
		address := address
		t.Run(address, func(t *testing.T) {
			testLiveMaster(t, address)
		})
	}
}

// testLiveMaster checks one endpoint without relying on a fixed server count.
func testLiveMaster(t *testing.T, address string) {
	t.Helper()

	client, err := msqp.NewClient(address, msqp.WithIOTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("NewClient(%q) error = %v", address, err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	page, err := client.Query(ctx, msqp.Request{})
	if err != nil {
		t.Fatalf("Query(%q) error = %v", address, err)
	}
	if len(page.Servers) == 0 {
		t.Fatalf("Query(%q) returned no servers", address)
	}
	for index, server := range page.Servers {
		if !server.IsValid() || !server.Addr().Is4() || server.Port() == 0 {
			t.Errorf("server[%d] = %s is not a valid IPv4 endpoint", index, server)
		}
	}
}

// readLiveMasters loads and validates the non-commented endpoints in a fixture.
func readLiveMasters(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open live master list: %w", err)
	}
	defer func() { _ = file.Close() }()

	var addresses []string
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := validateMasterAddress(line); err != nil {
			return nil, fmt.Errorf("live master list line %d: %w", lineNumber, err)
		}
		addresses = append(addresses, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read live master list: %w", err)
	}

	return addresses, nil
}

// validateMasterAddress checks the host:port syntax and UDP port range.
func validateMasterAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return fmt.Errorf("invalid UDP endpoint %q", address)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("invalid UDP port in %q", address)
	}

	return nil
}
