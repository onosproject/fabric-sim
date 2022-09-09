// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddresses(t *testing.T) {
	assert.Len(t, IP("1.2.3.4"), 4)
	assert.Equal(t, IP("1.2.3.4"), []byte{0x1, 0x2, 0x3, 0x4})
	assert.Len(t, MAC("11:22:33:44:55:66"), 6)
	assert.Equal(t, MAC("11:22:33:44:55:66"), []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})

	assert.Equal(t, IPString([]byte{0x1, 0x2, 0x3, 0x4}), "1.2.3.4")
	assert.Equal(t, IPString([]byte{0x11}), "0.0.0.0")
	assert.Equal(t, MACString([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}), "11:22:33:44:55:66")
	assert.Equal(t, MACString([]byte{0x11, 0x22}), "00:00:00:00:00:00")
}

func TestARPRequestPacket(t *testing.T) {
	ip := []byte{10, 10, 10, 42}
	ourIP := []byte{10, 10, 10, 69}
	ourMAC := []byte{10, 11, 12, 13, 14, 15}
	b, err := ARPRequestPacket(ip, ourMAC, ourIP)
	assert.NoError(t, err)
	assert.Len(t, b, 60)

	packet := gopacket.NewPacket(b, layers.LayerTypeEthernet, gopacket.Default)
	arpLayer := packet.Layer(layers.LayerTypeARP)
	assert.NotNil(t, arpLayer)
}

func TestControllerLLDPPacket(t *testing.T) {
	b, err := ControllerLLDPPacket("switch1", 123)
	assert.NoError(t, err)
	assert.Len(t, b, 60)

	packet := gopacket.NewPacket(b, layers.LayerTypeEthernet, gopacket.Default)
	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	assert.NotNil(t, lldpLayer)

	lldp := lldpLayer.(*layers.LinkLayerDiscovery)
	assert.NotNil(t, lldp)

	assert.Equal(t, "switch1", string(lldp.ChassisID.ID))
	assert.Equal(t, "123", string(lldp.PortID.ID))
}
