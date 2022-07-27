// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestLLDPSerialization is used as a playground to validate that LLDP library is being used properly.
func TestLLDPSerialization(t *testing.T) {
	// Create an LLDP packet
	lldp := layers.LinkLayerDiscovery{
		ChassisID: layers.LLDPChassisID{
			Subtype: layers.LLDPChassisIDSubTypeLocal,
			ID:      []byte("0"),
		},
		PortID: layers.LLDPPortID{
			Subtype: layers.LLDPPortIDSubtypeLocal,
			ID:      []byte("1"),
		},
		TTL:    0,
		Values: nil,
	}

	// Serialize it
	buffer := gopacket.NewSerializeBuffer()
	err := lldp.SerializeTo(buffer, gopacket.SerializeOptions{})
	assert.NoError(t, err)

	// No see if we can deserialize it
	packet := gopacket.NewPacket(buffer.Bytes(), layers.LayerTypeLinkLayerDiscovery, gopacket.Default)
	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	assert.NotNil(t, lldpLayer)

	t.Logf("lldp: %+v", lldpLayer)

	lldp2 := lldpLayer.(*layers.LinkLayerDiscovery)
	t.Logf("lldp: %+v", lldp2)
}
