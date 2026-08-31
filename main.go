// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/usbarmory/tamago-example/cmd"
	"github.com/usbarmory/tamago-example/internal/semihosting"
	"github.com/usbarmory/tamago-example/network"
	"github.com/usbarmory/tamago-example/shell"
)

func main() {
	log.SetFlags(0)

	logFile, _ := os.OpenFile("/tamago-example.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	name, _ := cmd.Target()

	network.Banner += fmt.Sprintf(" • %s", name)

	newConsole := func() *shell.Interface {
		return &shell.Interface{
			Banner:     network.Banner,
			ReadWriter: cmd.Terminal,
		}
	}

	if hasUSB, hasEth := cmd.HasNetwork(); hasUSB || hasEth {
		if err := network.Init(newConsole, hasUSB, hasEth, &cmd.NIC); err != nil {
			log.Print(err)
		}
	} else {
		console := newConsole()
		console.Start(true)
	}

	semihosting.Exit()
}
