// https://github.com/f-secure-foundry/tamago-example
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"log"
	"net"

	"github.com/f-secure-foundry/tamago/imx6/usb"
	"github.com/f-secure-foundry/tamago/imx6/usb/ethernet"
)

func configureDevice(device *usb.Device) {
	// Supported Language Code Zero: English
	device.SetLanguageCodes([]uint16{0x0409})

	// device descriptor
	device.Descriptor = &usb.DeviceDescriptor{}
	device.Descriptor.SetDefaults()

	// p5, Table 1-1. Device Descriptor Using Class Codes for IAD,
	// USB Interface Association Descriptor Device Class Code and Use Model.
	device.Descriptor.DeviceClass = 0xef
	device.Descriptor.DeviceSubClass = 0x02
	device.Descriptor.DeviceProtocol = 0x01

	// http://pid.codes/1209/2702/
	device.Descriptor.VendorId = 0x1209
	device.Descriptor.ProductId = 0x2702

	device.Descriptor.Device = 0x0001

	iManufacturer, _ := device.AddString(`TamaGo`)
	device.Descriptor.Manufacturer = iManufacturer

	iProduct, _ := device.AddString(`RNDIS/Ethernet Gadget`)
	device.Descriptor.Product = iProduct

	iSerial, _ := device.AddString(`0.1`)
	device.Descriptor.SerialNumber = iSerial

	conf := &usb.ConfigurationDescriptor{}
	conf.SetDefaults()

	device.AddConfiguration(conf)

	// device qualifier
	device.Qualifier = &usb.DeviceQualifierDescriptor{}
	device.Qualifier.SetDefaults()
	device.Qualifier.NumConfigurations = uint8(len(device.Configurations))
}

func StartUSB() {
	device := &usb.Device{}
	configureDevice(device)

	hostAddress, err := net.ParseMAC(hostMAC)

	if err != nil {
		log.Fatal(err)
	}

	deviceAddress, err := net.ParseMAC(deviceMAC)

	if err != nil {
		log.Fatal(err)
	}

	// Start basic networking and SSH HTTP services.
	link := StartNetworking()

	// Configure Ethernet over USB endpoints
	// (ECM protocol, only supported on Linux hosts).
	eth := ethernet.NIC{
		Host:   hostAddress,
		Device: deviceAddress,
		Link:   link,
	}

	err = eth.Init(device, 0)

	if err != nil {
		log.Fatal(err)
	}

	usb.USB1.Init()
	usb.USB1.DeviceMode()
	usb.USB1.Reset()

	// never returns
	usb.USB1.Start(device)
}
