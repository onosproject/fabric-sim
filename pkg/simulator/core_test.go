// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"github.com/onosproject/fabric-sim/pkg/topo"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testAgent struct {
}

func (t testAgent) Start(simulation *Simulation, deviceSim *DeviceSimulator) error {
	return nil
}

func (t testAgent) Stop(mode simapi.StopMode) error {
	return nil
}

func TestSimulationBasics(t *testing.T) {
	core := NewSimulation()
	topology := &topo.Topology{}
	err := topo.LoadTopologyFile("../../topologies/custom.yaml", topology)
	assert.NoError(t, err)

	// Fiddle with devices
	for _, dd := range topology.Devices {
		device := topo.ConstructDevice(dd)
		dsim, err := core.AddDeviceSimulator(device, &testAgent{})
		assert.NoError(t, err)
		assert.Equal(t, device.ID, dsim.Device.ID)
		dsim, err = core.GetDeviceSimulator(device.ID)
		assert.NoError(t, err)
		assert.Equal(t, device.ID, dsim.Device.ID)
	}
	devices := core.GetDeviceSimulators()
	assert.Len(t, devices, len(topology.Devices))

	dsim, err := core.GetDeviceSimulatorForPort(simapi.PortID("spine1/2"))
	assert.NoError(t, err)
	assert.Equal(t, simapi.DeviceID("spine1"), dsim.Device.ID)

	_, err = core.GetDeviceSimulatorForPort(simapi.PortID("spineX/2"))
	assert.Error(t, err)

	// Fiddle with links
	for _, ld := range topology.Links {
		link := topo.ConstructLink(ld)
		lsim, err := core.AddLinkSimulator(link)
		assert.NoError(t, err)
		assert.Equal(t, link.ID, lsim.Link.ID)
		lsim, err = core.GetLinkSimulator(link.ID)
		assert.NoError(t, err)
		assert.Equal(t, link.ID, lsim.Link.ID)
	}
	links := core.GetLinkSimulators()
	assert.Len(t, links, len(topology.Links))

	lfp1 := core.GetLinkFromPort(dsim.Device.Ports[0].ID)
	assert.NotNil(t, lfp1)

	lfp2 := core.GetLinkFromPort(simapi.PortID("spine1/5"))
	assert.Nil(t, lfp2)

	// Fiddle with hosts
	for _, hd := range topology.Hosts {
		host := topo.ConstructHost(hd)
		hsim, err := core.AddHostSimulator(host)
		assert.NoError(t, err)
		assert.Equal(t, host.ID, hsim.Host.ID)
		hsim, err = core.GetHostSimulator(host.ID)
		assert.NoError(t, err)
		assert.Equal(t, host.ID, hsim.Host.ID)
	}
	hosts := core.GetHostSimulators()
	assert.Len(t, hosts, len(topology.Hosts))

	rh1 := core.GetRandomHostSimulator(nil)
	assert.NotNil(t, rh1)
	rh2 := core.GetRandomHostSimulator(rh1)
	assert.NotNil(t, rh2)

	// Execute removals
	err = core.RemoveHostSimulator(hosts[0].Host.ID)
	assert.NoError(t, err)
	hosts = core.GetHostSimulators()
	assert.Len(t, hosts, len(topology.Hosts)-1)

	err = core.RemoveLinkSimulator(links[0].Link.ID)
	assert.NoError(t, err)
	links = core.GetLinkSimulators()
	assert.Len(t, links, len(topology.Links)-1)

	err = core.RemoveDeviceSimulator(devices[0].Device.ID)
	assert.NoError(t, err)
	devices = core.GetDeviceSimulators()
	assert.Len(t, devices, len(topology.Devices)-1)
}
