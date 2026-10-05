// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build usbarmory

package snvs

import (
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/sha256"

	"filippo.io/keygen"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
	"github.com/usbarmory/tamago/soc/nxp/snvs"
)

const diversifierDev = "GoKeySNVSDeviceK"

func init() {
	if !imx6ul.Native || !imx6ul.SNVS.Available() {
		return
	}

	// Disable ARM debug operations
	imx6ul.Debug(false)

	imx6ul.SNVS.SetPolicy(
		snvs.SecurityPolicy{
			Clock:             true,
			Temperature:       true,
			Voltage:           true,
			SecurityViolation: true,
			HardFail:          true,
		},
	)
}

// DeviceKey derives a device key, uniquely and deterministically generated for
// this SoC for attestation purposes.
func DeviceKey() (deviceKey *ecdsa.PrivateKey, err error) {
	iv := make([]byte, aes.BlockSize)
	key, err := imx6ul.CAAM.DeriveKey([]byte(diversifierDev), iv, -1)

	if err != nil {
		return
	}

	salt := imx6ul.UniqueID()

	if key, err = hkdf.Key(sha256.New, key, salt[:], "", sha256.BlockSize); err != nil {
		return
	}

	return keygen.ECDSA(elliptic.P256(), key)
}
