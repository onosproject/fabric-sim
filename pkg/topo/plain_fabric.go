// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

// GeneratePlainFabric generates topology YAML from the specified spine-leaf fabric recipe
func GeneratePlainFabric(fabric *PlainFabric) *Topology {
	log.Infof("Generating %dx%d Leaf-Spine Fabric", fabric.Spines, fabric.Leaves)

	topology := &Topology{}
	builder := NewBuilder()

	// First, create the spines
	for spine := 1; spine <= fabric.Spines; spine++ {
		createSwitch(fmt.Sprintf("spine%d", spine), fabric.SpinePortCount, builder, topology,
			pos(coord(spine-1, fabric.Spines, spinesGap, 0), spineY))
	}

	// Then, create the leaves and connect them to the spines
	for i := 1; i <= fabric.Leaves; i++ {
		leaf := createSwitch(fmt.Sprintf("leaf%d", i), fabric.LeafPortCount, builder, topology,
			pos(coord(i-1, fabric.Leaves, leafGap, 0), leafY)).ID

		// Attach the leaves to the spines
		for spine := 1; spine <= fabric.Spines; spine++ {
			spine := fmt.Sprintf("spine%d", spine)
			createLinkTrunk(spine, leaf, fabric.SpineTrunk, builder, topology)
		}

		// Finally, create the hosts and attach them to the leaf pairs
		createRackHosts(i, leaf, "", fabric.HostsPerLeaf, builder, topology,
			coord(i-1, fabric.Leaves, leafGap, 0), hostsPerRow/2)
	}

	return topology
}
