// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGenerateNetcfg(t *testing.T) {
	err := GenerateNetcfg("../../topologies/plain.yaml", "/tmp/plain_netcfg.json", "stratum-driver", "stratum-pipeconf", []int{1})
	defer os.Remove("/tmp/plain_netcfg.json")
	assert.NoError(t, err)
}

func TestIsLeaf(t *testing.T) {
	assert.False(t, isLeaf("spine1"))
	assert.True(t, isLeaf("leaf1"))
	assert.True(t, isLeaf("leaf2"))
	assert.True(t, isLeaf("leaf"))
}

func TestGetIndex(t *testing.T) {
	assert.Equal(t, 1, getIndex("spine1"))
	assert.Equal(t, 3, getIndex("leaf3"))
	assert.Equal(t, 17, getIndex("leaf17"))
	assert.Equal(t, 1, getIndex("leaf"))
}
