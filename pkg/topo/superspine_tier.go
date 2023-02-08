// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"fmt"
	"strings"
)

// GenerateSuperSpineTier generates topology YAML from the specified super-spine fabric recipe
func GenerateSuperSpineTier(fabric *SuperSpineTier, path string) (*Topology, error) {
	log.Infof("Generating Super-Spine Fabric")

	topology := &Topology{}
	builder := NewBuilder()

	// First, create all super-spines
	for superspine := 1; superspine <= fabric.SuperSpines; superspine++ {
		superspineID := fmt.Sprintf("superspine%d", superspine)
		createSwitch(superspineID, fabric.SuperSpinePortCount, builder, topology,
			pos(coord(1, 2, sspinesGap, 0), sspineY))

		// Generate set of unidirectional egress links to pod spines
		for pod := 1; pod <= fabric.Pods; pod++ {
			podSimID := fmt.Sprintf("fabric-sim-pod%02d", pod)
			for spine := 1; spine <= fabric.PodSpines; spine++ {
				spineID := fmt.Sprintf("spine%d", spine)

				// We're assuming that ports 1 and 2 of pod spines are reserved for superspine links
				builder.minPort[spineID] = 1
				builder.maxPort[spineID] = 2
				createExternalLinkTrunk(superspineID, spineID, podSimID, 2, builder, topology)
			}
		}
	}

	// Lastly, generate and save pod fabric topologies addendums containing their
	// unidirectional egress links to the superspines
	for pod := 1; pod <= fabric.Pods; pod++ {
		topology := &Topology{}
		builder := NewBuilder()

		for superspine := 1; superspine <= fabric.SuperSpines; superspine++ {
			superspineID := fmt.Sprintf("superspine%d", superspine)

			for spine := 1; spine <= fabric.PodSpines; spine++ {
				spineID := fmt.Sprintf("spine%d", spine)

				// We're assuming that ports 1 and 2 of pod spines are reserved for superspine links
				builder.minPort[spineID] = 1
				builder.maxPort[spineID] = 2
				createExternalLinkTrunk(spineID, superspineID, "fabric-sim-superspines", 2, builder, topology)
			}
		}

		podTopologyPath := strings.Replace(path, ".yaml", fmt.Sprintf("-pod%02d.yaml", pod), 1)
		if err := saveTopologyFile(topology, podTopologyPath); err != nil {
			return nil, err
		}
	}

	return topology, nil
}

// Create a trunk of specified number of unidirectional links from a device port to a remote device port
func createExternalLinkTrunk(src string, tgt string, tgtPod string, count int, builder *Builder, topology *Topology) {
	for i := 0; i < count; i++ {
		link := Link{
			SrcPortID:      builder.NextDevicePortID(src),
			TgtPortID:      fmt.Sprintf("%s:%s", tgtPod, builder.NextDevicePortID(tgt)),
			Unidirectional: true,
		}
		topology.Links = append(topology.Links, link)
	}
}
