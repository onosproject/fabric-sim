// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

const (
	sspineY    = -200
	sspinesGap = 800
)

// GenerateSuperSpineFabric generates topology YAML from the specified super-spine fabric recipe
func GenerateSuperSpineFabric(fabric *SuperSpineFabric) *Topology {
	log.Infof("Generating Super-Spine Fabric")

	topology := &Topology{}
	builder := NewBuilder()

	// First, create 2 super-spines
	createSwitch("sspine1", 32, builder, topology,
		pos(coord(1, 2, sspinesGap, 0), sspineY))
	createSwitch("sspine2", 32, builder, topology,
		pos(coord(2, 2, sspinesGap, 0), sspineY))

	// Next generate two rack-pair fabrics and connect it to super-spines
	createRackPairFabric(1, fabric, builder, topology)
	createRackPairFabric(3, fabric, builder, topology)
	return topology
}

func createRackPairFabric(rackID int, fabric *SuperSpineFabric, builder *Builder, topology *Topology) {
	// TODO: implement grid layout positioning

	// First, create 2 spines
	spine1 := createSwitch(fmt.Sprintf("spine%d", rackID), 32, builder, topology,
		pos(coord(rackID, 4, 2*spinesGap, 0), spineY)).ID
	spine2 := createSwitch(fmt.Sprintf("spine%d", rackID+1), 32, builder, topology,
		pos(coord(rackID+1, 4, 2*spinesGap, 0), spineY)).ID

	// Connect the spines to super-spines
	createLinkTrunk(spine1, "sspine1", 8, builder, topology)
	createLinkTrunk(spine1, "sspine2", 8, builder, topology)
	createLinkTrunk(spine2, "sspine1", 8, builder, topology)
	createLinkTrunk(spine2, "sspine2", 8, builder, topology)

	// Next, create 2 sets of paired leaves
	leaf11 := createSwitch(fmt.Sprintf("leaf%d1", rackID), 32, builder, topology,
		pos(coord(2*rackID-1, 8, leafGap, 0), leafY)).ID
	leaf12 := createSwitch(fmt.Sprintf("leaf%d2", rackID), 32, builder, topology,
		pos(coord(2*rackID+0, 8, leafGap, 0), leafY)).ID
	leaf21 := createSwitch(fmt.Sprintf("leaf%d1", rackID+1), 32, builder, topology,
		pos(coord(2*rackID+1, 8, leafGap, 0), leafY)).ID
	leaf22 := createSwitch(fmt.Sprintf("leaf%d2", rackID+1), 32, builder, topology,
		pos(coord(2*rackID+2, 8, leafGap, 0), leafY)).ID

	// Connect the leaves to the spines
	createLinkTrunk(leaf11, spine1, 4, builder, topology)
	createLinkTrunk(leaf11, spine2, 4, builder, topology)
	createLinkTrunk(leaf12, spine1, 4, builder, topology)
	createLinkTrunk(leaf12, spine2, 4, builder, topology)
	createLinkTrunk(leaf21, spine1, 4, builder, topology)
	createLinkTrunk(leaf21, spine2, 4, builder, topology)
	createLinkTrunk(leaf22, spine1, 4, builder, topology)
	createLinkTrunk(leaf22, spine2, 4, builder, topology)

	// Now actually pair the leaves
	createLinkTrunk(leaf11, leaf12, 2, builder, topology)
	createLinkTrunk(leaf21, leaf22, 2, builder, topology)

	// Finally, create hosts with dual interfaces to the paired leaves
	createRackHosts(rackID, leaf11, leaf12, 10, builder, topology,
		coord(2*rackID-1, 4, leafGap, -3*leafGap/2), hostsPerRow)
	createRackHosts(rackID+1, leaf21, leaf22, 10, builder, topology,
		coord(2*rackID+0, 4, leafGap, -leafGap/2), hostsPerRow)
}
