// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package network

import (
	"net"

	// maintained set of TLS roots for any potential TLS client requests
	_ "golang.org/x/crypto/x509roots/fallback"
)

// This example starts TCP/IP networking on all available network
// interfaces (either USB, Ethernet or both), for simplicity each NIC
// is assigned the same IP address and its own gVisor stack.
//
// For more advanced use cases gVisor supports sharing a single stack across
// different NIC IDs and routing while this example simply clones interface
// configuration and stack.
const (
	MAC      = "1a:55:89:a2:19:42"
	Netmask  = "255.255.255.0"
	IP       = "10.1.7.200"
	Gateway  = "10.1.7.100"
	Resolver = "8.8.8.8:53"
)

func init() {
	net.SetDefaultNS([]string{Resolver})
}
