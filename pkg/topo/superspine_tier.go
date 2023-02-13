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
	}

	// Generate set of unidirectional egress links to pod spines a set of pod fabric topologies addendums
	// unidirectional egress links to the superspines
	for pod := 1; pod <= fabric.Pods; pod++ {
		podTopology := &Topology{}
		podID := fmt.Sprintf("pod%02d", pod)
		podDomain := fmt.Sprintf(fabric.PodsDomain, pod)
		for spine := 1; spine <= fabric.PodSpines; spine++ {
			spineID := fmt.Sprintf("spine%d", spine)
			builder.maxPort[spineID] = fabric.SuperSpinePortCount
			for superspine := 1; superspine <= fabric.SuperSpines; superspine++ {
				superspineID := fmt.Sprintf("superspine%d", superspine)
				createExternalLinkTrunk(superspineID, fabric.SuperSpinesDomain, spineID, podDomain, 2, builder, topology, podTopology)
			}
		}

		podTopologyPath := strings.Replace(path, ".yaml", fmt.Sprintf("-%s.yaml", podID), 1)
		if err := saveTopologyFile(podTopology, podTopologyPath); err != nil {
			return nil, err
		}
	}

	return topology, nil
}

// Create a trunk of specified number of unidirectional links between two devices in two different domains and topologies
func createExternalLinkTrunk(d1 string, domain1 string, d2 string, domain2 string, count int, builder *Builder,
	t1 *Topology, t2 *Topology) {
	for i := 0; i < count; i++ {
		p1 := builder.NextDevicePortID(d1)
		p2 := builder.NextDevicePortID(d2)
		link := Link{
			SrcPortID:      p1,
			TgtPortID:      fmt.Sprintf("%s:%s", domain2, p2),
			Unidirectional: true,
		}
		t1.Links = append(t1.Links, link)

		link = Link{
			SrcPortID:      p2,
			TgtPortID:      fmt.Sprintf("%s:%s", domain1, p1),
			Unidirectional: true,
		}
		t2.Links = append(t2.Links, link)
	}
}
