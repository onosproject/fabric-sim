// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

const agentPortOffset = 20000

// State to assist generating dev cloud fabric topology
type devCloudBuilder struct {
	agentPort int32
}

// GenerateDevCloudFabric generates topology YAML from the specified access fabric recipe
func GenerateDevCloudFabric(fabric *DevCloudFabric) *Topology {
	log.Infof("Generating DevCloud Fabric")

	topology := &Topology{}
	builder := &devCloudBuilder{}

	// First, generate super-spines
	createSuperSpines(fabric, builder, topology)

	// Next generate rack-pair fabric and connect it to super-spines
	for i := 0; i < fabric.RackPairs; i++ {
		createRackPairFabric(fabric, builder, topology)
	}
	return topology
}

// Create the prescribed number of super-spines
func createSuperSpines(fabric *DevCloudFabric, builder *devCloudBuilder, topology *Topology) {
	count := defaultCount(fabric.SuperSpines, 2)
	for i := 1; i <= count; i++ {
		createSuperSpine(i, fabric, builder, topology)
	}
}

func createSuperSpine(id int, fabric *DevCloudFabric, builder *devCloudBuilder, topology *Topology) {
	rackPairCount := defaultCount(fabric.RackPairs, 2)
	deviceID := fmt.Sprintf("sspine%d", id)
	sspine := Device{
		ID:        deviceID,
		Type:      "switch",
		AgentPort: builder.agentPort + agentPortOffset,
		Stopped:   false,
		Ports:     createPorts(uint32(16*rackPairCount), deviceID),
	}
	builder.agentPort++
	topology.Devices = append(topology.Devices, sspine)
}

func createPorts(portCount uint32, deviceID string) []Port {
	ports := make([]Port, 0, portCount)
	for i := uint32(1); i <= portCount; i++ {
		port := Port{
			Number:    i,
			SDNNumber: i + 1023,
			Speed:     "100Gbps",
		}
		ports = append(ports, port)
	}
	return ports
}

func createRackPairFabric(fabric *DevCloudFabric, builder *devCloudBuilder, topology *Topology) {

}
