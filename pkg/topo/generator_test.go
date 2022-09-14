// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadTopologyFile(t *testing.T) {
	topo := &Topology{}
	err := LoadTopologyFile("../../topologies/plain.yaml", topo)
	assert.NoError(t, err)
	assert.Len(t, topo.Devices, 3+6)
	assert.Len(t, topo.Links, 3*3*6)
	assert.Len(t, topo.Hosts, 6*20)
}

func TestGeneratePlainFabric(t *testing.T) {
	topo := GeneratePlainFabric(&PlainFabric{
		Spines:         2,
		SpinePortCount: 32,
		Leaves:         4,
		LeafPortCount:  32,
		SpineTrunk:     3,
		HostsPerLeaf:   20,
	})
	assert.Len(t, topo.Devices, 2+4)
	assert.Len(t, topo.Links, 3*2*4)
	assert.Len(t, topo.Hosts, 4*20)

	testFromRecipe(t, "access", `plain_fabric:
  spines: 2
  spine_port_count: 32
  leaves: 4
  leaf_port_count: 32
  spine_trunk: 3
  hosts_per_leaf: 10`)
}

func TestGenerateAccessFabric(t *testing.T) {
	topo := GenerateAccessFabric(&AccessFabric{
		Spines:         2,
		SpinePortCount: 32,
		LeafPairs:      2,
		LeafPortCount:  32,
		SpineTrunk:     3,
		PairTrunk:      2,
		HostsPerPair:   20,
	})
	assert.Len(t, topo.Devices, 2+2*2)
	assert.Len(t, topo.Links, 2*2+3*2*4)
	assert.Len(t, topo.Hosts, 2*20)

	testFromRecipe(t, "access", `access_fabric:
  spines: 2
  spine_port_count: 32
  leaf_pairs: 2
  leaf_port_count: 32
  spine_trunk: 3
  pair_trunk: 2
  hosts_per_pair: 20
`)
}

func TestGenerateSuperspineFabric(t *testing.T) {
	topo := GenerateSuperSpineFabric(&SuperSpineFabric{})
	assert.Len(t, topo.Devices, 2+4+4*2)
	assert.Len(t, topo.Links, 2*(2*2+8*4+4*8))
	assert.Len(t, topo.Hosts, 2*2*10)

	testFromRecipe(t, "superspine", `superspine_fabric:
  none: false
`)
}

func testFromRecipe(t *testing.T, name string, recipe string) {
	recipeFile := fmt.Sprintf("/tmp/%s_recipe.yaml", name)
	topoFile := fmt.Sprintf("/tmp/%s.yaml", name)
	defer os.Remove(recipeFile)
	defer os.Remove(topoFile)

	err := ioutil.WriteFile(recipeFile, []byte(recipe), 0644)
	assert.NoError(t, err)
	err = GenerateTopology(recipeFile, topoFile)
	assert.NoError(t, err)
}
