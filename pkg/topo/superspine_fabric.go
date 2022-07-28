// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

const agentPortOffset = 20000

// State to assist generating super-spine fabric topology
type superspineBuilder struct {
	agentPort int32
	nextPort  map[string]int
}

// GenerateSuperSpineFabric generates topology YAML from the specified super-spine fabric recipe
func GenerateSuperSpineFabric(fabric *SuperSpineFabric) *Topology {
	log.Infof("Generating Super-Spine Fabric")

	topology := &Topology{}
	builder := &superspineBuilder{
		nextPort: make(map[string]int),
	}

	// First, create 2 super-spines
	createSwitch("sspine1", 32, builder, topology)
	createSwitch("sspine2", 32, builder, topology)

	// Next generate two rack-pair fabrics and connect it to super-spines
	createRackPairFabric(1, fabric, builder, topology)
	createRackPairFabric(3, fabric, builder, topology)
	return topology
}

func createSwitch(deviceID string, portCount int, builder *superspineBuilder, topology *Topology) {
	device := Device{
		ID:        deviceID,
		Type:      "switch",
		AgentPort: builder.agentPort + agentPortOffset,
		Stopped:   false,
		Ports:     createPorts(portCount, deviceID),
	}
	topology.Devices = append(topology.Devices, device)
	builder.agentPort++
}

func createPorts(portCount int, deviceID string) []Port {
	ports := make([]Port, 0, portCount)
	for i := uint32(1); i <= uint32(portCount); i++ {
		port := Port{
			Number:    i,
			SDNNumber: i + 1023,
			Speed:     "100Gbps",
		}
		ports = append(ports, port)
	}
	return ports
}

func createRackPairFabric(rackID int, fabric *SuperSpineFabric, builder *superspineBuilder, topology *Topology) {
	// First, create 2 spines
	spine1 := fmt.Sprintf("spine%d", rackID)
	spine2 := fmt.Sprintf("spine%d", rackID+1)
	createSwitch(spine1, 32, builder, topology)
	createSwitch(spine2, 32, builder, topology)

	// Connect the spines to super-spines
	createLinkTrunk(spine1, "sspine1", 8, builder, topology)
	createLinkTrunk(spine1, "sspine2", 8, builder, topology)
	createLinkTrunk(spine2, "sspine1", 8, builder, topology)
	createLinkTrunk(spine2, "sspine2", 8, builder, topology)

	// Next, create 2 sets of paired leaves
	leaf11 := fmt.Sprintf("leaf%d1", rackID)
	leaf12 := fmt.Sprintf("leaf%d2", rackID)
	leaf21 := fmt.Sprintf("leaf%d1", rackID+1)
	leaf22 := fmt.Sprintf("leaf%d2", rackID+1)
	createSwitch(leaf11, 32, builder, topology)
	createSwitch(leaf12, 32, builder, topology)
	createSwitch(leaf21, 32, builder, topology)
	createSwitch(leaf22, 32, builder, topology)

	// Connect the leaves to the spines
	createLinkTrunk(leaf11, spine1, 8, builder, topology)
	createLinkTrunk(leaf11, spine2, 8, builder, topology)
	createLinkTrunk(leaf12, spine1, 8, builder, topology)
	createLinkTrunk(leaf12, spine2, 8, builder, topology)
	createLinkTrunk(leaf21, spine1, 8, builder, topology)
	createLinkTrunk(leaf21, spine2, 8, builder, topology)
	createLinkTrunk(leaf22, spine1, 8, builder, topology)
	createLinkTrunk(leaf22, spine2, 8, builder, topology)

	// Now actually pair the leaves
	createLinkTrunk(leaf11, leaf12, 2, builder, topology)
	createLinkTrunk(leaf21, leaf22, 2, builder, topology)

	// Finally, create hosts with dual interfaces to the paired leaves
	createRackHosts(rackID, leaf11, leaf12, 10, builder, topology)
	createRackHosts(rackID+1, leaf21, leaf22, 10, builder, topology)
}

func createLinkTrunk(src string, tgt string, count int, builder *superspineBuilder, topology *Topology) {
	for i := 0; i < count; i++ {
		link := Link{
			SrcPortID:      nextDevicePortID(src, builder),
			TgtPortID:      nextDevicePortID(tgt, builder),
			Unidirectional: false,
		}
		topology.Links = append(topology.Links, link)
	}
}

func createRackHosts(rackID int, leaf1 string, leaf2 string, count int, builder *superspineBuilder, topology *Topology) {
	for i := 1; i <= count; i++ {
		createRackHost(rackID, i, leaf1, leaf2, builder, topology)
	}
}

func createRackHost(rackID int, hostID int, leaf1 string, leaf2 string, builder *superspineBuilder, topology *Topology) {
	nic1 := NIC{
		Mac:  mac(rackID, hostID, 1),
		IPv4: ipv4(rackID, hostID, 1),
		IPV6: ipv6(rackID, hostID, 1),
		Port: nextDevicePortID(leaf1, builder),
	}
	nic2 := NIC{
		Mac:  mac(rackID, hostID, 2),
		IPv4: ipv4(rackID, hostID, 2),
		IPV6: ipv6(rackID, hostID, 2),
		Port: nextDevicePortID(leaf2, builder),
	}
	host := Host{
		ID:   fmt.Sprintf("host%d%d", rackID, hostID),
		NICs: []NIC{nic1, nic2},
	}
	topology.Hosts = append(topology.Hosts, host)
}

func mac(rackID int, hostID int, leafID int) string {
	return fmt.Sprintf("00:ca:fe:%02d:%02d:%02d", rackID, leafID, hostID)
}

func ipv4(rackID int, hostID int, leafID int) string {
	return fmt.Sprintf("10.10.%d%d.%d", rackID, leafID, hostID)
}

func ipv6(rackID int, hostID int, leafID int) string {
	return fmt.Sprintf("2001:dead:beef:baad:cafe:%d:%d:%d", rackID, leafID, hostID)
}

func nextDevicePortID(deviceID string, builder *superspineBuilder) string {
	portNumber, ok := builder.nextPort[deviceID]
	if !ok {
		portNumber = 1
	}
	portID := fmt.Sprintf("%s/%d", deviceID, portNumber)
	builder.nextPort[deviceID] = portNumber + 1
	return portID
}
