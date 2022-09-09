// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

// IP  returns the given IP address as bytes
func IP(addr string) []byte {
	b := net.ParseIP(addr)
	if len(b) == 16 {
		return b[12:]
	}
	return b
}

// IPString returns a string representation of an IP address
func IPString(addr []byte) string {
	if len(addr) == 4 {
		return fmt.Sprintf("%d.%d.%d.%d", addr[0], addr[1], addr[2], addr[3])
	}
	return "0.0.0.0"
}

// MAC returns the given MAC address as bytes
func MAC(addr string) []byte {
	b, _ := net.ParseMAC(addr)
	return b
}

// MACString returns a string representation of a MAC address
func MACString(addr []byte) string {
	if len(addr) == 6 {
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", addr[0], addr[1], addr[2], addr[3], addr[4], addr[5])
	}
	return "00:00:00:00:00:00"
}

// ARPRequestPacket returns packet bytes with an ARP request for the specified IP address
func ARPRequestPacket(theirIP []byte, ourMAC []byte, ourIP []byte) ([]byte, error) {
	eth := &layers.Ethernet{
		SrcMAC:       ourMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   ourMAC,
		SourceProtAddress: ourIP,
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    theirIP,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buf, opts, eth, arp)
	return buf.Bytes(), err
}

// ControllerLLDPPacket returns packet bytes for an ONOS link discovery packet
func ControllerLLDPPacket(chassisID string, egressPort uint32) ([]byte, error) {
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x60, 0x08, 0x69, 0x97, 0xef}, // use what SONiC uses
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeLinkLayerDiscovery,
	}

	lldp := &layers.LinkLayerDiscovery{
		BaseLayer: layers.BaseLayer{},
		ChassisID: layers.LLDPChassisID{
			Subtype: layers.LLDPChassisIDSubTypeLocal,
			ID:      []byte(chassisID),
		},
		// Note that this is not really used; instead the egress port number must be encoded as controller meta-data
		PortID: layers.LLDPPortID{
			Subtype: layers.LLDPPortIDSubtypeLocal,
			ID:      []byte(fmt.Sprintf("%d", egressPort)),
		},
		TTL:    0,
		Values: nil,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buf, opts, eth, lldp)
	return buf.Bytes(), err
}
