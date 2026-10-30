// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/msqp

package msqp_test

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/woozymasta/msqp"
)

func ExampleRequest() {
	addr := netip.MustParseAddrPort("192.0.2.10:27015")
	request := msqp.Request{
		Cursor: msqp.Cursor{Addr: addr},
		Filter: `\appid\221100\secure\1`,
		Region: 3,
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(requestJSON))

	var decodedRequest msqp.Request
	if err := json.Unmarshal(requestJSON, &decodedRequest); err != nil {
		panic(err)
	}
	fmt.Println(decodedRequest.Cursor.Addr, decodedRequest.Filter, decodedRequest.Region)

	response := msqp.Response{
		NextCursor: msqp.Cursor{Addr: addr},
		Servers:    []netip.AddrPort{addr},
		Done:       true,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(responseJSON))

	var decodedResponse msqp.Response
	if err := json.Unmarshal(responseJSON, &decodedResponse); err != nil {
		panic(err)
	}
	fmt.Println(decodedResponse.NextCursor.Addr, decodedResponse.Servers, decodedResponse.Done)

	// Output:
	// {"cursor":{"addr":"192.0.2.10:27015"},"filter":"\\appid\\221100\\secure\\1","region":3}
	// 192.0.2.10:27015 \appid\221100\secure\1 3
	// {"next_cursor":{"addr":"192.0.2.10:27015"},"servers":["192.0.2.10:27015"],"done":true}
	// 192.0.2.10:27015 [192.0.2.10:27015] true
}
