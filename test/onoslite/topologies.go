// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package onoslite

import (
	"github.com/onosproject/fabric-sim/test/basic"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

// TestLiteONOSWithPlainMidFabric tests mid fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithPlainMidFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/plain_mid.yaml", 2+4, (3*2*4)*2, 4*20,
		func(device *simapi.Device) int { return 32 }, func(host *simapi.Host) int { return 1 }, 90*time.Second)
}

// TestLiteONOSWithPlainLargeFabric tests mid fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithPlainLargeFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/plain_large.yaml", 4+8, (3*4*8)*2, 8*50,
		func(device *simapi.Device) int { return 64 }, func(host *simapi.Host) int { return 1 }, 90*time.Second)
}

// TestLiteONOSWithPodFabric tests pod fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithPodFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/pod.yaml", 2+6+6*12, (4*2*6+6*2*12)*2, 6*12*(20+1),
		func(device *simapi.Device) int {
			if strings.HasPrefix(string(device.ID), "spine") {
				return 64
			} else if strings.HasPrefix(string(device.ID), "leaf") {
				return 32
			}
			return 20 + 1 + 2 // IPU
		},
		func(host *simapi.Host) int { return 1 }, 200*time.Second)
}

// TestLiteONOSWithPlainMaxFabric tests max fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithPlainMaxFabric(t *testing.T) {
	t.Skip("Requires longer discovery time...")
	RunLiteONOSWithTopology(t, "topologies/plain_max.yaml", 4+60, (4*60)*2, 60*15,
		func(device *simapi.Device) int {
			if strings.Contains(string(device.ID), "leaf") {
				return 32
			}
			return 64
		},
		func(host *simapi.Host) int { return 1 }, 200*time.Second)
}

// TestLiteONOSWithMaxTopo tests max topo with ONOS lite
func (s *TestSuite) TestLiteONOSWithMaxTopo(t *testing.T) {
	t.Skip("Requires longer discovery time...")
	RunLiteONOSWithTopology(t, "topologies/max.yaml", 3+97, (3*97)*2, 105*97,
		func(device *simapi.Device) int { return 128 },
		func(host *simapi.Host) int { return 1 }, 480*time.Second)
}

// TestLiteONOSWithAccessFabric tests access fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithAccessFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/access.yaml", 3+6, (3*3*6+3*2)*2, 3*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 }, 90*time.Second)
}

// TestLiteONOSWithFixedFabric tests fixed fabric with ONOS lite
func (s *TestSuite) TestLiteONOSWithFixedFabric(t *testing.T) {
	RunLiteONOSWithTopology(t, "topologies/fixed_fabric.yaml", 14, 2*136, 40,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 }, 90*time.Second)
}

// RunLiteONOSWithTopology loads simulator with the specified topology, creates lite ONOS controller
// and points it at the simulated topology validating that the network environment gets properly discovered.
func RunLiteONOSWithTopology(t *testing.T, topologyPath string, deviceCount int, linkCount int, hostCount int,
	portsPerDevice basic.DevicePortCount, nicsPerHost basic.HostNICCount, delay time.Duration) {
	devices, _, hosts := basic.LoadAndValidate(t, topologyPath, deviceCount, linkCount, hostCount, portsPerDevice, nicsPerHost)
	defer basic.CleanUp(t)

	onos := NewLiteONOS()

	pointers := extractDevicePointers(devices)
	err := onos.Start(pointers)
	assert.NoError(t, err)

	defer func() { _ = onos.Stop() }()

	time.Sleep(delay)

	t.Logf("Validating discovered topology...")
	t.Logf("Devices: %d; Links: %d; Hosts: %d", len(onos.Devices), len(onos.Links), len(onos.Hosts))

	// Did we discover all devices?
	assert.Equal(t, len(onos.Devices), deviceCount)

	// Do all devices have the right number of ports?
	for _, device := range onos.Devices {
		assert.Len(t, device.Ports, portsPerDevice(findSimDevice(device.ID, devices)))
	}

	// Did we discover all links?
	assert.Equal(t, len(onos.Links), linkCount)

	// Did we discover all hosts? Since these are NICs really, add up all the NICs per host.
	nicCount := 0
	for _, host := range hosts {
		nicCount = nicCount + nicsPerHost(host)
	}
	assert.Equal(t, len(onos.Hosts), nicCount)
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
