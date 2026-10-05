// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build usbarmory

package snvs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/sha256"

	"filippo.io/keygen"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

const diversifierDev = "ArmoredWitTamaGoExampleDeviceKey"

// DeviceKey derives a device key, uniquely and deterministically generated for
// this SoC for attestation purposes.
func DeviceKey() (deviceKey *ecdsa.PrivateKey, err error) {
	key := make([]byte, sha256.Size)

	if err = imx6ul.CAAM.DeriveKey([]byte(diversifierDev), key); err != nil {
		return
	}

	salt := imx6ul.UniqueID()

	if key, err = hkdf.Key(sha256.New, key, salt[:], "", sha256.BlockSize); err != nil {
		return
	}

	return keygen.ECDSA(elliptic.P256(), key)
}
