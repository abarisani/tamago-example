// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !(cloud_hypervisor || firecracker || microvm || gcp || imx8mpevk || mx6ullevk || usbarmory)

package network

import (
	"fmt"
	"log"
	"runtime"
)

var Banner = fmt.Sprintf("%s/%s (%s)", runtime.GOOS, runtime.GOARCH, runtime.Version())

func Init(_ any, _ bool, _ bool, _ any) (_ any) {
	log.Fatal("unsupported")
	return
}
