// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

const (
	spineY      = 0
	spinesGap   = 400
	leafY       = 200
	leafGap     = 400
	hostsY      = 450
	hostsGap    = 70
	hostsPerRow = 10
	hostRowGap  = 80
)

// GenerateAccessFabric generates topology YAML from the specified access fabric recipe
func GenerateAccessFabric(fabric *AccessFabric) *Topology {
	log.Infof("Generating %dx%d Access Fabric", fabric.Spines, fabric.LeafPairs*2)

	topology := &Topology{}
	builder := NewBuilder()

	// First, create the spines
	for spine := 1; spine <= fabric.Spines; spine++ {
		createSwitch(fmt.Sprintf("spine%d", spine), fabric.SpinePortCount, builder, topology,
			pos(coord(spine, fabric.Spines, spinesGap, 0), spineY))
	}

	// Then, create the leaves and connect them to the spines
	for pair := 1; pair <= fabric.LeafPairs; pair++ {
		sw1 := createSwitch(fmt.Sprintf("leaf%d%d", pair, 1), fabric.LeafPortCount, builder, topology,
			pos(coord(2*pair-1, 2*fabric.LeafPairs, leafGap, 0), leafY))
		sw2 := createSwitch(fmt.Sprintf("leaf%d%d", pair, 2), fabric.LeafPortCount, builder, topology,
			pos(coord(2*pair, 2*fabric.LeafPairs, leafGap, 0), leafY))

		leaf1 := sw1.ID
		leaf2 := sw2.ID

		// Attach the leaves to the spines
		for spine := 1; spine <= fabric.Spines; spine++ {
			spine := fmt.Sprintf("spine%d", spine)
			createLinkTrunk(spine, leaf1, fabric.SpineTrunk, builder, topology)
			createLinkTrunk(spine, leaf2, fabric.SpineTrunk, builder, topology)
		}

		// Pair the leaves
		createLinkTrunk(leaf1, leaf2, fabric.PairTrunk, builder, topology)

		// Latch the min ports for host attachment
		builder.minPort[sw1.ID] = builder.nextPort[sw1.ID]
		builder.minPort[sw2.ID] = builder.nextPort[sw2.ID]

		// Finally, create the hosts and attach them to the leaf pairs
		createRackHosts(pair, leaf1, leaf2, fabric.HostsPerPair, false, builder, topology, coord(2*pair, 2*fabric.LeafPairs, leafGap, -leafGap/2), hostsPerRow)
	}

	return topology
}
