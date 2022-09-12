// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package onoslite

import (
	"github.com/onosproject/fabric-sim/test/basic"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestLiteONOSWithPlainFabric tests superspine fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithPlainFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/plain.yaml", 6, 48, 80,
		func(device *simapi.Device) int { return 32 }, func(host *simapi.Host) int { return 1 })
}

// TestLiteONOSWithAccessFabric tests superspine fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithAccessFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/access.yaml", 3+6, (3*3*6+3*2)*2, 3*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
}

// TestLiteONOSWithSuperspineFabric tests superspine fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithSuperspineFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/superspine.yaml", 14, 2*136, 40,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
}

// RunLiteONOSWithTopology loads simulator with the specified topology, creates lite ONOS controller
// and points it at the simulated topology validating that the network environment gets properly discovered.
func RunLiteONOSWithTopology(t *testing.T, topologyPath string, deviceCount int, linkCount int, hostCount int,
	portsPerDevice basic.DevicePortCount, nicsPerHost basic.HostNICCount) {
	devices, _, hosts := basic.LoadAndValidate(t, topologyPath, deviceCount, linkCount, hostCount, portsPerDevice, nicsPerHost)
	defer basic.CleanUp(t)

	onos := NewLiteONOS()

	pointers := extractDevicePointers(devices)
	err := onos.Start(pointers)
	assert.NoError(t, err)

	defer func() { _ = onos.Stop() }()

	time.Sleep(1 * time.Minute)

	t.Logf("Validating discovered topology...")

	// Did we discover all devices?
	assert.Len(t, onos.Devices, deviceCount)

	// Do all devices have the right number of ports?
	for _, device := range onos.Devices {
		assert.Len(t, device.Ports, portsPerDevice(findSimDevice(device.ID, devices)))
	}

	// Did we discover all links?
	assert.Len(t, onos.Links, linkCount)

	// Did we discover all hosts? Since these are NICs really, add up all the NICs per host.
	nicCount := 0
	for _, host := range hosts {
		nicCount = nicCount + nicsPerHost(host)
	}
	assert.Len(t, onos.Hosts, nicCount)
}

func extractDevicePointers(devices []*simapi.Device) []*DevicePointer {
	pointers := make([]*DevicePointer, 0, len(devices))
	for _, device := range devices {
		pointers = append(pointers, &DevicePointer{ID: string(device.ID), ChassisID: device.ChassisID, ControlPort: device.ControlPort})
	}
	return pointers
}

func findSimDevice(id string, devices []*simapi.Device) *simapi.Device {
	for _, device := range devices {
		if id == string(device.ID) {
			return device
		}
	}
	return nil
}
