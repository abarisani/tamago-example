// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build mx6ullevk

package cmd

import (
	"github.com/usbarmory/tamago-example/shell"
	"github.com/usbarmory/tamago/board/nxp/mx6ullevk"
	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

const boardName = "MCIMX6ULL-EVK"

func init() {
	Terminal = mx6ullevk.UART1

	if !imx6ul.Native {
		return
	}

	mx6ullevk.I2C1.Init()
	I2C = append(I2C, mx6ullevk.I2C1)

	mx6ullevk.I2C2.Init()
	I2C = append(I2C, mx6ullevk.I2C2)

	MMC = append(MMC, mx6ullevk.SD1)
	MMC = append(MMC, mx6ullevk.SD2)
}

func rebootCmd(_ *shell.Interface, _ []string) (_ string, _ error) {
	mx6ullevk.Reset()
	return
}

func HasNetwork() (usb bool, eth bool) {
	return imx6ul.Native, true
}
