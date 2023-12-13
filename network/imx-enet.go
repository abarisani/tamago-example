// https://github.com/usbarmory/tamago-example
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build mx6ullevk || usbarmory
// +build mx6ullevk usbarmory

package network

import (
	"log"
	"net"
	"os"

	imxenet "github.com/usbarmory/imx-enet"
	"github.com/usbarmory/tamago/soc/nxp/enet"
	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

const (
	Netmask = "255.255.255.0"
	Gateway = "10.0.0.2"
)

func handleEthernetInterrupt(eth *enet.ENET) {
	for buf := eth.Rx(); buf != nil; buf = eth.Rx() {
		eth.RxHandler(buf)
		eth.ClearInterrupt(enet.IRQ_RXF)
	}
}

func StartEth(console consoleHandler, journalFile *os.File) (eth *enet.ENET) {
	eth = imx6ul.ENET2

	if !imx6ul.Native {
		eth = imx6ul.ENET1
	}

	iface, err := imxenet.Init(eth, IP, Netmask, MAC, Gateway, 1)

	if err != nil {
		log.Fatalf("could not initialize Ethernet networking, %v", err)
	}

	iface.EnableICMP()

	if console != nil {
		listenerSSH, err := iface.ListenerTCP4(22)

		if err != nil {
			log.Fatalf("could not initialize SSH listener, %v", err)
		}

		// wait for server to start before responding to USB requests
		StartSSHServer(listenerSSH, console)
	}

	listenerHTTP, err := iface.ListenerTCP4(80)

	if err != nil {
		log.Fatalf("could not initialize HTTP listener, %v", err)
	}

	listenerHTTPS, err := iface.ListenerTCP4(443)

	if err != nil {
		log.Fatalf("could not initialize HTTP listener, %v", err)
	}

	go StartWebServer(listenerHTTP, IP, 80, false)
	go StartWebServer(listenerHTTPS, IP, 443, true)

	journal = journalFile

	// hook interface into Go runtime
	net.SocketFunc = iface.Socket

	// This example illustrates IRQ handling, alternatively a poller can be
	// used with `eth.Start(true)`.

	eth.EnableInterrupt(enet.IRQ_RXF)
	eth.Start(false)

	return
}
