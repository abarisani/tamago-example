// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !usbarmory

package snvs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

func DeviceKey() (deviceKey *ecdsa.PrivateKey, err error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
