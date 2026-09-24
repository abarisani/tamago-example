// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build virt_loong64

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"

	"github.com/usbarmory/tamago-example/shell"
	_ "github.com/usbarmory/tamago/board/qemu/virt"
	"github.com/usbarmory/tamago/soc/loongson/ls3a5000"
)

const boardName = "qemu-system-loongarch64 (virt)"

var NIC interface{}

func init() {
	Terminal = ls3a5000.UART0
}

func date(epoch int64) {
	ls3a5000.LA64.SetTime(epoch)
}

func uptime() (ns int64) {
	return ls3a5000.LA64.GetTime() - ls3a5000.LA64.TimerOffset
}

func infoCmd(_ *shell.Interface, _ []string) (string, error) {
	var res bytes.Buffer

	ramStart, ramEnd := runtime.MemRegion()
	name, freq := Target()
	features := ls3a5000.LA64.Features()

	fmt.Fprintf(&res, "Runtime ......: %s %s/%s thread %d\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, ls3a5000.LA64.ID())
	fmt.Fprintf(&res, "RAM ..........: %#08x-%#08x (%d MiB)\n", ramStart, ramEnd, (ramEnd-ramStart)/(1024*1024))
	fmt.Fprintf(&res, "Board ........: %s\n", boardName)
	fmt.Fprintf(&res, "SoC ..........: %s\n", name)
	fmt.Fprintf(&res, "Features .....: %+v\n", features)
	fmt.Fprintf(&res, "Frequency ....: %v MHz\n", freq/1e6)

	return res.String(), nil
}

func rebootCmd(_ *shell.Interface, _ []string) (_ string, err error) {
	return "", errors.New("unimplemented")
}

func cryptoTest() {
	spawn(btcTest)
	spawn(kemTest)
}

func storageTest() {
	return
}

func HasNetwork() (_ bool, eth bool) {
	return false, false
}

func Target() (name string, freq uint32) {
	return "ls3a5000", 100e6
}
