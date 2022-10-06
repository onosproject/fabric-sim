// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"fmt"
	"github.com/onosproject/fabric-sim/pkg/simulator/config"
	"github.com/onosproject/fabric-sim/pkg/topo"
	"github.com/onosproject/fabric-sim/pkg/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-api/go/onos/stratum"
	"github.com/openconfig/gnmi/proto/gnmi"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/code"
	"testing"
)

func TestNewDeviceSimulator(t *testing.T) {
	topology := &topo.Topology{}
	err := topo.LoadTopologyFile("../../topologies/custom.yaml", topology)
	assert.NoError(t, err)
	sim := NewDeviceSimulator(topo.ConstructDevice(topology.Devices[0]), nil, nil)
	assert.Equal(t, simapi.DeviceID(topology.Devices[0].ID), sim.Device.ID)
}

type dummyStreamResponder struct {
}

func (d dummyStreamResponder) GetConnection() *simapi.Connection {
	return &simapi.Connection{}
}

func (d dummyStreamResponder) GetRoleConfig() *stratum.P4RoleConfig {
	panic("implement me")
}

func (d dummyStreamResponder) LatchMastershipArbitration(arbitration *p4api.MasterArbitrationUpdate) *p4api.MasterArbitrationUpdate {
	panic("implement me")
}

func (d dummyStreamResponder) SendMastershipArbitration(role *p4api.Role, masterElectionID *p4api.Uint128, failCode code.Code) {
	panic("implement me")
}

func (d dummyStreamResponder) Send(response *p4api.StreamMessageResponse) {
	panic("implement me")
}

func (d dummyStreamResponder) IsMaster(role *p4api.Role, masterElectionID *p4api.Uint128) bool {
	panic("implement me")
}

func TestAddRemoveStreamResponder(t *testing.T) {
	ds := &DeviceSimulator{Device: &simapi.Device{}}
	r1 := &dummyStreamResponder{}
	r2 := &dummyStreamResponder{}
	ds.AddStreamResponder(r1)
	assert.Len(t, ds.streamResponders, 1)
	ds.AddStreamResponder(r2)
	assert.Len(t, ds.streamResponders, 2)
	ds.RemoveStreamResponder(r2)
	assert.Len(t, ds.streamResponders, 1)
	ds.RemoveStreamResponder(r1)
	assert.Len(t, ds.streamResponders, 0)
}

type dummySubscribeResponder struct {
}

func (d dummySubscribeResponder) GetConnection() *simapi.Connection {
	return &simapi.Connection{}
}

func (d dummySubscribeResponder) Send(response *gnmi.SubscribeResponse) {
	panic("implement me")
}

func TestAddRemoveSubscribeResponder(t *testing.T) {
	ds := &DeviceSimulator{Device: &simapi.Device{}}
	r1 := &dummySubscribeResponder{}
	r2 := &dummySubscribeResponder{}
	ds.AddSubscribeResponder(r1)
	assert.Len(t, ds.subscribeResponders, 1)
	ds.AddSubscribeResponder(r2)
	assert.Len(t, ds.subscribeResponders, 2)
	ds.RemoveSubscribeResponder(r2)
	assert.Len(t, ds.subscribeResponders, 1)
	ds.RemoveSubscribeResponder(r1)
	assert.Len(t, ds.subscribeResponders, 0)
}

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
			Enabled:        true,
		}
	}
	return config.NewSwitchConfig(ports)
}
