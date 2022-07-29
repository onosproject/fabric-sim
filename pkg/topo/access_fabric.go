// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import "fmt"

// GenerateAccessFabric generates topology YAML from the specified access fabric recipe
func GenerateAccessFabric(fabric *AccessFabric) *Topology {
	log.Infof("Generating Access Fabric")

	topology := &Topology{}
	builder := NewBuilder()

	// First, create the spines
	for spine := 1; spine <= fabric.Spines; spine++ {
		createSwitch(fmt.Sprintf("spine%d", spine), fabric.SpinePortCount, builder, topology)
	}

	// Then, create the leaves and connect them to the spines
	for pair := 1; pair <= fabric.LeafPairs; pair++ {
		leaf1 := createSwitch(fmt.Sprintf("leaf%d%d", pair, 1), fabric.LeafPortCount, builder, topology).ID
		leaf2 := createSwitch(fmt.Sprintf("leaf%d%d", pair, 2), fabric.LeafPortCount, builder, topology).ID

		// Attach the leaves to the spines
		for spine := 1; spine <= fabric.Spines; spine++ {
			spine := fmt.Sprintf("spine%d", spine)
			createLinkTrunk(spine, leaf1, fabric.SpineTrunk, builder, topology)
			createLinkTrunk(spine, leaf2, fabric.SpineTrunk, builder, topology)
		}

		// Pair the leaves
		createLinkTrunk(leaf1, leaf2, fabric.PairTrunk, builder, topology)

		// Finally, create the hosts and attach them to the leaf pairs
		createRackHosts(pair, leaf1, leaf2, fabric.HostsPerPair, builder, topology)
	}

	return topology
}
