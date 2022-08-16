// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestDeviceConfig is used as a playground to validate the creation of device gNMI config.
func TestDeviceConfig(t *testing.T) {
	rootNode := CreateSwitchConfig(8)
	assert.NotNil(t, rootNode.Get("interfaces", nil))

	node := rootNode.GetPath("interfaces/interface[name=5]/state/id")
	assert.NotNil(t, node.Value().GetIntVal())
	assert.Equal(t, "id", node.Name())
	assert.Equal(t, uint64(1029), node.Value().GetUintVal())

	nodes := rootNode.FindAll("interfaces/interface[name=7]")
	assert.Len(t, nodes, 20)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state")
	assert.Len(t, nodes, 18)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state/counters")
	assert.Len(t, nodes, 14)

	nodes = rootNode.FindAll("interfaces/interface[name=7]/state/ifindex")
	assert.Len(t, nodes, 1)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state")
	assert.Len(t, nodes, 8*18)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state/ifindex")
	assert.Len(t, nodes, 8)

	nodes = rootNode.FindAll("interfaces/interface[name=...]/state/counters")
	assert.Len(t, nodes, 8*14)

	node = rootNode.GetPath("interfaces/interface[name=2]/state/counters")
	assert.NotNil(t, node)
	node = rootNode.DeletePath("interfaces/interface[name=2]/state/counters")
	assert.NotNil(t, node)
	node = rootNode.GetPath("interfaces/interface[name=2]/state/counters")
	assert.Nil(t, node)
}

// CreateSwitchConfig creates a test device configuration
func CreateSwitchConfig(portCount uint32) *Node {
	ports := make(map[simapi.PortID]*simapi.Port)
	for i := uint32(1); i <= portCount; i++ {
		id := simapi.PortID(fmt.Sprintf("%d", i))
		ports[id] = &simapi.Port{
			ID:             id,
			Name:           string(id),
			Number:         i,
			InternalNumber: 1024 + i,
			Speed:          "100GB",
		}
	}
	return NewSwitchConfig(ports)
}
