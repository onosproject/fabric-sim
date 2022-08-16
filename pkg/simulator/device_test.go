// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"fmt"
	"github.com/onosproject/fabric-sim/pkg/simulator/config"
	"github.com/onosproject/fabric-sim/pkg/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestDeviceProcessGet tests operation of configuration retrieval
func TestDeviceProcessGet(t *testing.T) {
	rootNode := CreateSwitchConfig(8)
	ds := &DeviceSimulator{config: rootNode}

	n, err := ds.ProcessConfigGet(nil, []*gnmi.Path{utils.ToPath("interfaces/interface[name=4]")})
	assert.NoError(t, err)
	assert.Len(t, n, 1)
	assert.Len(t, n[0].Update, 20)
}

// TestDeviceProcessGet tests operation of configuration retrieval
func TestDeviceProcessSet(t *testing.T) {
	rootNode := CreateSwitchConfig(8)
	ds := &DeviceSimulator{config: rootNode}

	n, err := ds.ProcessConfigGet(nil, []*gnmi.Path{utils.ToPath("interfaces/interface[name=4]/config/enabled")})
	assert.NoError(t, err)
	assert.True(t, n[0].Update[0].Val.GetBoolVal())

	_, err = ds.ProcessConfigSet(nil,
		[]*gnmi.Update{
			{
				Path: utils.ToPath("interfaces/interface[name=4]/config/enabled"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_BoolVal{BoolVal: false}},
			},
		}, []*gnmi.Update{}, []*gnmi.Path{})
	assert.NoError(t, err)

	n, err = ds.ProcessConfigGet(nil, []*gnmi.Path{utils.ToPath("interfaces/interface[name=4]/config/enabled")})
	assert.NoError(t, err)
	assert.False(t, n[0].Update[0].Val.GetBoolVal())
}

// CreateSwitchConfig creates a test device configuration
func CreateSwitchConfig(portCount uint32) *config.Node {
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
	return config.NewSwitchConfig(ports)
}
